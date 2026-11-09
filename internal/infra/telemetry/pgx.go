package telemetry

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type queryStateKey struct{}

type queryState struct {
	startedAt  time.Time
	operation  string
	collection string
}

type pgxQueryTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

func (t *Telemetry) PGXTracer() pgx.QueryTracer {
	return &pgxQueryTracer{
		Tracer: otelpgx.NewTracer(
			otelpgx.WithTracerProvider(t.traceProvider),
			otelpgx.WithMeterProvider(metricnoop.NewMeterProvider()),
			otelpgx.WithDisableSQLStatementInAttributes(),
			otelpgx.WithDisableConnectionDetailsInAttributes(),
		),
		duration: t.dbDuration,
	}
}

func (t *pgxQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	operation := databaseOperation(data.SQL)
	collection := databaseCollection(data.SQL)
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.String("db.operation.name", operation),
		attribute.String("db.collection.name", collection),
	)
	return context.WithValue(ctx, queryStateKey{}, queryState{
		startedAt:  time.Now(),
		operation:  operation,
		collection: collection,
	})
}

func (t *pgxQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	state, ok := ctx.Value(queryStateKey{}).(queryState)
	if ok {
		attrs := []attribute.KeyValue{
			attribute.String("db.operation.name", state.operation),
			attribute.String("db.collection.name", state.collection),
			attribute.String("error.type", databaseErrorType(data.Err)),
		}
		if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
			trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", databaseErrorType(data.Err)))
		}
		t.duration.Record(ctx, time.Since(state.startedAt).Seconds(), metric.WithAttributes(attrs...))
	}
	t.Tracer.TraceQueryEnd(ctx, conn, data)
}

func databaseOperation(statement string) string {
	fields := strings.Fields(statement)
	if len(fields) == 0 {
		return "UNKNOWN"
	}
	return strings.ToUpper(fields[0])
}

func databaseCollection(statement string) string {
	statement = strings.ToLower(statement)
	switch {
	case strings.Contains(statement, "trip_status_history"):
		return "trip_status_history"
	case strings.Contains(statement, "trips"):
		return "trips"
	default:
		return "unknown"
	}
}

func databaseErrorType(err error) string {
	if err == nil || errors.Is(err, pgx.ErrNoRows) {
		return ""
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		return pgError.Code
	}
	return reflect.TypeOf(err).String()
}
