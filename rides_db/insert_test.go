package rides_db

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/pedeveaux/kafkarideshare/events"
)

func TestInsertRideEvent_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	DB = db // override global for test

	evt := events.RideEvent{
		ID:          uuid.New().String(),
		TripID:      "trip-123",
		Type:        "trip_started",
		State:       "in_progress",
		Timestamp:   time.Now(),
		DriverID:    "driver-1",
		PassengerID: "rider-1",
		Payload: events.RideStartedPayload{StartTime: time.Now(),
		},
	}

	mock.ExpectExec("INSERT INTO ride_events").
		WithArgs(sqlmock.AnyArg(), "trip-123", "trip_started", "in_progress", sqlmock.AnyArg(), "driver-1", "rider-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	if err := InsertRideEvent(ctx, evt); err != nil {
		t.Errorf("InsertRideEvent failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
