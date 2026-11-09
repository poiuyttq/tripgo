package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/poiuyttq/tripgo/internal/domain/trip"
)

func TestTripFinish(t *testing.T) {
	finishedAt := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		status       trip.Status
		wantErr      error
		wantStatus   trip.Status
		wantFinished bool
	}{
		{
			name:         "active trip finishes",
			status:       trip.StatusActive,
			wantStatus:   trip.StatusCompleted,
			wantFinished: true,
		},
		{
			name:       "completed trip cannot finish again",
			status:     trip.StatusCompleted,
			wantErr:    trip.ErrTripCompleted,
			wantStatus: trip.StatusCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := trip.Trip{Status: tt.status}
			err := entity.Finish(finishedAt)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Finish() error = %v, want %v", err, tt.wantErr)
			}
			if entity.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", entity.Status, tt.wantStatus)
			}
			if (entity.FinishedAt != nil) != tt.wantFinished {
				t.Fatalf("finished timestamp present = %t, want %t", entity.FinishedAt != nil, tt.wantFinished)
			}
			if entity.FinishedAt != nil && !entity.FinishedAt.Equal(finishedAt) {
				t.Fatalf("finished timestamp = %v, want %v", *entity.FinishedAt, finishedAt)
			}
		})
	}
}

func TestTripValidate(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		mutate    func(*trip.Trip)
		wantField string
	}{
		{name: "active trip is valid"},
		{name: "completed trip with timestamp is valid", mutate: func(entity *trip.Trip) {
			finishedAt := entity.StartedAt.Add(time.Minute)
			entity.Status = trip.StatusCompleted
			entity.FinishedAt = &finishedAt
		}},
		{name: "coordinate boundaries are inclusive", mutate: func(entity *trip.Trip) {
			entity.StartPoint = trip.Coordinates{Latitude: 90, Longitude: -180}
			entity.EndPoint = trip.Coordinates{Latitude: -90, Longitude: 180}
		}},
		{name: "id is required", mutate: func(entity *trip.Trip) { entity.ID = "" }, wantField: "id"},
		{name: "user id is required", mutate: func(entity *trip.Trip) { entity.UserID = "" }, wantField: "user_id"},
		{name: "driver id is required", mutate: func(entity *trip.Trip) { entity.DriverID = "" }, wantField: "driver_id"},
		{name: "started at is required", mutate: func(entity *trip.Trip) { entity.StartedAt = time.Time{} }, wantField: "started_at"},
		{name: "price cannot be negative", mutate: func(entity *trip.Trip) { entity.Price = -1 }, wantField: "price"},
		{name: "start latitude is bounded", mutate: func(entity *trip.Trip) { entity.StartPoint.Latitude = 91 }, wantField: "start_point.latitude"},
		{name: "end longitude is bounded", mutate: func(entity *trip.Trip) { entity.EndPoint.Longitude = -181 }, wantField: "end_point.longitude"},
		{name: "active trip cannot have finished timestamp", mutate: func(entity *trip.Trip) { entity.FinishedAt = &now }, wantField: "finished_at"},
		{name: "completed trip requires finished timestamp", mutate: func(entity *trip.Trip) { entity.Status = trip.StatusCompleted }, wantField: "finished_at"},
		{name: "status must be known", mutate: func(entity *trip.Trip) { entity.Status = "cancelled" }, wantField: "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity := validTrip(now)
			if tt.mutate != nil {
				tt.mutate(&entity)
			}

			err := entity.Validate()
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}

			var validationErr *trip.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("Validate() error = %v, want *ValidationError", err)
			}
			if validationErr.Field != tt.wantField {
				t.Fatalf("validation field = %q, want %q", validationErr.Field, tt.wantField)
			}
		})
	}
}

func validTrip(startedAt time.Time) trip.Trip {
	return trip.Trip{
		ID:       "11111111-1111-4111-8111-111111111111",
		UserID:   "22222222-2222-4222-8222-222222222222",
		DriverID: "33333333-3333-4333-8333-333333333333",
		StartPoint: trip.Coordinates{
			Latitude:  59.9,
			Longitude: 30.3,
		},
		EndPoint: trip.Coordinates{
			Latitude:  59.8,
			Longitude: 30.4,
		},
		Price:     1450,
		Status:    trip.StatusActive,
		StartedAt: startedAt,
	}
}
