package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/poiuyttq/tripgo/internal/httpserver"
)

type instrumentationCall struct {
	layer  string
	method string
	route  string
}

func TestRouterInstrumentsMethodNotAllowedAndNotFound(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantRoute  string
	}{
		{
			name:       "known path with unsupported method",
			method:     http.MethodGet,
			path:       "/api/v1/trips",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodPost,
			wantRoute:  "/api/v1/trips",
		},
		{
			name:       "parameterized path with unsupported method",
			method:     http.MethodPost,
			path:       "/api/v1/trips/11111111-1111-4111-8111-111111111111",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  http.MethodGet,
			wantRoute:  "/api/v1/trips/{tripId}",
		},
		{
			name:       "unknown path",
			method:     http.MethodGet,
			path:       "/unknown/path/123",
			wantStatus: http.StatusNotFound,
			wantRoute:  "unmatched",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []instrumentationCall
			wrap := func(layer string) func(method, route string, next http.Handler) http.Handler {
				return func(method, route string, next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls = append(calls, instrumentationCall{layer: layer, method: method, route: route})
						next.ServeHTTP(w, r)
					})
				}
			}
			instrumentation := httpserver.Instrumentation{
				Recover: func(next http.Handler) http.Handler { return next },
				Tracing: wrap("tracing"),
				Logging: wrap("logging"),
				Metrics: wrap("metrics"),
			}
			emptyHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
			router := httpserver.NewRouter(emptyHandler, emptyHandler, emptyHandler, emptyHandler, instrumentation)
			request := httptest.NewRequest(tt.method, tt.path, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if got := response.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, tt.wantAllow)
			}
			if len(calls) != 3 {
				t.Fatalf("instrumentation calls = %d, want tracing, logging, metrics", len(calls))
			}
			for _, call := range calls {
				if call.method != tt.method || call.route != tt.wantRoute {
					t.Errorf("%s labels = %s %s, want %s %s", call.layer, call.method, call.route, tt.method, tt.wantRoute)
				}
			}
		})
	}
}
