package events

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestRideEventPayloadImplementations(t *testing.T) {
	var _ RideEventPayload = RideRequestedPayload{}
	var _ RideEventPayload = RideAcceptedPayload{}
	var _ RideEventPayload = RideStartedPayload{}
	var _ RideEventPayload = RideCompletedPayload{}
	var _ RideEventPayload = RideCancelledPayload{}
}

func TestRideStatesAndEventsConstants(t *testing.T) {
	if StateNew == "" || StateRequested == "" || StateAccepted == "" ||
		StateInProgress == "" || StateCompleted == "" || StateCancelled == "" {
		t.Error("one or more RideState constants are empty")
	}
	if EventRideRequested == "" || EventRideAccepted == "" ||
		EventTripStarted == "" || EventTripCompleted == "" || EventTripCancelled == "" {
		t.Error("one or more RideEventType constants are empty")
	}
}

func TestRideEventJSONMarshalling_AllTypes(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name    string
		event   RideEvent
		wantTyp interface{}
	}{
		{
			name: "Requested",
			event: RideEvent{
				ID:          "id1",
				TripID:      "trip1",
				Type:        EventRideRequested,
				Timestamp:   now,
				State:       StateRequested,
				PassengerID: "rider-1",
				Payload:     RideRequestedPayload{Passenger: "rider-1", PickupLocation: "A", DropoffLocation: "B"},
			},
			wantTyp: RideRequestedPayload{},
		},
		{
			name: "Accepted",
			event: RideEvent{
				ID:        "id2",
				TripID:    "trip2",
				Type:      EventRideAccepted,
				Timestamp: now,
				State:     StateAccepted,
				DriverID:  "driver-1",
				Payload:   RideAcceptedPayload{DriverID: "driver-1"},
			},
			wantTyp: RideAcceptedPayload{},
		},
		{
			name: "Started",
			event: RideEvent{
				ID:        "id3",
				TripID:    "trip3",
				Type:      EventTripStarted,
				Timestamp: now,
				State:     StateInProgress,
				Payload:   RideStartedPayload{StartTime: now},
			},
			wantTyp: RideStartedPayload{},
		},
		{
			name: "Completed",
			event: RideEvent{
				ID:        "id4",
				TripID:    "trip4",
				Type:      EventTripCompleted,
				Timestamp: now,
				State:     StateCompleted,
				Payload:   RideCompletedPayload{EndTime: now, DistanceKM: 10.5, FareUSD: 25.0},
			},
			wantTyp: RideCompletedPayload{},
		},
		{
			name: "Cancelled",
			event: RideEvent{
				ID:        "id5",
				TripID:    "trip5",
				Type:      EventTripCancelled,
				Timestamp: now,
				State:     StateCancelled,
				Payload:   RideCancelledPayload{CancelledBy: "driver", Reason: "no show"},
			},
			wantTyp: RideCancelledPayload{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.event)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			var unmarshalled RideEvent
			err = json.Unmarshal(data, &unmarshalled)
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if unmarshalled.Type != tc.event.Type {
				t.Errorf("expected Type %s, got %s", tc.event.Type, unmarshalled.Type)
			}
			// Check payload type
			if _, ok := unmarshalled.Payload.(interface{ isPayload() }); !ok {
				t.Errorf("Payload does not implement isPayload for %s", tc.name)
			}
			// Check concrete type
			if fmt.Sprintf("%T", unmarshalled.Payload) != fmt.Sprintf("%T", tc.wantTyp) {
				t.Errorf("expected payload type %T, got %T", tc.wantTyp, unmarshalled.Payload)
			}
		})
	}
}
