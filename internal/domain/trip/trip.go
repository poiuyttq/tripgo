package trip

import "time"

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID         string
	UserID     string
	DriverID   string
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
}

func (t Trip) Validate() error {
	if t.ID == "" {
		return &ValidationError{Field: "id", Reason: requiredFieldReason}
	}
	if t.UserID == "" {
		return &ValidationError{Field: "user_id", Reason: requiredFieldReason}
	}
	if t.DriverID == "" {
		return &ValidationError{Field: "driver_id", Reason: requiredFieldReason}
	}
	if t.StartedAt.IsZero() {
		return &ValidationError{Field: "started_at", Reason: requiredFieldReason}
	}
	if t.Price < 0 {
		return &ValidationError{Field: "price", Reason: "must be greater than or equal to 0"}
	}
	if err := validateCoordinates("start_point", t.StartPoint); err != nil {
		return err
	}
	if err := validateCoordinates("end_point", t.EndPoint); err != nil {
		return err
	}

	switch t.Status {
	case StatusActive:
		if t.FinishedAt != nil {
			return &ValidationError{Field: "finished_at", Reason: "must be absent for an active trip"}
		}
	case StatusCompleted:
		if t.FinishedAt == nil {
			return &ValidationError{Field: "finished_at", Reason: "is required for a completed trip"}
		}
	default:
		return &ValidationError{Field: "status", Reason: "must be active or completed"}
	}
	return nil
}

func validateCoordinates(field string, coordinates Coordinates) error {
	if coordinates.Latitude < -90 || coordinates.Latitude > 90 {
		return &ValidationError{Field: field + ".latitude", Reason: "must be between -90 and 90"}
	}
	if coordinates.Longitude < -180 || coordinates.Longitude > 180 {
		return &ValidationError{Field: field + ".longitude", Reason: "must be between -180 and 180"}
	}
	return nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}

	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}
