package kafka

import "github.com/segmentio/kafka-go"

type ProducerConfig struct {
	Brokers      []string
	Topic        string
	BatchSize    int
	BatchTimeout int
	Async        bool
	RequiredAcks kafka.RequiredAcks
}
