package response

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/poiuyttq/tripgo/internal/http/domainerror"
	"github.com/poiuyttq/tripgo/internal/requesttrace"
)

func WriteDomainError(ctx context.Context, w http.ResponseWriter, err error, instance string) {
	requesttrace.SetError(ctx, err)
	status, problem := domainerror.Problem(err)
	problem.Instance = &instance
	WriteProblem(w, int(status), problem)
}

func WriteInvalidRequest(w http.ResponseWriter, detail, instance string) {
	problem := domainProblem(int32(http.StatusBadRequest), "invalid_request", "Invalid request", detail, instance)
	WriteProblem(w, http.StatusBadRequest, problem)
}

func WriteProblem(w http.ResponseWriter, status int, problem any) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
