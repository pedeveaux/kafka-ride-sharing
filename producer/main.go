package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/google/uuid"

	"github.com/pedeveaux/kafkarideshare/events"
	"github.com/pedeveaux/kafkarideshare/logger"
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
		events.EventTripCancelled: events.StateCancelled,
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

// IsCancelable checks if the current state allows for cancellation.
// A ride can be cancelled if it is in the Requested or Accepted state.
func (f *FSM) IsCancelable() bool {
	return f.State == events.StateRequested || f.State == events.StateAccepted
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

// generateFare generates a fare based on the distance of the ride.
// It simulates a fare calculation by applying a base fare and a per-kilometer rate.
// The fare is rounded to two decimal places to represent a monetary value.
func generateFare(distance float64) float64 {
	// Generate a random fare based on distance
	// Assuming a base fare of $2.50 and $1.00 per km
	baseFare := 2.50
	perKmRate := 1.00
	return math.Round((baseFare+(perKmRate*distance))*100) / 100 // Round to two decimal places
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
	if !ride.FSM.IsTerminal() && rand.Float64() < 0.1 && ride.FSM.IsCancelable() {
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
	case events.EventRideRequested:
		payload = events.RideRequestedPayload{
			Passenger:       ride.PassengerID,
			PickupLocation:  gofakeit.Street(),
			DropoffLocation: gofakeit.Street(),
		}
	case events.EventRideAccepted:
		payload = events.RideAcceptedPayload{
			DriverID: ride.DriverID,
		}
	case events.EventTripStarted:
		payload = events.RideStartedPayload{}
	case events.EventTripCompleted:
		distance := math.Round(gofakeit.Float64Range(2.0, 25.0)*100) / 100
		fare := generateFare(distance)
		payload = events.RideCompletedPayload{
			EndTime:    now,
			DistanceKM: distance,
			FareUSD:    fare,
		}
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
	logger.Init(slog.LevelInfo, "json")
	slog.Info("Starting ride producer")

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
					slog.Error("Delivery failed", "key", ev.Key, "topic partition", ev.TopicPartition.Partition, "error", ev.TopicPartition.Error)
				} else {
					slog.Info("Delivery successful", "key", ev.Key, "topic partition", ev.TopicPartition.Partition)
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
		slog.Info("Received shutdown signal.")
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
					ID:          uuid.NewString(),
					TripID:      ride.TripID,
					DriverID:    ride.DriverID,
					PassengerID: ride.PassengerID,
					Type:        events.EventRideRequested,
					State:       events.StateRequested,
					Timestamp:   ride.UpdatedAt,
					Payload: events.RideRequestedPayload{
						Passenger:       ride.PassengerID,
						PickupLocation:  gofakeit.Street(),
						DropoffLocation: gofakeit.Street(),
					},
				}
				bytes, err := json.Marshal(evt)
				if err != nil {
					slog.Error("Failed to marshal ride event", "error", err, "tripID", ride.TripID)
					continue
				}
				producer.Produce(&kafka.Message{
					TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
					Key:            []byte(ride.TripID),
					Value:          bytes,
				}, nil)
			}
			// Process each active ride to generate the next event.
			for tripID, ride := range activeRides {
				event, err := getNextEvent(ride)
				if err != nil {
					slog.Error("Ride Error", "error", err, "tripID", tripID)
					delete(activeRides, tripID)
					continue
				}
				if event.Type == "" || event.TripID == "" {
					slog.Warn("Skipping empty event", "tripID", tripID, "eventType", event.Type)
					continue
				}

				bytes, err := json.Marshal(event)
				if err != nil {
					slog.Error("Failed to marshal event", "error", err, "tripID", tripID)
					continue
				}
				producer.Produce(&kafka.Message{
					TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
					Key:            []byte(ride.TripID),
					Value:          bytes,
				}, nil)

				if ride.FSM.IsTerminal() {
					delete(activeRides, tripID)
				}
			}
		// Handle OS signals for graceful shutdown.
		case <-ctx.Done():
			slog.Info("Shutting down via context cancel")
			break loop
		}
	}

	producer.Flush(5000)
}
