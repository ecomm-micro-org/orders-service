package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewOrderReturnsEmptyOrder(t *testing.T) {
	order := NewOrder()
	require.NotNil(t, order)
	assert.Equal(t, primitive.NilObjectID, order.ID)
	assert.Equal(t, uuid.Nil, order.CustomerID)
	assert.Empty(t, order.Address)
	assert.Empty(t, order.OrderItems)
	assert.False(t, order.IsPaid)
	assert.False(t, order.IsDelivered)
}
