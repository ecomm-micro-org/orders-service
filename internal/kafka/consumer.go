package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Consumer struct {
	consumer *kafka.Reader
}

func NewConsumer(topic Topic) *Consumer {
	return &Consumer{
		consumer: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  config.Config().Brokers,
			Topic:    topic.String(),
			GroupID:  "orders-service",
			MinBytes: 10e3,
			MaxBytes: 10e6,
			MaxWait:  500 * time.Millisecond,
		}),
	}
}

func (c *Consumer) ConsumeTopicPayments(fn func(primitive.ObjectID, bool) error, errs chan error) {
	if c.consumer.Config().Topic != TopicPayments.String() {
		errs <- fmt.Errorf("ConsumeTopicPayments requires topic to be set to payments topic")
		return
	}

	ctx := context.Background()
	for {
		msg, err := c.consumer.FetchMessage(ctx)
		if err != nil {
			errs <- err
			continue
		}

		if err := c.consumer.CommitMessages(ctx, msg); err != nil {
			errs <- err
			continue
		}

		var data struct {
			ID     string `json:"order_id"`
			IsPaid bool   `json:"is_paid"`
		}
		if err := json.Unmarshal(msg.Value, &data); err != nil {
			errs <- err
			continue
		}

		id, err := primitive.ObjectIDFromHex(data.ID)
		if err != nil {
			errs <- err
			continue
		}
		go func(id primitive.ObjectID, isPaid bool) {
			if err := fn(id, isPaid); err != nil {
				errs <- err
			}
		}(id, data.IsPaid)
	}
}

func (c *Consumer) ConsumeTopicDeliveries(fn func(primitive.ObjectID, bool) error, errs chan error) {
	if c.consumer.Config().Topic != TopicDeliveries.String() {
		errs <- fmt.Errorf("ConsumeTopicPayments requires topic to be set to deliveries topic")
		return
	}

	ctx := context.Background()
	for {
		msg, err := c.consumer.FetchMessage(ctx)
		if err != nil {
			errs <- err
			continue
		}

		if err := c.consumer.CommitMessages(ctx, msg); err != nil {
			errs <- err
			continue
		}

		var data struct {
			ID          string `json:"order_id"`
			IsDelivered bool   `json:"is_delivered"`
		}
		if err := json.Unmarshal(msg.Value, &data); err != nil {
			errs <- err
			continue
		}

		id, err := primitive.ObjectIDFromHex(data.ID)
		if err != nil {
			errs <- err
			continue
		}
		go func(id primitive.ObjectID, isDelivered bool) {
			if err := fn(id, isDelivered); err != nil {
				errs <- err
			}
		}(id, data.IsDelivered)
	}
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}
