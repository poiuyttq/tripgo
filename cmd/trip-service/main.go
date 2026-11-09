package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/poiuyttq/tripgo/internal/config"
	"github.com/poiuyttq/tripgo/internal/container"
	"github.com/poiuyttq/tripgo/internal/infra/telemetry"
)

func main() {
	if err := run(); err != nil {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		logger.Error("trip service failed", "service", "trip-service", "trace_id", "", "span_id", "", "error", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	telemetryProvider, err := telemetry.New(ctx, cfg, buildVersion())
	if err != nil {
		return fmt.Errorf("initialize telemetry: %w", err)
	}
	telemetryClosed := false
	defer func() {
		if !telemetryClosed {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			runErr = errors.Join(runErr, telemetryProvider.Shutdown(shutdownCtx))
		}
	}()
	logger := telemetryProvider.Logger()
	logger.InfoContext(ctx, "service starting", "address", cfg.HTTPAddr)

	app, err := container.New(ctx, cfg, telemetryProvider)
	if err != nil {
		return err
	}
	appClosed := false
	defer func() {
		if !appClosed {
			app.Close()
		}
	}()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.Handler(),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	serverFailed := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverFailed <- err
		}
	}()

	var serverErr error
	select {
	case <-ctx.Done():
		// graceful shutdown
	case serverErr = <-serverFailed:
		logger.ErrorContext(ctx, "HTTP server failed", "error", serverErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		logger.ErrorContext(shutdownCtx, "graceful shutdown timed out", "error", shutdownErr)
		_ = server.Close()
	}
	app.Close()
	appClosed = true
	telemetryErr := telemetryProvider.Shutdown(shutdownCtx)
	telemetryClosed = true
	if serverErr != nil {
		runErr = errors.Join(runErr, serverErr)
	}
	if shutdownErr != nil {
		runErr = errors.Join(runErr, shutdownErr)
	}
	if telemetryErr != nil {
		runErr = errors.Join(runErr, telemetryErr)
	}
	logger.InfoContext(context.Background(), "service stopped")
	return runErr
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return "unknown"
	}
	return info.Main.Version
}
