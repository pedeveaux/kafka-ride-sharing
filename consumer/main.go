package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/joho/godotenv"
	"github.com/pedeveaux/kafkarideshare/events"
	"github.com/pedeveaux/kafkarideshare/logger"
	"github.com/pedeveaux/kafkarideshare/rides_db"
)

func main() {
	logger.Init(slog.LevelInfo, "json")
	slog.Info("Starting ride consumer service...")

	err := godotenv.Load()
	if err != nil {
		slog.Error("No .env file found. Falling back to system environment variables.", "error", err)
	}

	connStr := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	// Initialize the database connection
	if err := rides_db.Init(connStr); err != nil {
		slog.Error("Failed to connect to database", "error", err)
	}
	// Create a context for the database operations
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up channel to listen for interrupt or termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start a goroutine that cancels the context when a signal is received
	go func() {
		<-signalChan
		slog.Info("Received shutdown signal. Exiting gracefully...")
		cancel()
	}()

	// Initialize the Kafka consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "redpanda:9092",
		"group.id":          "ride-consumer-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		logger.Fatal("Failed to create consumer", "error", err)
	}
	defer consumer.Close()

	consumer.Subscribe("ride-events", nil)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled. Exiting...")
			return
		default:
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				var event events.RideEvent
				if err := event.UnmarshalJSON(msg.Value); err != nil {
					slog.Error("Failed to unmarshal message", "event_ID", event.ID, "event type", event.Type, "error", err)
					continue
				}
				// Process the event as needed
				if err := rides_db.InsertRideEvent(ctx, event); err != nil {
					slog.Error("Failed to insert event into database", "error", err)
					continue
				}
				// Log the consumed message details
				slog.Info("Consumed message", "partition", msg.TopicPartition.Partition, "offset", msg.TopicPartition.Offset, "key", string(msg.Key), "trip_id", event.TripID, "type", event.Type)
			} else {
				slog.Error("Consumer error", "error", err)
			}
		}
	}
}
