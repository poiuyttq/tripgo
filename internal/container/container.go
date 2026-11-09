package container

import (
	"context"
	"net/http"

	"github.com/poiuyttq/tripgo/internal/config"
	httpmiddleware "github.com/poiuyttq/tripgo/internal/http/middleware"
	servicehttp "github.com/poiuyttq/tripgo/internal/http/service"
	createhandler "github.com/poiuyttq/tripgo/internal/http/trip/create"
	finishhandler "github.com/poiuyttq/tripgo/internal/http/trip/finish"
	gethandler "github.com/poiuyttq/tripgo/internal/http/trip/get"
	"github.com/poiuyttq/tripgo/internal/httpserver"
	"github.com/poiuyttq/tripgo/internal/infra/postgres"
	"github.com/poiuyttq/tripgo/internal/infra/postgres/transaction"
	"github.com/poiuyttq/tripgo/internal/infra/telemetry"
	"github.com/poiuyttq/tripgo/internal/repository"
	"github.com/poiuyttq/tripgo/internal/usecase/trip/create"
	"github.com/poiuyttq/tripgo/internal/usecase/trip/finish"
	"github.com/poiuyttq/tripgo/internal/usecase/trip/get"
)

type App struct {
	handler http.Handler
	close   func()
}

func New(ctx context.Context, cfg config.Config, instrumentation *telemetry.Telemetry) (*App, error) {
	pool, err := postgres.NewPool(ctx, cfg, instrumentation.PGXTracer())
	if err != nil {
		return nil, err
	}

	tripRepo := repository.NewTripRepository(pool)
	historyRepo := repository.NewTripStatusHistoryRepository()
	txManager := transaction.New(pool)

	createTrip := create.New(tripRepo, historyRepo, txManager, instrumentation)
	getTrip := get.New(tripRepo)
	finishTrip := finish.New(tripRepo, historyRepo, txManager, instrumentation)

	handler := httpserver.NewRouter(
		servicehttp.ReadyHandler(pool),
		createhandler.NewHandler(createTrip),
		gethandler.NewHandler(getTrip),
		finishhandler.NewHandler(finishTrip, getTrip),
		httpserver.Instrumentation{
			Recover: httpmiddleware.Recover(instrumentation.Logger()),
			Tracing: func(method, route string, next http.Handler) http.Handler {
				return httpmiddleware.Tracing(instrumentation.TraceProvider(), instrumentation.Propagator(), method, route)(next)
			},
			Logging: func(method, route string, next http.Handler) http.Handler {
				return httpmiddleware.Logging(instrumentation.Logger(), method, route)(next)
			},
			Metrics: func(method, route string, next http.Handler) http.Handler {
				return httpmiddleware.Metrics(instrumentation, method, route)(next)
			},
		},
	)

	return &App{handler: handler, close: pool.Close}, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Close() {
	a.close()
}
