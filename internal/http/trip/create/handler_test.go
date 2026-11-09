package create_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	domaintrip "github.com/poiuyttq/tripgo/internal/domain/trip"
	api "github.com/poiuyttq/tripgo/internal/generated"
	create "github.com/poiuyttq/tripgo/internal/http/trip/create"
	"github.com/poiuyttq/tripgo/internal/model"
)

const (
	validCreateTripRequest = `{"user_id":"96c74278-9417-4b3f-9d87-62a32cfa7066","driver_id":"d8abf18c-f38b-462e-be8c-855c03367844","start_point":{"latitude":59.25,"longitude":30.5},"end_point":{"latitude":0,"longitude":-180},"price":1450}`
	invalidRequestCode     = "invalid_request"
)

func TestCreateTripHandler(t *testing.T) {
	internalErr := errors.New("database password must not be exposed")
	created := model.Trip{
		ID:       "11111111-1111-4111-8111-111111111111",
		UserID:   "96c74278-9417-4b3f-9d87-62a32cfa7066",
		DriverID: "d8abf18c-f38b-462e-be8c-855c03367844",
		Status:   "active",
	}
	tests := []struct {
		name       string
		body       string
		useCaseErr error
		wantStatus int
		wantCode   string
		call       bool
	}{
		{name: "creates trip", body: validCreateTripRequest, wantStatus: http.StatusCreated, call: true},
		{name: "rejects malformed JSON", body: `{"user_id":`, wantStatus: http.StatusBadRequest, wantCode: invalidRequestCode},
		{name: "rejects trailing JSON value", body: validCreateTripRequest + ` {}`, wantStatus: http.StatusBadRequest, wantCode: invalidRequestCode},
		{name: "rejects unknown nested field", body: strings.Replace(validCreateTripRequest, `"latitude":59.25`, `"extra":true,"latitude":59.25`, 1), wantStatus: http.StatusBadRequest, wantCode: invalidRequestCode},
		{name: "rejects server-managed field", body: strings.TrimSuffix(validCreateTripRequest, "}") + `,"status":"completed"}`, wantStatus: http.StatusBadRequest, wantCode: invalidRequestCode},
		{name: "rejects missing required field", body: `{"user_id":"96c74278-9417-4b3f-9d87-62a32cfa7066"}`, wantStatus: http.StatusBadRequest, wantCode: invalidRequestCode},
		{name: "rejects invalid coordinates", body: strings.Replace(validCreateTripRequest, `"latitude":59.25`, `"latitude":91`, 1), wantStatus: http.StatusBadRequest, wantCode: invalidRequestCode},
		{name: "rejects negative price", body: strings.Replace(validCreateTripRequest, `"price":1450`, `"price":-1`, 1), wantStatus: http.StatusBadRequest, wantCode: invalidRequestCode},
		{name: "maps driver busy", body: validCreateTripRequest, useCaseErr: domaintrip.ErrDriverBusy, wantStatus: http.StatusConflict, wantCode: "driver_busy", call: true},
		{name: "hides internal error", body: validCreateTripRequest, useCaseErr: internalErr, wantStatus: http.StatusInternalServerError, wantCode: "internal_error", call: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			useCase := NewMocktripCreator(controller)
			if tt.call {
				useCase.EXPECT().CreateTrip(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got model.Trip) (model.Trip, error) {
					if tt.useCaseErr == nil && (got.UserID != created.UserID || got.DriverID != created.DriverID || got.Price != 1450 || got.Status != "active") {
						t.Errorf("unexpected trip passed to usecase: %+v", got)
					}
					if tt.useCaseErr != nil {
						return model.Trip{}, tt.useCaseErr
					}
					return created, nil
				})
			}

			request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			create.NewHandler(useCase).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.wantStatus == http.StatusCreated {
				if got := response.Header().Get("Location"); got != "/api/v1/trips/"+created.ID {
					t.Fatalf("Location = %q, want trip resource location", got)
				}
				var body api.Trip
				if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if body.Id.String() != created.ID || body.Status != api.Active {
					t.Fatalf("response trip = %+v, want id %s and active status", body, created.ID)
				}
				return
			}

			if got := response.Header().Get("Content-Type"); got != "application/problem+json" {
				t.Fatalf("Content-Type = %q, want application/problem+json", got)
			}
			var problem api.Problem
			if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&problem); err != nil {
				t.Fatalf("decode problem response: %v", err)
			}
			if problem.Code != tt.wantCode {
				t.Fatalf("problem code = %q, want %q", problem.Code, tt.wantCode)
			}
			if tt.useCaseErr != nil && strings.Contains(response.Body.String(), tt.useCaseErr.Error()) {
				t.Fatalf("problem body exposes internal error: %s", response.Body.String())
			}
		})
	}
}
