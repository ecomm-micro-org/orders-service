package kafka

import (
	"sync"
)

var (
	defaultProducer *Producer
	once            sync.Once
)

func Init(brokers []string) {
	once.Do(func() {
		defaultProducer = NewProducer()
	})
}

func Client() *Producer {
	return defaultProducer
}

func Close() {
	if defaultProducer != nil {
		defaultProducer.Close()
	}
}
