package trip

import (
	"github.com/google/uuid"

	api "github.com/poiuyttq/tripgo/internal/generated"
	"github.com/poiuyttq/tripgo/internal/model"
)

func ToAPI(trip model.Trip) api.Trip {
	tripID, err := uuid.Parse(trip.ID)
	if err != nil {
		tripID = uuid.Nil
	}
	userID, err := uuid.Parse(trip.UserID)
	if err != nil {
		userID = uuid.Nil
	}
	driverID, err := uuid.Parse(trip.DriverID)
	if err != nil {
		driverID = uuid.Nil
	}

	return api.Trip{
		Id:             tripID,
		UserId:         userID,
		DriverId:       driverID,
		StartPoint:     api.Coordinates{Latitude: trip.StartLat, Longitude: trip.StartLng},
		EndPoint:       api.Coordinates{Latitude: trip.EndLat, Longitude: trip.EndLng},
		Price:          trip.Price,
		Status:         api.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: nil,
	}
}
