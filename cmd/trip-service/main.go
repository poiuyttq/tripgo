package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"errors"
	"github.com/poiuyttq/tripgo/internal/config"
	"github.com/poiuyttq/tripgo/internal/httpserver"
	"github.com/poiuyttq/tripgo/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	log.Printf("config loaded: %+v", cfg)

	dbPool, err := storage.New(ctx, cfg)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	defer dbPool.Close()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpserver.NewRouter(dbPool),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	log.Printf("starting http server on %s", cfg.HTTPAddr)

	serverFailed := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverFailed <- err
		}
	}()

	select{
		case <-ctx.Done():
			// graceful shutdown
		case err :=<-serverFailed:
			log.Fatalf("http server failed: %v", err)
			// server failed
	}
	

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("server exited properly")
}
