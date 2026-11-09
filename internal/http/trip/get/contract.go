package get

//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package get_test

import (
	"context"

	"github.com/poiuyttq/tripgo/internal/model"
)

type tripGetter interface {
	GetTrip(ctx context.Context, tripID string) (model.Trip, error)
}
