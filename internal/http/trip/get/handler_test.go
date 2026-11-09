package get_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/poiuyttq/tripgo/internal/domain/trip"
	api "github.com/poiuyttq/tripgo/internal/generated"
	gethandler "github.com/poiuyttq/tripgo/internal/http/trip/get"
	"github.com/poiuyttq/tripgo/internal/model"
)

func TestGetTripHandler(t *testing.T) {
	tripID := "11111111-1111-4111-8111-111111111111"
	internalErr := errors.New("database failure details")
	wantTrip := model.Trip{
		ID:       tripID,
		UserID:   "22222222-2222-4222-8222-222222222222",
		DriverID: "33333333-3333-4333-8333-333333333333",
		Status:   "active",
	}
	tests := []struct {
		name       string
		pathID     string
		trip       model.Trip
		err        error
		wantStatus int
		wantCode   string
		call       bool
	}{
		{name: "found", pathID: tripID, trip: wantTrip, wantStatus: http.StatusOK, call: true},
		{name: "invalid id", pathID: "not-a-uuid", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "not found", pathID: tripID, err: domaintrip.ErrTripNotFound, wantStatus: http.StatusNotFound, wantCode: "trip_not_found", call: true},
		{name: "hides repository details", pathID: tripID, err: internalErr, wantStatus: http.StatusInternalServerError, wantCode: "internal_error", call: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			useCase := NewMocktripGetter(controller)
			if tt.call {
				useCase.EXPECT().GetTrip(gomock.Any(), tt.pathID).Return(tt.trip, tt.err)
			}

			router := chi.NewRouter()
			router.Get("/api/v1/trips/{tripId}", gethandler.NewHandler(useCase))
			request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tt.pathID, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var body api.Trip
				if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if body.Id != uuid.MustParse(wantTrip.ID) || body.Status != api.Active {
					t.Fatalf("response trip = %+v", body)
				}
				return
			}

			var problem api.Problem
			if err := json.NewDecoder(response.Body).Decode(&problem); err != nil {
				t.Fatalf("decode problem response: %v", err)
			}
			if problem.Code != tt.wantCode {
				t.Fatalf("problem code = %q, want %q", problem.Code, tt.wantCode)
			}
			if tt.err != nil && response.Body.String() != "" && problem.Detail != nil && *problem.Detail == tt.err.Error() {
				t.Fatalf("problem response exposed repository error: %s", response.Body.String())
			}
		})
	}
}
