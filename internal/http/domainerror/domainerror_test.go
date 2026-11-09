package domainerror

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	domaintrip "github.com/poiuyttq/tripgo/internal/domain/trip"
)

func TestProblem(t *testing.T) {
	internalErr := errors.New("database password must not be exposed")
	tests := []struct {
		name     string
		err      error
		status   int32
		code     string
		mustHide string
	}{
		{
			name:   "validation error",
			err:    &domaintrip.ValidationError{Field: "price", Reason: "must be non-negative"},
			status: http.StatusBadRequest,
			code:   "invalid_request",
		},
		{
			name:   "driver busy",
			err:    fmt.Errorf("create trip: %w", domaintrip.ErrDriverBusy),
			status: http.StatusConflict,
			code:   "driver_busy",
		},
		{
			name:   "trip completed",
			err:    domaintrip.ErrTripCompleted,
			status: http.StatusConflict,
			code:   "trip_completed",
		},
		{
			name:   "trip not found",
			err:    domaintrip.ErrTripNotFound,
			status: http.StatusNotFound,
			code:   "trip_not_found",
		},
		{
			name:     "internal error hides details",
			err:      internalErr,
			status:   http.StatusInternalServerError,
			code:     "internal_error",
			mustHide: internalErr.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, problem := Problem(tt.err)
			if status != tt.status {
				t.Fatalf("status = %d, want %d", status, tt.status)
			}
			if problem.Status != tt.status {
				t.Fatalf("problem status = %d, want %d", problem.Status, tt.status)
			}
			if problem.Code != tt.code {
				t.Fatalf("problem code = %q, want %q", problem.Code, tt.code)
			}
			if tt.mustHide != "" && problem.Detail != nil && strings.Contains(*problem.Detail, tt.mustHide) {
				t.Fatalf("problem detail exposes internal error: %q", *problem.Detail)
			}
		})
	}
}
