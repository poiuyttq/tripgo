package finish

//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package finish_test

import (
	"context"

	"github.com/poiuyttq/tripgo/internal/model"
)

type tripFinisher interface {
	FinishTrip(ctx context.Context, tripID string) error
}

type tripGetter interface {
	GetTrip(ctx context.Context, tripID string) (model.Trip, error)
}
