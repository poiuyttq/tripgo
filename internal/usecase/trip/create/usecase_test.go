package create_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/poiuyttq/tripgo/internal/domain/trip"
	"github.com/poiuyttq/tripgo/internal/model"
	create "github.com/poiuyttq/tripgo/internal/usecase/trip/create"
)

var (
	errLookup  = errors.New("lookup failed")
	errInsert  = errors.New("insert failed")
	errHistory = errors.New("history insert failed")
)

func TestCreateTrip(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testing.T, *MocktripRepository, *MockhistoryRepository, *MocktxManager, model.Trip)
		wantErr  error
		wantTrip bool
	}{
		{
			name: "creates trip and history",
			setup: func(t *testing.T, trips *MocktripRepository, history *MockhistoryRepository, tx *MocktxManager, expected model.Trip) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetActiveByDriverID(gomock.Any(), expected.DriverID).Return(model.Trip{}, false, nil)
				trips.EXPECT().Create(gomock.Any(), expected).Return(nil)
				history.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got model.TripStatusHistory) error {
					if got.TripID != expected.ID || got.ToStatus != expected.Status || got.ChangedAt.IsZero() {
						t.Errorf("unexpected history record: %+v", got)
					}
					return nil
				})
			},
			wantTrip: true,
		},
		{
			name: "rejects driver with active trip",
			setup: func(_ *testing.T, trips *MocktripRepository, _ *MockhistoryRepository, tx *MocktxManager, expected model.Trip) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetActiveByDriverID(gomock.Any(), expected.DriverID).Return(model.Trip{}, true, nil)
			},
			wantErr: trip.ErrDriverBusy,
		},
		{
			name: "wraps active trip lookup error",
			setup: func(_ *testing.T, trips *MocktripRepository, _ *MockhistoryRepository, tx *MocktxManager, expected model.Trip) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetActiveByDriverID(gomock.Any(), expected.DriverID).Return(model.Trip{}, false, errLookup)
			},
			wantErr: errLookup,
		},
		{
			name: "returns create error",
			setup: func(_ *testing.T, trips *MocktripRepository, _ *MockhistoryRepository, tx *MocktxManager, expected model.Trip) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetActiveByDriverID(gomock.Any(), expected.DriverID).Return(model.Trip{}, false, nil)
				trips.EXPECT().Create(gomock.Any(), expected).Return(errInsert)
			},
			wantErr: errInsert,
		},
		{
			name: "wraps history error",
			setup: func(_ *testing.T, trips *MocktripRepository, history *MockhistoryRepository, tx *MocktxManager, expected model.Trip) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetActiveByDriverID(gomock.Any(), expected.DriverID).Return(model.Trip{}, false, nil)
				trips.EXPECT().Create(gomock.Any(), expected).Return(nil)
				history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(errHistory)
			},
			wantErr: errHistory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			trips := NewMocktripRepository(controller)
			history := NewMockhistoryRepository(controller)
			tx := NewMocktxManager(controller)
			observer := NewMocktripObserver(controller)
			input := testTrip()
			tt.setup(t, trips, history, tx, input)
			if tt.wantTrip {
				observer.EXPECT().TripCreated(gomock.Any(), input.ID)
			}

			got, err := create.New(trips, history, tx, observer).CreateTrip(context.Background(), input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateTrip() error = %v, want %v", err, tt.wantErr)
			}
			if (err == nil) != tt.wantTrip {
				t.Fatalf("CreateTrip() error = %v, want success = %t", err, tt.wantTrip)
			}
			if tt.wantTrip && !reflect.DeepEqual(got, input) {
				t.Fatalf("CreateTrip() trip = %+v, want %+v", got, input)
			}
		})
	}
}

func TestCreateTripRejectsInvalidModelBeforeTransaction(t *testing.T) {
	controller := gomock.NewController(t)
	trips := NewMocktripRepository(controller)
	history := NewMockhistoryRepository(controller)
	tx := NewMocktxManager(controller)
	observer := NewMocktripObserver(controller)
	input := testTrip()
	input.Price = -1

	_, err := create.New(trips, history, tx, observer).CreateTrip(context.Background(), input)
	var validationErr *trip.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("CreateTrip() error = %v, want *ValidationError", err)
	}
	if validationErr.Field != "price" {
		t.Fatalf("validation field = %q, want price", validationErr.Field)
	}
}

func runTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func testTrip() model.Trip {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	return model.Trip{
		ID:        "11111111-1111-4111-8111-111111111111",
		UserID:    "22222222-2222-4222-8222-222222222222",
		DriverID:  "33333333-3333-4333-8333-333333333333",
		StartLat:  59.9,
		StartLng:  30.3,
		EndLat:    59.8,
		EndLng:    30.4,
		Price:     1450,
		Status:    "active",
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
