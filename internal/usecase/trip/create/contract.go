package create

//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package create_test

import (
	"context"

	"github.com/poiuyttq/tripgo/internal/model"
)

type tripRepository interface {
	GetActiveByDriverID(ctx context.Context, driverID string) (model.Trip, bool, error)
	Create(ctx context.Context, trip model.Trip) error
}

type historyRepository interface {
	Add(ctx context.Context, history model.TripStatusHistory) error
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type tripObserver interface {
	TripCreated(ctx context.Context, tripID string)
}
