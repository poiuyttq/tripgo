package finish

import (
	"context"
	"fmt"
	"time"

	domaintrip "github.com/poiuyttq/tripgo/internal/domain/trip"
	"github.com/poiuyttq/tripgo/internal/model"
)

type UseCase struct {
	trips    tripRepository
	history  historyRepository
	tx       txManager
	observer tripObserver
}

func New(trips tripRepository, history historyRepository, tx txManager, observer tripObserver) *UseCase {
	return &UseCase{trips: trips, history: history, tx: tx, observer: observer}
}

func (u *UseCase) FinishTrip(ctx context.Context, tripID string) error {
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		current, err := u.trips.GetByIDForUpdate(ctx, tripID)
		if err != nil {
			return fmt.Errorf("get trip: %w", err)
		}

		entity := domaintrip.Trip{Status: domaintrip.Status(current.Status)}
		finishedAt := time.Now().UTC()
		if err := entity.Finish(finishedAt); err != nil {
			return err
		}

		if err := u.trips.Finish(ctx, tripID, *entity.FinishedAt); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		fromStatus := current.Status
		history := model.TripStatusHistory{
			TripID:     current.ID,
			FromStatus: &fromStatus,
			ToStatus:   string(entity.Status),
			ChangedAt:  *entity.FinishedAt,
		}
		if err := u.history.Add(ctx, history); err != nil {
			return fmt.Errorf("write trip status history: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	u.observer.TripCompleted(ctx, tripID)
	return nil
}
