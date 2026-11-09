package create

//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package create_test

import (
	"context"

	"github.com/poiuyttq/tripgo/internal/model"
)

type tripCreator interface {
	CreateTrip(ctx context.Context, trip model.Trip) (model.Trip, error)
}
