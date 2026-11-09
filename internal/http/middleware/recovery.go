package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	api "github.com/poiuyttq/tripgo/internal/generated"
	"github.com/poiuyttq/tripgo/internal/http/response"
	"github.com/poiuyttq/tripgo/internal/requesttrace"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(requesttrace.WithState(r.Context()))
			defer func() {
				if panicValue := recover(); panicValue != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						"panic_type", fmt.Sprintf("%T", panicValue),
						"stack", string(debug.Stack()),
					)
					detail := "Internal server error"
					instance := r.URL.Path
					response.WriteProblem(w, http.StatusInternalServerError, api.Problem{
						Type:     "https://tripgo.example/problems/internal_error",
						Title:    "Internal Server Error",
						Status:   http.StatusInternalServerError,
						Code:     "internal_error",
						Detail:   &detail,
						Instance: &instance,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
