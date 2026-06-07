package kafka

import (
	"testing"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProducerInitializesWriter(t *testing.T) {
	p := NewProducer(ProducerConfig{
		Brokers:      []string{"localhost:9092"},
		Topic:        TopicOrderCreated.String(),
		BatchSize:    10,
		BatchTimeout: 50,
		Async:        true,
		RequiredAcks: kafkago.RequireNone,
	})
	require.NotNil(t, p)
	require.NotNil(t, p.w)
	assert.NoError(t, p.Close())
}

func TestTopicStringReturnsUnderlyingValue(t *testing.T) {
	assert.Equal(t, "orders.created", TopicOrderCreated.String())
}
