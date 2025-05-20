package main

import (
	"fmt"
	"log"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "redpanda:9092"})
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	topic := "ride-events"

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

	for {
		// Simulate a ride event
		event := fmt.Sprintf(`{"trip_id":"%s", "driver_id":"%s", "city":"%s", "timestamp":"%s"}`,
			gofakeit.UUID(), gofakeit.UUID(), gofakeit.City(), time.Now().Format(time.RFC3339))

		err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(event),
			Key:            []byte(gofakeit.UUID()), // simulate keyed partitioning
		}, nil)

		if err != nil {
			log.Printf("Error producing message: %s", err)
		}

		time.Sleep(1 * time.Second)
	}
}