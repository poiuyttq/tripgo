package service

import (
	"context"
	"net/http"
	"time"

	api "github.com/poiuyttq/tripgo/internal/generated"
	"github.com/poiuyttq/tripgo/internal/http/response"
	"github.com/poiuyttq/tripgo/internal/requesttrace"
)

type readinessChecker interface {
	Ping(ctx context.Context) error
}

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func ReadyHandler(checker readinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := checker.Ping(ctx); err != nil {
			requesttrace.SetError(r.Context(), err)
			response.WriteJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
			return
		}
		response.WriteJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
	}
}
