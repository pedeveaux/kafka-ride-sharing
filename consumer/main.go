package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func insertRideEvent(db * sql.DB, eventType string, payload []byte) error {
	query := "INSERT INTO ride_events (event_type, payload) VALUES ($1, $2)"
	_, err := db.Exec(query, eventType, payload)
	return err
}


func main() {
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
		msg, err := consumer.ReadMessage(-1)
		if err == nil {
			fmt.Printf("Received: %s\n", string(msg.Value))
		} else {
			log.Printf("Consumer error: %v\n", err)
		}
	}
}