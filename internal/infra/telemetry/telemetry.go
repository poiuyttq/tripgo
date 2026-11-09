package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/poiuyttq/tripgo/internal/config"
)

const (
	defaultServiceName = "trip-service"
	serviceNamespace   = "tripgo"
	defaultEnvironment = "local"
	metricScope        = "github.com/poiuyttq/tripgo/internal/infra/telemetry"
)

type Telemetry struct {
	logger          *slog.Logger
	traceProvider   *sdktrace.TracerProvider
	metricProvider  *sdkmetric.MeterProvider
	propagator      propagation.TextMapPropagator
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
	dbDuration      metric.Float64Histogram
	tripsCreated    metric.Int64Counter
	tripsCompleted  metric.Int64Counter
}

func New(ctx context.Context, cfg config.Config, version string) (*Telemetry, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return nil, errors.New("OTEL_EXPORTER_OTLP_ENDPOINT is required")
	}
	if protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); protocol != "" && protocol != "http/protobuf" {
		return nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q: this service uses http/protobuf", protocol)
	}

	serviceName := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
	if serviceName == "" {
		serviceName = defaultServiceName
	}
	logger, err := newLogger(serviceName, cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(defaultServiceName),
			semconv.ServiceNamespace(serviceNamespace),
			semconv.ServiceVersion(version),
			attribute.String("deployment.environment", defaultEnvironment),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	sampler, err := samplerFromEnvironment()
	if err != nil {
		return nil, err
	}
	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithTimeout(2*time.Second))
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(time.Second)),
	)

	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithTimeout(2*time.Second))
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return nil, errors.Join(fmt.Errorf("create OTLP metric exporter: %w", err), traceProvider.Shutdown(cleanupCtx))
	}
	metricProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)

	meter := metricProvider.Meter(metricScope)
	requestDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of incoming HTTP requests"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10),
	)
	if err != nil {
		return nil, shutdownOnSetupError(ctx, metricProvider, traceProvider, fmt.Errorf("create HTTP duration metric: %w", err))
	}
	activeRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of HTTP requests currently being served"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, shutdownOnSetupError(ctx, metricProvider, traceProvider, fmt.Errorf("create active HTTP requests metric: %w", err))
	}
	dbDuration, err := meter.Float64Histogram(
		"db.client.operation.duration",
		metric.WithDescription("Duration of PostgreSQL client operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5),
	)
	if err != nil {
		return nil, shutdownOnSetupError(ctx, metricProvider, traceProvider, fmt.Errorf("create database duration metric: %w", err))
	}
	tripsCreated, err := meter.Int64Counter("tripgo.trips.created", metric.WithDescription("Number of created trips"), metric.WithUnit("{trip}"))
	if err != nil {
		return nil, shutdownOnSetupError(ctx, metricProvider, traceProvider, fmt.Errorf("create trips created metric: %w", err))
	}
	tripsCompleted, err := meter.Int64Counter("tripgo.trips.completed", metric.WithDescription("Number of completed trips"), metric.WithUnit("{trip}"))
	if err != nil {
		return nil, shutdownOnSetupError(ctx, metricProvider, traceProvider, fmt.Errorf("create trips completed metric: %w", err))
	}

	return &Telemetry{
		logger:          logger,
		traceProvider:   traceProvider,
		metricProvider:  metricProvider,
		propagator:      propagation.TraceContext{},
		requestDuration: requestDuration,
		activeRequests:  activeRequests,
		dbDuration:      dbDuration,
		tripsCreated:    tripsCreated,
		tripsCompleted:  tripsCompleted,
	}, nil
}

func shutdownOnSetupError(ctx context.Context, metrics *sdkmetric.MeterProvider, traces *sdktrace.TracerProvider, setupErr error) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(setupErr, metrics.Shutdown(cleanupCtx), traces.Shutdown(cleanupCtx))
}

func (t *Telemetry) Logger() *slog.Logger {
	return t.logger
}

func (t *Telemetry) TraceProvider() trace.TracerProvider {
	return t.traceProvider
}

func (t *Telemetry) Propagator() propagation.TextMapPropagator {
	return t.propagator
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.metricProvider.Shutdown(ctx), t.traceProvider.Shutdown(ctx))
}

func (t *Telemetry) RequestStarted(ctx context.Context, method, route string) {
	t.activeRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
	))
}

func (t *Telemetry) RequestFinished(ctx context.Context, method, route string, status int, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.response.status_code", status),
	}
	t.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs[:2]...))
	t.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

func (t *Telemetry) TripCreated(ctx context.Context, tripID string) {
	t.tripsCreated.Add(ctx, 1)
	t.logger.InfoContext(ctx, "trip created", "trip_id", tripID)
}

func (t *Telemetry) TripCompleted(ctx context.Context, tripID string) {
	t.tripsCompleted.Add(ctx, 1)
	t.logger.InfoContext(ctx, "trip completed", "trip_id", tripID)
}

func samplerFromEnvironment() (sdktrace.Sampler, error) {
	samplerName := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER")))
	if samplerName == "" {
		samplerName = "parentbased_always_on"
	}
	if samplerName == "always_on" {
		return sdktrace.AlwaysSample(), nil
	}
	if samplerName == "always_off" {
		return sdktrace.NeverSample(), nil
	}

	ratio := 1.0
	if samplerName == "traceidratio" || samplerName == "parentbased_traceidratio" {
		configuredRatio := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
		if configuredRatio != "" {
			parsedRatio, err := strconv.ParseFloat(configuredRatio, 64)
			if err != nil || parsedRatio < 0 || parsedRatio > 1 {
				return nil, fmt.Errorf("OTEL_TRACES_SAMPLER_ARG must be between 0 and 1")
			}
			ratio = parsedRatio
		}
	}
	switch samplerName {
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(ratio), nil
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio)), nil
	default:
		return nil, fmt.Errorf("unsupported OTEL_TRACES_SAMPLER %q", samplerName)
	}
}
