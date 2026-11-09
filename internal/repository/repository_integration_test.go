//go:build integration

package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domaintrip "github.com/poiuyttq/tripgo/internal/domain/trip"
	"github.com/poiuyttq/tripgo/internal/infra/postgres/transaction"
	"github.com/poiuyttq/tripgo/internal/model"
	"github.com/poiuyttq/tripgo/internal/repository"
)

func TestCreateAndHistoryInsertRollbackTogether(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL must be set; start the tripgo environment and apply migrations")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	tripID := uuid.NewString()
	now := time.Now().UTC()
	trip := model.Trip{
		ID:        tripID,
		UserID:    uuid.NewString(),
		DriverID:  uuid.NewString(),
		StartLat:  59.9,
		StartLng:  30.3,
		EndLat:    59.8,
		EndLng:    30.4,
		Price:     1450,
		Status:    string(domaintrip.StatusActive),
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, "DELETE FROM trips WHERE id = $1", tripID); err != nil {
			t.Errorf("cleanup trip %s: %v", tripID, err)
		}
	})

	tripRepository := repository.NewTripRepository(pool)
	historyRepository := repository.NewTripStatusHistoryRepository()
	txManager := transaction.New(pool)

	err = txManager.Do(ctx, func(ctx context.Context) error {
		if err := tripRepository.Create(ctx, trip); err != nil {
			return err
		}
		return historyRepository.Add(ctx, model.TripStatusHistory{
			TripID:    tripID,
			ToStatus:  "invalid-status",
			ChangedAt: now,
		})
	})
	if err == nil {
		t.Fatal("expected history insert to fail its status constraint")
	}

	if _, err := tripRepository.GetByID(ctx, tripID); !errors.Is(err, domaintrip.ErrTripNotFound) {
		t.Fatalf("trip should be rolled back, GetByID() error = %v", err)
	}

	var tripCount, historyCount int64
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trips WHERE id = $1", tripID).Scan(&tripCount); err != nil {
		t.Fatalf("count trip rows: %v", err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id = $1", tripID).Scan(&historyCount); err != nil {
		t.Fatalf("count history rows: %v", err)
	}
	if tripCount != 0 || historyCount != 0 {
		t.Fatalf("expected transaction rollback to leave no rows; trips=%d history=%d", tripCount, historyCount)
	}
}
