package store

import (
	"testing"

	"github.com/google/uuid"
	"github.com/risbern21/runaway/orders-service/gen/pb"
	"github.com/risbern21/runaway/orders-service/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestNewMemoryStoreImplementsStorer(t *testing.T) {
	ms := NewMemoryStore()
	require.NotNil(t, ms)
	var _ Storer = ms
}

func TestMemoryStoreCreateAndGetOrderByID(t *testing.T) {
	ms := NewMemoryStore()
	orderID := primitive.NewObjectID()
	customerID := uuid.New()
	order := &models.Order{
		ID:         orderID,
		CustomerID: customerID,
		Address:    "Address 1",
		OrderItems: []*pb.OrderItem{{ProductId: 10, Quantity: 2}},
	}

	require.NoError(t, ms.CreateOrder(order))

	got, err := ms.GetOrderByID(orderID)
	require.NoError(t, err)
	assert.Equal(t, orderID, got.ID)
	assert.Equal(t, customerID, got.CustomerID)
	assert.Equal(t, "Address 1", got.Address)
	require.Len(t, got.OrderItems, 1)
	assert.Equal(t, uint64(10), got.OrderItems[0].ProductId)
}

func TestMemoryStoreGetOrderByIDReturnsNotFound(t *testing.T) {
	ms := NewMemoryStore()

	_, err := ms.GetOrderByID(primitive.NewObjectID())
	require.Error(t, err)
	assert.ErrorIs(t, err, mongo.ErrNoDocuments)
}

func TestMemoryStoreGetOrdersByCustomerIDFiltersOrders(t *testing.T) {
	ms := NewMemoryStore()
	customerID := uuid.New()
	otherCustomerID := uuid.New()

	require.NoError(t, ms.CreateOrder(&models.Order{ID: primitive.NewObjectID(), CustomerID: customerID, Address: "A"}))
	require.NoError(t, ms.CreateOrder(&models.Order{ID: primitive.NewObjectID(), CustomerID: customerID, Address: "B"}))
	require.NoError(t, ms.CreateOrder(&models.Order{ID: primitive.NewObjectID(), CustomerID: otherCustomerID, Address: "C"}))

	orders, err := ms.GetOrdersByCustomerID(customerID)
	require.NoError(t, err)
	require.Len(t, orders, 2)
}

func TestMemoryStoreUpdateDeliveryAddress(t *testing.T) {
	ms := NewMemoryStore()
	orderID := primitive.NewObjectID()
	require.NoError(t, ms.CreateOrder(&models.Order{ID: orderID, CustomerID: uuid.New(), Address: "Old"}))

	require.NoError(t, ms.UpdateDeliveryAddress(orderID, "New"))

	got, err := ms.GetOrderByID(orderID)
	require.NoError(t, err)
	assert.Equal(t, "New", got.Address)
}

func TestMemoryStoreUpdatePaymentStatus(t *testing.T) {
	ms := NewMemoryStore()
	orderID := primitive.NewObjectID()
	require.NoError(t, ms.CreateOrder(&models.Order{ID: orderID, CustomerID: uuid.New()}))

	require.NoError(t, ms.UpdatePaymentStatus(orderID, true))

	got, err := ms.GetOrderByID(orderID)
	require.NoError(t, err)
	assert.True(t, got.IsPaid)
}

func TestMemoryStoreUpdatePaymentStatusReturnsNotFound(t *testing.T) {
	ms := NewMemoryStore()

	err := ms.UpdatePaymentStatus(primitive.NewObjectID(), true)
	require.Error(t, err)
	assert.EqualError(t, err, "no order found")
}

func TestMemoryStoreUpdateDeliveryStatus(t *testing.T) {
	ms := NewMemoryStore()
	orderID := primitive.NewObjectID()
	require.NoError(t, ms.CreateOrder(&models.Order{ID: orderID, CustomerID: uuid.New()}))

	require.NoError(t, ms.UpdateDeliveryStatus(orderID, true))

	got, err := ms.GetOrderByID(orderID)
	require.NoError(t, err)
	assert.True(t, got.IsDelivered)
}

func TestMemoryStoreCancelOrder(t *testing.T) {
	ms := NewMemoryStore()
	orderID := primitive.NewObjectID()
	require.NoError(t, ms.CreateOrder(&models.Order{ID: orderID, CustomerID: uuid.New()}))

	require.NoError(t, ms.CancelOrder(orderID))

	_, err := ms.GetOrderByID(orderID)
	require.Error(t, err)
	assert.ErrorIs(t, err, mongo.ErrNoDocuments)
}

func TestMemoryStoreCancelOrderReturnsNotFound(t *testing.T) {
	ms := NewMemoryStore()

	err := ms.CancelOrder(primitive.NewObjectID())
	require.Error(t, err)
	assert.EqualError(t, err, "no order found")
}

func TestMemoryStoreStoresCopies(t *testing.T) {
	ms := NewMemoryStore()
	orderID := primitive.NewObjectID()
	order := &models.Order{
		ID:         orderID,
		CustomerID: uuid.New(),
		Address:    "Original",
		OrderItems: []*pb.OrderItem{{ProductId: 1, Quantity: 1}},
	}

	require.NoError(t, ms.CreateOrder(order))
	order.Address = "Mutated"
	order.OrderItems[0].Quantity = 99

	got, err := ms.GetOrderByID(orderID)
	require.NoError(t, err)
	assert.Equal(t, "Original", got.Address)
	assert.Equal(t, uint32(1), got.OrderItems[0].Quantity)
}
