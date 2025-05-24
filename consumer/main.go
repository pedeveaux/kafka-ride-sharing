package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/joho/godotenv"
	"github.com/pedeveaux/kafkarideshare/events"
	"github.com/pedeveaux/kafkarideshare/rides_db"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Falling back to system environment variables.")
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
		log.Fatalf("Failed to connect to database: %v", err)
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
		fmt.Println("Received shutdown signal. Exiting gracefully...")
		cancel()
	}()

	// Initialize the Kafka consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "redpanda:9092",
		"group.id":          "ride-consumer-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	consumer.Subscribe("ride-events", nil)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled. Exiting...")
			return
		default:
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				var event events.RideEvent
				if err := event.UnmarshalJSON(msg.Value); err != nil {
					log.Printf("Failed to unmarshal message: %v", err)
					continue
				}
				if err := rides_db.InsertRideEvent(ctx, event); err != nil {
					log.Printf("Failed to insert event into database: %v", err)
					continue
				}
				// Process the event as needed
				fmt.Printf("Received: %s\n", string(msg.Value))
			} else {
				log.Printf("Consumer error: %v\n", err)
			}
		}
	}
}
