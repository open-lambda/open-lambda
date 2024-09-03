# Sample Kafka code for Go using github.com/segmentio/kafka-go

`Dockerfile`
```
FROM golang:1.18

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o producer producer.go
RUN go build -o consumer consumer.go

CMD ["./producer"]
```

`docker-compose.yml`
```
version: '3.8'

services:
  zookeeper:
    image: confluentinc/cp-zookeeper:7.5.0
    ports:
      - "2181:2181"
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000

  kafka:
    image: confluentinc/cp-kafka:7.5.0
    ports:
      - "9092:9092"
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_BROKER_RESTART_TIMEOUT_MS: 5000  # Add a restart timeout for Kafka
      KAFKA_RETRY_BACKOFF_MS: 1000  # Add retry backoff to give Zookeeper time to start
    depends_on:
      - zookeeper

  producer:
    build:
      context: .
      dockerfile: Dockerfile
    depends_on:
      - kafka
    environment:
      - KAFKA_BROKER=kafka:9092
    command: sh -c "sleep 10 && ./producer"  # Add a 10-second delay before starting

  consumer:
    build:
      context: .
      dockerfile: Dockerfile
    depends_on:
      - kafka
    environment:
      - KAFKA_BROKER=kafka:9092
    command: sh -c "sleep 10 && ./consumer"  # Add a 10-second delay before starting
```

`producer.go`
```
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	// Define the topic and partition
	topic := "example-topic"
	partition := 0

	// Create a new Kafka connection to the broker
	conn, err := kafka.DialLeader(context.Background(), "tcp", "kafka:9092", topic, partition)
	if err != nil {
		log.Fatal("Failed to connect to Kafka:", err)
	}
	defer conn.Close()

	// Create the topic with specific configuration
	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		log.Fatal("Failed to create topic:", err)
	}
	fmt.Println("Topic created successfully")

	// Initialize the writer
	writer := kafka.Writer{
		Addr:     kafka.TCP("kafka:9092"),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	for i := 0; i < 5; i++ {
		// Prepare the message to be sent
		msg := kafka.Message{
			Key:   []byte("example-key"),
			Value: []byte(fmt.Sprintf("Hello world %d!", i)),
		}

		// Write the message
		err = writer.WriteMessages(context.Background(), msg)
		if err != nil {
			log.Fatal("Failed to write message:", err)
		}

		fmt.Println("Message written successfully")

		time.Sleep(1 * time.Second)
	}
}
```

`consumer.go`
```
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

func main() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"kafka:9092"}, // Use service name defined in docker-compose.yml
		Topic:   "example-topic",
		GroupID: "example-group",
	})

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatal("Failed to read message:", err)
		}

		fmt.Printf("Received message: %s\n", string(msg.Value))
	}

	if err := reader.Close(); err != nil {
		log.Fatal("Failed to close reader:", err)
	}
}
```