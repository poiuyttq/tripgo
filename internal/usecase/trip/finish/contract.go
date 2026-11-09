package finish

//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package finish_test

import (
	"context"
	"time"

	"github.com/poiuyttq/tripgo/internal/model"
)

type tripRepository interface {
	GetByIDForUpdate(ctx context.Context, id string) (model.Trip, error)
	Finish(ctx context.Context, tripID string, finishedAt time.Time) error
}

type historyRepository interface {
	Add(ctx context.Context, history model.TripStatusHistory) error
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type tripObserver interface {
	TripCompleted(ctx context.Context, tripID string)
}
