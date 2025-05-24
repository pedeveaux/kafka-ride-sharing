package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/google/uuid"

	"github.com/pedeveaux/kafkarideshare/events"
)

// transitions defines the state transitions for the ride lifecycle.
// It maps the current state to a map of valid events and their resulting states.
// The keys of the outer map are the current states, and the values are maps
// where the keys are the events and the values are the resulting states.
var transitions = map[events.RideState]map[events.RideEventType]events.RideState{
	events.StateRequested: {
		events.EventRideAccepted:  events.StateAccepted,
		events.EventTripCancelled: events.StateCancelled,
	},
	events.StateAccepted: {
		events.EventTripStarted:   events.StateInProgress,
		events.EventTripCancelled: events.StateCancelled,
	},
	events.StateInProgress: {
		events.EventTripCompleted: events.StateCompleted,
	},
}

// FSM represents a finite state machine for the ride lifecycle.
// It manages the current state and applies events to transition between states.
// It also provides a method to check if the current state is terminal.
// The FSM is initialized with a starting state and can transition to other states
// based on the defined transitions.
type FSM struct {
	State events.RideState
}

// Apply applies an event to the FSM and transitions to the new state.
// It checks if the event is valid for the current state and updates the state accordingly.
// If the event is not valid, it returns an error.
func (f *FSM) Apply(event events.RideEventType) error {
	valid, ok := transitions[f.State]
	if !ok {
		return fmt.Errorf("no transitions defined for state %s", f.State)
	}
	newState, ok := valid[event]
	if !ok {
		return fmt.Errorf("event %s not valid from state %s", event, f.State)
	}
	f.State = newState
	return nil
}

// IsTerminal checks if the current state is a terminal state.
// Terminal states are those where no further transitions are possible.
// In this case, the terminal states are StateCompleted and StateCancelled.
// The method returns true if the current state is terminal, and false otherwise.
func (f *FSM) IsTerminal() bool {
	return f.State == events.StateCompleted || f.State == events.StateCancelled
}

// Ride represents a ride in the rideshare application.
// It contains the trip ID, driver ID, rider ID, and the FSM for managing the ride's state.
// The ride also has an updated timestamp to track the last time it was modified.
type Ride struct {
	TripID      string
	DriverID    string
	PassengerID string
	FSM         FSM
	UpdatedAt   time.Time
}

// getNextEvent generates the next event for a given ride.
// It simulates the ride lifecycle by applying the next event based on the current state.
// The method also handles the case where a ride is cancelled with a 10% chance.
// If the ride is cancelled, it creates a cancellation event and updates the ride's state.
// The method returns the generated event and any error encountered during the process.
// The event contains the trip ID, driver ID, rider ID, event type, state, timestamp,
// and any additional payload data specific to the event type.
// The payload can be of different types depending on the event type.
// The method uses a random number generator to simulate the cancellation event.
// The ride's updated timestamp is also set to the current time.
func getNextEvent(ride *Ride) (events.RideEvent, error) {
	now := time.Now()

	// Simulate cancellation with 10% chance when not terminal
	if !ride.FSM.IsTerminal() && rand.Float64() < 0.1 {
		err := ride.FSM.Apply(events.EventTripCancelled)
		if err != nil {
			return events.RideEvent{}, err
		}
		evt := events.RideEvent{
			ID:          uuid.NewString(),
			TripID:      ride.TripID,
			DriverID:    ride.DriverID,
			PassengerID: ride.PassengerID,
			Type:        events.EventTripCancelled,
			State:       events.StateCancelled,
			Timestamp:   now,
			Payload: events.RideCancelledPayload{
				CancelledBy: "passenger",
				Reason:      "no_show",
			},
		}
		ride.UpdatedAt = now
		return evt, nil
	}

	var next events.RideEventType
	// Determine the next event based on the current state
	// and the defined transitions
	switch ride.FSM.State {
	case events.StateRequested:
		next = events.EventRideAccepted
	case events.StateAccepted:
		next = events.EventTripStarted
	case events.StateInProgress:
		next = events.EventTripCompleted
	default:
		return events.RideEvent{}, nil // terminal or unknown state
	}

	err := ride.FSM.Apply(next)
	if err != nil {
		return events.RideEvent{}, err
	}

	// Map the event type to the corresponding payload type
	var payload events.RideEventPayload
	switch next {
	case events.EventRideAccepted:
		payload = events.RideAcceptedPayload{}
	case events.EventTripStarted:
		payload = events.RideStartedPayload{}
	case events.EventTripCompleted:
		payload = events.RideCompletedPayload{}
	default:
		payload = nil
	}

	evt := events.RideEvent{
		ID:          uuid.NewString(),
		TripID:      ride.TripID,
		DriverID:    ride.DriverID,
		PassengerID: ride.PassengerID,
		Type:        next,
		State:       ride.FSM.State,
		Timestamp:   now,
		Payload:     payload,
	}

	ride.UpdatedAt = now
	return evt, nil
}

func main() {

	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "redpanda:9092"})
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					log.Printf("Delivered to: %v\n", ev.TopicPartition)
				}
			}
		}
	}()
	// Initialize the ride events topic and active rides map
	// and start the ticker for generating ride events.
	topic := "ride-events"
	activeRides := make(map[string]*Ride)
	ticker := time.NewTicker(1 * time.Second)

	// Set up a context for graceful shutdown and signal handling.
	// This context will be used to cancel the ticker and producer flush on shutdown.
	// It listens for OS signals like SIGINT and SIGTERM to gracefully shut down the producer.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		<-sigchan
		log.Println("Received shutdown signal.")
		cancel()
	}()

loop:
	for {
		select {
		// Generate a new ride request every second if there are fewer than 100 active rides.
		case <-ticker.C:
			if len(activeRides) < 100 {
				tripID := uuid.NewString()
				ride := &Ride{
					TripID:      tripID,
					DriverID:    uuid.NewString(),
					PassengerID: uuid.NewString(),
					FSM:         FSM{State: events.StateRequested},
					UpdatedAt:   time.Now(),
				}
				activeRides[tripID] = ride
				evt := events.RideEvent{
					TripID:      ride.TripID,
					DriverID:    ride.DriverID,
					PassengerID: ride.PassengerID,
					Type:        events.EventRideRequested,
					Timestamp:   ride.UpdatedAt,
				}
				bytes, _ := json.Marshal(evt)
				producer.Produce(&kafka.Message{
					TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
					Value:          bytes,
				}, nil)
			}
			// Process each active ride to generate the next event.
			for tripID, ride := range activeRides {
				event, err := getNextEvent(ride)
				if err != nil {
					log.Printf("Ride %s event error: %v", tripID, err)
					delete(activeRides, tripID)
					continue
				}
				if event.Type == "" {
					continue
				}

				bytes, _ := json.Marshal(event)
				producer.Produce(&kafka.Message{
					TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
					Value:          bytes,
				}, nil)

				if ride.FSM.IsTerminal() {
					delete(activeRides, tripID)
				}
			}
		// Handle OS signals for graceful shutdown.
		case <-ctx.Done():
			log.Println("Shutting down via context cancel")
			break loop
		}
	}

	producer.Flush(5000)
}
