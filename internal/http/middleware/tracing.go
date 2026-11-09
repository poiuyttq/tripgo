package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/poiuyttq/tripgo/internal/requesttrace"
)

func Tracing(provider trace.TracerProvider, propagator propagation.TextMapPropagator, method, route string) func(http.Handler) http.Handler {
	operation := fmt.Sprintf("%s %s", method, route)
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())
			spanContext := span.SpanContext()
			if spanContext.IsValid() {
				requesttrace.Set(r.Context(), spanContext.TraceID().String(), spanContext.SpanID().String())
			}
			span.SetAttributes(attribute.String("http.route", route))
			writer := newResponseWriter(w)
			defer func() {
				if panicValue := recover(); panicValue != nil {
					span.RecordError(errors.New("HTTP handler panicked"))
					span.SetStatus(codes.Error, "HTTP handler panicked")
					span.SetAttributes(attribute.String("error.type", "panic"))
					panic(panicValue)
				}
				if writer.status >= http.StatusInternalServerError {
					span.SetStatus(codes.Error, http.StatusText(writer.status))
					span.SetAttributes(attribute.String("error.type", "http.server.error"))
				}
			}()
			next.ServeHTTP(writer, r)
		}), operation,
			otelhttp.WithTracerProvider(provider),
			otelhttp.WithMeterProvider(metricnoop.NewMeterProvider()),
			otelhttp.WithPropagators(propagator),
			otelhttp.WithSpanNameFormatter(func(string, *http.Request) string { return operation }),
		)
	}
}
