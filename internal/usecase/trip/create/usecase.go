package create

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

func (u *UseCase) CreateTrip(ctx context.Context, trip model.Trip) (model.Trip, error) {
	entity := domaintrip.Trip{
		ID:       trip.ID,
		UserID:   trip.UserID,
		DriverID: trip.DriverID,
		StartPoint: domaintrip.Coordinates{
			Latitude:  trip.StartLat,
			Longitude: trip.StartLng,
		},
		EndPoint: domaintrip.Coordinates{
			Latitude:  trip.EndLat,
			Longitude: trip.EndLng,
		},
		Price:      trip.Price,
		Status:     domaintrip.Status(trip.Status),
		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,
	}
	if err := entity.Validate(); err != nil {
		return model.Trip{}, err
	}

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		_, exists, err := u.trips.GetActiveByDriverID(ctx, trip.DriverID)
		if err != nil {
			return fmt.Errorf("check active trip: %w", err)
		}
		if exists {
			return domaintrip.ErrDriverBusy
		}

		if err := u.trips.Create(ctx, trip); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}

		history := model.TripStatusHistory{
			TripID:    trip.ID,
			ToStatus:  trip.Status,
			ChangedAt: time.Now().UTC(),
		}
		if err := u.history.Add(ctx, history); err != nil {
			return fmt.Errorf("create trip status history: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.Trip{}, err
	}

	u.observer.TripCreated(ctx, trip.ID)
	return trip, nil
}
