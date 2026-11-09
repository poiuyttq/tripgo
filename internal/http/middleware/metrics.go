package middleware

import (
	"context"
	"net/http"
	"time"
)

type RequestMetrics interface {
	RequestStarted(ctx context.Context, method, route string)
	RequestFinished(ctx context.Context, method, route string, status int, duration time.Duration)
}

func Metrics(metrics RequestMetrics, method, route string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			writer := newResponseWriter(w)
			metrics.RequestStarted(r.Context(), method, route)
			defer func() {
				panicValue := recover()
				status := writer.status
				if panicValue != nil {
					status = http.StatusInternalServerError
				}
				metrics.RequestFinished(r.Context(), method, route, status, time.Since(startedAt))
				if panicValue != nil {
					panic(panicValue)
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
