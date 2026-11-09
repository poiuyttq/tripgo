package finish

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domainhttp "github.com/poiuyttq/tripgo/internal/http/response"
	triphttp "github.com/poiuyttq/tripgo/internal/http/trip"
)

func NewHandler(useCase tripFinisher, getTrip tripGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := chi.URLParam(r, "tripId")
		if _, err := uuid.Parse(tripID); err != nil {
			domainhttp.WriteInvalidRequest(w, "tripId must be a valid UUID", r.URL.Path)
			return
		}

		if err := useCase.FinishTrip(r.Context(), tripID); err != nil {
			domainhttp.WriteDomainError(r.Context(), w, err, r.URL.Path)
			return
		}

		trip, err := getTrip.GetTrip(r.Context(), tripID)
		if err != nil {
			domainhttp.WriteDomainError(r.Context(), w, err, r.URL.Path)
			return
		}
		domainhttp.WriteJSON(w, http.StatusOK, triphttp.ToAPI(trip))
	}
}
