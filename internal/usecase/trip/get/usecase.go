package get

import (
	"context"

	"github.com/poiuyttq/tripgo/internal/model"
)

type UseCase struct {
	trips tripRepository
}

func New(trips tripRepository) *UseCase {
	return &UseCase{trips: trips}
}

func (u *UseCase) GetTrip(ctx context.Context, tripID string) (model.Trip, error) {
	return u.trips.GetByID(ctx, tripID)
}
