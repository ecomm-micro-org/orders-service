package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	w *kafka.Writer
}

func NewProducer(pCfg ProducerConfig) *Producer {
	return &Producer{
		w: kafka.NewWriter(
			kafka.WriterConfig{
				Brokers:      pCfg.Brokers,
				Topic:        pCfg.Topic,
				Balancer:     &kafka.LeastBytes{},
				BatchSize:    pCfg.BatchSize,
				BatchTimeout: time.Duration(pCfg.BatchTimeout) * time.Millisecond,
				RequiredAcks: int(pCfg.RequiredAcks),
				Async:        pCfg.Async,
			},
		),
	}
}

func (p *Producer) Produce(ctx context.Context, key, val []byte) error {
	return p.w.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: val,
	})
}

func (p *Producer) Close() error {
	return p.w.Close()
}
