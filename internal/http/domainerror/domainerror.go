package domainerror

import (
	"errors"
	"net/http"

	domaintrip "github.com/poiuyttq/tripgo/internal/domain/trip"
	api "github.com/poiuyttq/tripgo/internal/generated"
)

func Problem(err error) (int32, api.Problem) {
	status := int32(http.StatusInternalServerError)
	code := "internal_error"
	title := "Internal Server Error"
	detail := "Internal server error"
	var validationErr *domaintrip.ValidationError

	switch {
	case errors.As(err, &validationErr):
		status = int32(http.StatusBadRequest)
		code = "invalid_request"
		title = "Invalid request"
		detail = validationErr.Error()
	case errors.Is(err, domaintrip.ErrDriverBusy):
		status = int32(http.StatusConflict)
		code = "driver_busy"
		title = "Driver busy"
		detail = "Driver already has an active trip"
	case errors.Is(err, domaintrip.ErrTripCompleted):
		status = int32(http.StatusConflict)
		code = "trip_completed"
		title = "Trip completed"
		detail = "Operation is not allowed for a completed trip"
	case errors.Is(err, domaintrip.ErrTripNotFound):
		status = int32(http.StatusNotFound)
		code = "trip_not_found"
		title = "Trip not found"
		detail = "Trip was not found"
	}

	return status, api.Problem{
		Type:   "https://tripgo.example/problems/" + code,
		Title:  title,
		Status: status,
		Code:   code,
		Detail: &detail,
	}
}
