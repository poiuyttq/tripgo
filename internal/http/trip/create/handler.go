package create

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	api "github.com/poiuyttq/tripgo/internal/generated"
	domainhttp "github.com/poiuyttq/tripgo/internal/http/response"
	triphttp "github.com/poiuyttq/tripgo/internal/http/trip"
	"github.com/poiuyttq/tripgo/internal/model"
)

type tripRequest struct {
	UserID     api.TripId        `json:"user_id"`
	DriverID   api.TripId        `json:"driver_id"`
	StartPoint *coordinatesInput `json:"start_point"`
	EndPoint   *coordinatesInput `json:"end_point"`
	Price      *int64            `json:"price"`
}

type coordinatesInput struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func NewHandler(useCase tripCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request tripRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			domainhttp.WriteInvalidRequest(w, "Request body is invalid", r.URL.Path)
			return
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			domainhttp.WriteInvalidRequest(w, "Request body must contain exactly one JSON value", r.URL.Path)
			return
		}
		if err := validate(request); err != nil {
			domainhttp.WriteInvalidRequest(w, err.Error(), r.URL.Path)
			return
		}

		now := time.Now().UTC()
		trip := model.Trip{
			ID:        uuid.NewString(),
			UserID:    request.UserID.String(),
			DriverID:  request.DriverID.String(),
			StartLat:  *request.StartPoint.Latitude,
			StartLng:  *request.StartPoint.Longitude,
			EndLat:    *request.EndPoint.Latitude,
			EndLng:    *request.EndPoint.Longitude,
			Price:     *request.Price,
			Status:    "active",
			StartedAt: now,
			CreatedAt: now,
			UpdatedAt: now,
		}

		created, err := useCase.CreateTrip(r.Context(), trip)
		if err != nil {
			domainhttp.WriteDomainError(r.Context(), w, err, r.URL.Path)
			return
		}

		w.Header().Set("Location", "/api/v1/trips/"+created.ID)
		domainhttp.WriteJSON(w, http.StatusCreated, triphttp.ToAPI(created))
	}
}

func validate(request tripRequest) error {
	if request.UserID == uuid.Nil || request.DriverID == uuid.Nil {
		return fmt.Errorf("user_id and driver_id are required")
	}
	if request.StartPoint == nil || request.StartPoint.Latitude == nil || request.StartPoint.Longitude == nil {
		return fmt.Errorf("start_point latitude and longitude are required")
	}
	if request.EndPoint == nil || request.EndPoint.Latitude == nil || request.EndPoint.Longitude == nil {
		return fmt.Errorf("end_point latitude and longitude are required")
	}
	if request.Price == nil {
		return fmt.Errorf("price is required")
	}
	if *request.StartPoint.Latitude < -90 || *request.StartPoint.Latitude > 90 {
		return fmt.Errorf("start_point.latitude must be between -90 and 90")
	}
	if *request.StartPoint.Longitude < -180 || *request.StartPoint.Longitude > 180 {
		return fmt.Errorf("start_point.longitude must be between -180 and 180")
	}
	if *request.EndPoint.Latitude < -90 || *request.EndPoint.Latitude > 90 {
		return fmt.Errorf("end_point.latitude must be between -90 and 90")
	}
	if *request.EndPoint.Longitude < -180 || *request.EndPoint.Longitude > 180 {
		return fmt.Errorf("end_point.longitude must be between -180 and 180")
	}
	if *request.Price < 0 {
		return fmt.Errorf("price must be greater than or equal to 0")
	}
	return nil
}
