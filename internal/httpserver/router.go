package httpserver

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	servicehttp "github.com/poiuyttq/tripgo/internal/http/service"
)

type Instrumentation struct {
	Recover func(http.Handler) http.Handler
	Tracing func(method, route string, next http.Handler) http.Handler
	Logging func(method, route string, next http.Handler) http.Handler
	Metrics func(method, route string, next http.Handler) http.Handler
}

func NewRouter(ready, createTrip, getTrip, finishTrip http.Handler, instrumentation Instrumentation) http.Handler {
	r := chi.NewRouter()
	allowedMethods := make(map[string][]string)
	register := func(method, route string, handler http.Handler) http.Handler {
		handler = instrumentation.Metrics(method, route, handler)
		handler = instrumentation.Logging(method, route, handler)
		handler = instrumentation.Tracing(method, route, handler)
		return handler
	}
	addRoute := func(method, route string, handler http.Handler) {
		allowedMethods[route] = append(allowedMethods[route], method)
		r.Method(method, route, register(method, route, handler))
	}

	addRoute(http.MethodGet, "/health", http.HandlerFunc(servicehttp.HealthHandler))
	addRoute(http.MethodGet, "/ready", ready)
	addRoute(http.MethodPost, "/api/v1/trips", createTrip)
	addRoute(http.MethodGet, "/api/v1/trips/{tripId}", getTrip)
	addRoute(http.MethodPost, "/api/v1/trips/{tripId}/finish", finishTrip)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		register(r.Method, "unmatched", http.NotFoundHandler()).ServeHTTP(w, r)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		route := methodNotAllowedRoute(r.URL.Path, allowedMethods)
		for _, method := range allowedMethods[route] {
			w.Header().Add("Allow", method)
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusMethodNotAllowed)
		})
		register(r.Method, route, handler).ServeHTTP(w, r)
	})

	return instrumentation.Recover(r)
}

func methodNotAllowedRoute(path string, routes map[string][]string) string {
	pathSegments := splitPath(path)
	for route := range routes {
		routeSegments := splitPath(route)
		if len(pathSegments) != len(routeSegments) {
			continue
		}
		matches := true
		for index, routeSegment := range routeSegments {
			if strings.HasPrefix(routeSegment, "{") && strings.HasSuffix(routeSegment, "}") {
				continue
			}
			if routeSegment != pathSegments[index] {
				matches = false
				break
			}
		}
		if matches {
			return route
		}
	}
	return "method_not_allowed"
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
