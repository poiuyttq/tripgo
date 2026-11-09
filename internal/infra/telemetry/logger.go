package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/poiuyttq/tripgo/internal/requesttrace"
)

type contextHandler struct {
	slog.Handler
	service string
}

func newLogger(service, configuredLevel string) (*slog.Logger, error) {
	level, err := parseLogLevel(configuredLevel)
	if err != nil {
		return nil, err
	}
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(&contextHandler{Handler: base, service: service}), nil
}

func parseLogLevel(configuredLevel string) (slog.Level, error) {
	switch strings.ToLower(configuredLevel) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported LOG_LEVEL %q", configuredLevel)
	}
}

func (h *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	traceID, spanID := requesttrace.IDs(ctx)
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
		spanID = spanContext.SpanID().String()
	}
	record.AddAttrs(
		slog.String("service", h.service),
		slog.String("trace_id", traceID),
		slog.String("span_id", spanID),
	)
	return h.Handler.Handle(ctx, record)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs), service: h.service}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name), service: h.service}
}

var _ slog.Handler = (*contextHandler)(nil)
