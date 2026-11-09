package get_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/poiuyttq/tripgo/internal/domain/trip"
	"github.com/poiuyttq/tripgo/internal/model"
	get "github.com/poiuyttq/tripgo/internal/usecase/trip/get"
)

func TestGetTrip(t *testing.T) {
	tripID := "11111111-1111-4111-8111-111111111111"
	want := model.Trip{ID: tripID, Status: "active"}
	repositoryErr := trip.ErrTripNotFound

	tests := []struct {
		name    string
		result  model.Trip
		err     error
		want    model.Trip
		wantErr error
	}{
		{name: "found", result: want, want: want},
		{name: "not found", err: repositoryErr, wantErr: repositoryErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := NewMocktripRepository(controller)
			repository.EXPECT().GetByID(gomock.Any(), tripID).Return(tt.result, tt.err)

			got, err := get.New(repository).GetTrip(context.Background(), tripID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetTrip() error = %v, want %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetTrip() trip = %+v, want %+v", got, tt.want)
			}
		})
	}
}
