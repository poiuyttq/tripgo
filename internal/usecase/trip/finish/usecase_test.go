package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	domaintrip "github.com/poiuyttq/tripgo/internal/domain/trip"
	"github.com/poiuyttq/tripgo/internal/model"
	finish "github.com/poiuyttq/tripgo/internal/usecase/trip/finish"
)

const (
	activeTripStatus    = "active"
	completedTripStatus = "completed"
)

func TestFinishTrip(t *testing.T) {
	tripID := "11111111-1111-4111-8111-111111111111"
	updateErr := errors.New("update failed")
	historyErr := errors.New("history failed")
	lookupErr := domaintrip.ErrTripNotFound

	tests := []struct {
		name  string
		setup func(*testing.T, *MocktripRepository, *MockhistoryRepository, *MocktxManager)
		want  error
	}{
		{
			name: "finishes active trip and writes matching history",
			setup: func(t *testing.T, trips *MocktripRepository, history *MockhistoryRepository, tx *MocktxManager) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{ID: tripID, Status: activeTripStatus}, nil)
				var finishedAt time.Time
				trips.EXPECT().Finish(gomock.Any(), tripID, gomock.Any()).DoAndReturn(func(_ context.Context, _ string, at time.Time) error {
					finishedAt = at
					if at.IsZero() {
						t.Error("finish timestamp must be set")
					}
					return nil
				})
				history.EXPECT().Add(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got model.TripStatusHistory) error {
					if got.TripID != tripID || got.FromStatus == nil || *got.FromStatus != activeTripStatus || got.ToStatus != completedTripStatus || !got.ChangedAt.Equal(finishedAt) {
						t.Errorf("unexpected history record: %+v", got)
					}
					return nil
				})
			},
		},
		{
			name: "rejects already completed trip",
			setup: func(_ *testing.T, trips *MocktripRepository, _ *MockhistoryRepository, tx *MocktxManager) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{ID: tripID, Status: completedTripStatus}, nil)
			},
			want: domaintrip.ErrTripCompleted,
		},
		{
			name: "wraps lookup error",
			setup: func(_ *testing.T, trips *MocktripRepository, _ *MockhistoryRepository, tx *MocktxManager) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{}, lookupErr)
			},
			want: lookupErr,
		},
		{
			name: "wraps finish error",
			setup: func(_ *testing.T, trips *MocktripRepository, _ *MockhistoryRepository, tx *MocktxManager) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{ID: tripID, Status: activeTripStatus}, nil)
				trips.EXPECT().Finish(gomock.Any(), tripID, gomock.Any()).Return(updateErr)
			},
			want: updateErr,
		},
		{
			name: "wraps history error",
			setup: func(_ *testing.T, trips *MocktripRepository, history *MockhistoryRepository, tx *MocktxManager) {
				tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(runTransaction)
				trips.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{ID: tripID, Status: activeTripStatus}, nil)
				trips.EXPECT().Finish(gomock.Any(), tripID, gomock.Any()).Return(nil)
				history.EXPECT().Add(gomock.Any(), gomock.Any()).Return(historyErr)
			},
			want: historyErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			trips := NewMocktripRepository(controller)
			history := NewMockhistoryRepository(controller)
			tx := NewMocktxManager(controller)
			observer := NewMocktripObserver(controller)
			tt.setup(t, trips, history, tx)
			if tt.want == nil {
				observer.EXPECT().TripCompleted(gomock.Any(), tripID)
			}

			err := finish.New(trips, history, tx, observer).FinishTrip(context.Background(), tripID)
			if !errors.Is(err, tt.want) {
				t.Fatalf("FinishTrip() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func runTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
