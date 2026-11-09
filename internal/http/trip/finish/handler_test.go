package finish_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	"github.com/poiuyttq/tripgo/internal/domain/trip"
	api "github.com/poiuyttq/tripgo/internal/generated"
	finishhandler "github.com/poiuyttq/tripgo/internal/http/trip/finish"
	"github.com/poiuyttq/tripgo/internal/model"
)

func TestFinishTripHandler(t *testing.T) {
	tripID := "11111111-1111-4111-8111-111111111111"
	tests := []struct {
		name       string
		pathID     string
		finishErr  error
		wantStatus int
		wantCode   string
		callFinish bool
		callGet    bool
	}{
		{name: "finishes trip", pathID: tripID, wantStatus: http.StatusOK, callFinish: true, callGet: true},
		{name: "invalid id", pathID: "not-a-uuid", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "already completed", pathID: tripID, finishErr: trip.ErrTripCompleted, wantStatus: http.StatusConflict, wantCode: "trip_completed", callFinish: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			finisher := NewMocktripFinisher(controller)
			getter := NewMocktripGetter(controller)
			if tt.callFinish {
				finisher.EXPECT().FinishTrip(gomock.Any(), tt.pathID).Return(tt.finishErr)
			}
			if tt.callGet {
				getter.EXPECT().GetTrip(gomock.Any(), tt.pathID).Return(model.Trip{
					ID:       tripID,
					UserID:   "22222222-2222-4222-8222-222222222222",
					DriverID: "33333333-3333-4333-8333-333333333333",
					Status:   "completed",
				}, nil)
			}

			router := chi.NewRouter()
			router.Post("/api/v1/trips/{tripId}/finish", finishhandler.NewHandler(finisher, getter))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tt.pathID+"/finish", nil)
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
				if body.Status != api.Completed || body.Id.String() != tripID {
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
		})
	}
}
