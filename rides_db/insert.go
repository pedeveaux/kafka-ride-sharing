package rides_db

import (
    "context"
    "encoding/json"
	"github.com/pedeveaux/kafkarideshare/events"
)

func InsertRideEvent(ctx context.Context, e events.RideEvent) error {
    payloadBytes, err := json.Marshal(e.Payload)
    if err != nil {
        return err
    }

    _, err = DB.ExecContext(ctx, `
        INSERT INTO ride_events 
        (id, trip_id, event_type, event_state, event_time, driver_id, passenger_id, payload)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (trip_id, event_type) DO NOTHING
    `, e.ID, e.TripID, e.Type, e.State, e.Timestamp, e.DriverID, e.PassengerID, payloadBytes)

    return err
}