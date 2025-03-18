package messaging

import (
	"encoding/json"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaProducer struct {
	producer *kafka.Producer
	topic    string
}

func NewKafkaProducer(brokers, topic string) (*KafkaProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		logger.Warnf("Failed to create Kafka producer: %v", err)
		return nil, err
	}

	logger.Infof("Kafka producer initialized for topic: %s", topic)
	return &KafkaProducer{
		producer: p,
		topic:    topic,
	}, nil
}

func (p *KafkaProducer) Produce(eventType string, data interface{}) error {
	event := struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{
		Type: eventType,
		Data: data,
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		logger.Warnf("Failed to marshal event: %v", err)
		return err
	}

	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.topic, Partition: kafka.PartitionAny},
		Value:          jsonData,
	}, nil)
	if err != nil {
		logger.Warnf("Failed to produce message: %v", err)
		return err
	}

	logger.Infof("Message produced to topic %s: %s", p.topic, jsonData)
	return nil
}

func (p *KafkaProducer) Close() {
	p.producer.Flush(5000)
	p.producer.Close()
	logger.Info("Kafka producer closed gracefully")
}
