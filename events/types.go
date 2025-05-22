package events

import (
	"encoding/json"
	"time"
)

// RideEventPayload is a marker interface for all payloads
type RideEventPayload interface {
	isPayload()
}

// RideRequestedPayload holds data for when a ride is requested
type RideRequestedPayload struct {
	Passenger       string `json:"passenger"`
	PickupLocation  string `json:"pickup_location"`
	DropoffLocation string `json:"dropoff_location"`
}

func (RideRequestedPayload) isPayload() {}

// RideAcceptedPayload holds data for when a ride is accepted
type RideAcceptedPayload struct {
	DriverID string `json:"driver_id"`
}

func (RideAcceptedPayload) isPayload() {}

// RideStartedPayload holds data for when a ride begins
type RideStartedPayload struct {
	StartTime time.Time `json:"start_time"`
}

func (RideStartedPayload) isPayload() {}

// RideCompletedPayload holds data for when a ride is completed
type RideCompletedPayload struct {
	EndTime    time.Time `json:"end_time"`
	DistanceKM float64   `json:"distance_km"`
	FareUSD    float64   `json:"fare_usd"`
}

func (RideCompletedPayload) isPayload() {}

// RideCancelledPayload holds data for when a ride is cancelled
type RideCancelledPayload struct {
	CancelledBy string `json:"cancelled_by"` // "passenger" or "driver"
	Reason      string `json:"reason,omitempty"`
}

func (RideCancelledPayload) isPayload() {}

// RideEventType is a string-based enum for Kafka event types.
type RideEventType string

const (
	EventRideRequested RideEventType = "REQUESTED"
	EventRideAccepted  RideEventType = "ACCEPTED"
	EventTripStarted   RideEventType = "STARTED"
	EventTripCompleted RideEventType = "COMPLETED"
	EventTripCancelled RideEventType = "CANCELLED"
)

// RideState represents the state of a ride in the FSM.
type RideState string

const (
	StateNew        RideState = "NEW"
	StateRequested  RideState = "REQUESTED"
	StateAccepted   RideState = "ACCEPTED"
	StateInProgress RideState = "IN_PROGRESS"
	StateCompleted  RideState = "COMPLETED"
	StateCancelled  RideState = "CANCELLED"
)

// RideEvent represents a single state transition in the ride lifecycle.
type RideEvent struct {
	ID        string           `json:"id"`
	TripID    string           `json:"trip_id"`
	Type      RideEventType    `json:"type"`
	Timestamp time.Time        `json:"timestamp"`
	State     RideState        `json:"state"`
	DriverID  string           `json:"driver_id,omitempty"`
	RiderID   string           `json:"rider_id,omitempty"`
	Payload   RideEventPayload `json:"payload,omitempty"` // use type switches on deserialization
}

// UnmarshalJSON customizes the unmarshalling of RideEvent to handle the Payload field.
func (e *RideEvent) UnmarshalJSON(data []byte) error {
	type Alias RideEvent // Prevent recursion
	aux := &struct {
		Payload json.RawMessage `json:"payload"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch e.Type {
	case EventRideRequested:
		var p RideRequestedPayload
		if err := json.Unmarshal(aux.Payload, &p); err != nil {
			return err
		}
		e.Payload = p
	case EventRideAccepted:
		var p RideAcceptedPayload
		if err := json.Unmarshal(aux.Payload, &p); err != nil {
			return err
		}
		e.Payload = p
	case EventTripStarted:
		var p RideStartedPayload
		if err := json.Unmarshal(aux.Payload, &p); err != nil {
			return err
		}
		e.Payload = p
	case EventTripCompleted:
		var p RideCompletedPayload
		if err := json.Unmarshal(aux.Payload, &p); err != nil {
			return err
		}
		e.Payload = p
	case EventTripCancelled:
		var p RideCancelledPayload
		if err := json.Unmarshal(aux.Payload, &p); err != nil {
			return err
		}
		e.Payload = p
	default:
		// Unknown type, leave as nil or handle as needed
		e.Payload = nil
	}
	return nil
}
