package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/poiuyttq/tripgo/internal/requesttrace"
)

func Logging(logger *slog.Logger, method, route string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			writer := newResponseWriter(w)
			defer func() {
				panicValue := recover()
				status := writer.status
				if panicValue != nil {
					status = http.StatusInternalServerError
				}

				level := slog.LevelInfo
				if status >= http.StatusInternalServerError {
					level = slog.LevelError
				}
				attrs := []slog.Attr{
					slog.String("method", method),
					slog.String("route", route),
					slog.Int("status", status),
					slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
				}
				if status >= http.StatusInternalServerError {
					if err := requesttrace.Error(r.Context()); err != nil {
						attrs = append(attrs, slog.String("error", err.Error()))
					}
				}
				logger.LogAttrs(r.Context(), level, "request handled", attrs...)
				if panicValue != nil {
					panic(panicValue)
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
