package get

//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package get_test

import (
	"context"

	"github.com/poiuyttq/tripgo/internal/model"
)

type tripRepository interface {
	GetByID(ctx context.Context, id string) (model.Trip, error)
}
