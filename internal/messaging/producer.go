package messaging

import (
	"encoding/json"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

// Producer wraps Kafka producer
type Producer struct {
	producer *kafka.Producer
	topic    string
}

// NewProducer initializes a new Kafka producer
func NewProducer(brokers, topic string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return nil, err
	}

	return &Producer{
		producer: p,
		topic:    topic,
	}, nil
}

// Produce sends a message to Kafka
func (p *Producer) Produce(eventType string, data interface{}) error {
	event := struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{
		Type: eventType,
		Data: data,
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Value:          jsonData,
	}, nil)
	if err != nil {
		return err
	}

	return nil
}

// Close gracefully shuts down the producer
func (p *Producer) Close() {
	p.producer.Flush(5000) // Wait up to 5 seconds for messages to be delivered
	p.producer.Close()
}
