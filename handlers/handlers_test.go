package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/ecomm-micro-org/orders-service/internal/config"
	"github.com/ecomm-micro-org/orders-service/models"
	"github.com/ecomm-micro-org/orders-service/pb"
	"github.com/ecomm-micro-org/orders-service/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type handlerTestStore struct {
	createOrderFn           func(*models.Order) error
	getOrderByIDFn          func(primitive.ObjectID) (*models.Order, error)
	getOrdersByCustomerIDFn func(uuid.UUID) ([]models.Order, error)
	updateDeliveryAddressFn func(primitive.ObjectID, string) error
	updatePaymentStatusFn   func(primitive.ObjectID, bool) error
	updateDeliveryStatusFn  func(primitive.ObjectID, bool) error
	cancelOrderFn           func(primitive.ObjectID) error
}

func (m handlerTestStore) CreateOrder(o *models.Order) error {
	if m.createOrderFn != nil {
		return m.createOrderFn(o)
	}
	return nil
}

func (m handlerTestStore) GetOrderByID(id primitive.ObjectID) (*models.Order, error) {
	if m.getOrderByIDFn != nil {
		return m.getOrderByIDFn(id)
	}
	return nil, nil
}

func (m handlerTestStore) GetOrdersByCustomerID(customerID uuid.UUID) ([]models.Order, error) {
	if m.getOrdersByCustomerIDFn != nil {
		return m.getOrdersByCustomerIDFn(customerID)
	}
	return nil, nil
}

func (m handlerTestStore) UpdateDeliveryAddress(id primitive.ObjectID, address string) error {
	if m.updateDeliveryAddressFn != nil {
		return m.updateDeliveryAddressFn(id, address)
	}
	return nil
}

func (m handlerTestStore) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	if m.updatePaymentStatusFn != nil {
		return m.updatePaymentStatusFn(id, isPaid)
	}
	return nil
}

func (m handlerTestStore) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	if m.updateDeliveryStatusFn != nil {
		return m.updateDeliveryStatusFn(id, isDelivered)
	}
	return nil
}

func (m handlerTestStore) CancelOrder(id primitive.ObjectID) error {
	if m.cancelOrderFn != nil {
		return m.cancelOrderFn(id)
	}
	return nil
}

func newHandlerWithStore(s handlerTestStore) *OrderHandler {
	return NewOrderHandler(services.NewOrderService(s, nil, nil, nil, nil))
}

func ctxWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), "userID", userID)
}

// GetOrderByID

func TestGetOrderByIDRejectsInvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.GetOrderByID(context.Background(), &pb.GetOrderByIDRequest{Id: "bad-id"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetOrderByIDRejectsInvalidUserID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.GetOrderByID(ctxWithUserID("bad-user"), &pb.GetOrderByIDRequest{Id: primitive.NewObjectID().Hex()})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGetOrderByIDReturnsNotFoundWhenStoreMisses(t *testing.T) {
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return nil, mongo.ErrNoDocuments
		},
	})

	_, err := h.GetOrderByID(ctxWithUserID(uuid.NewString()), &pb.GetOrderByIDRequest{Id: primitive.NewObjectID().Hex()})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetOrderByIDReturnsInternalForUnexpectedError(t *testing.T) {
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return nil, errors.New("boom")
		},
	})

	_, err := h.GetOrderByID(ctxWithUserID(uuid.NewString()), &pb.GetOrderByIDRequest{Id: primitive.NewObjectID().Hex()})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetOrderByIDReturnsResponseOnSuccess(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(id primitive.ObjectID) (*models.Order, error) {
			assert.Equal(t, orderID, id)
			return &models.Order{
				ID:            orderID,
				CustomerID:    customerID,
				Address:       "Address",
				Pincode:       "12345",
				Currency:      "INR",
				CheckoutTotal: 499,
				IsPaid:        true,
				IsDelivered:   false,
			}, nil
		},
	})

	res, err := h.GetOrderByID(ctxWithUserID(customerID.String()), &pb.GetOrderByIDRequest{Id: orderID.Hex()})
	require.NoError(t, err)
	assert.Equal(t, orderID.Hex(), res.Id)
	assert.Equal(t, customerID.String(), res.CustomerId)
	assert.Equal(t, "Address", res.Address)
}

// GetOrdersByCustomerID (unary)

func TestGetOrdersByCustomerIDRejectsInvalidUserID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.GetOrdersByCustomerID(ctxWithUserID("bad-user"), &emptypb.Empty{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGetOrdersByCustomerIDReturnsOrders(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	h := newHandlerWithStore(handlerTestStore{
		getOrdersByCustomerIDFn: func(id uuid.UUID) ([]models.Order, error) {
			assert.Equal(t, customerID, id)
			return []models.Order{{
				ID:            orderID,
				CustomerID:    customerID,
				Address:       "Address",
				Pincode:       "560001",
				Currency:      "INR",
				CheckoutTotal: 250,
			}}, nil
		},
	})

	res, err := h.GetOrdersByCustomerID(ctxWithUserID(customerID.String()), &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, res.Orders, 1)
	assert.Equal(t, orderID.Hex(), res.Orders[0].Id)
	assert.Equal(t, customerID.String(), res.Orders[0].CustomerId)
}

func TestGetOrdersByCustomerIDPropagatesStoreError(t *testing.T) {
	h := newHandlerWithStore(handlerTestStore{
		getOrdersByCustomerIDFn: func(uuid.UUID) ([]models.Order, error) {
			return nil, errors.New("db error")
		},
	})

	_, err := h.GetOrdersByCustomerID(ctxWithUserID(uuid.NewString()), &emptypb.Empty{})
	require.Error(t, err)
}

// CreateOrder

func TestCreateOrderRejectsInvalidUserID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.CreateOrder(ctxWithUserID("bad-user"), &pb.CreateOrderRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// GetKey

func TestGetKeyReturnsConfiguredRazorpayKey(t *testing.T) {
	t.Setenv("BROKERS", "")
	t.Setenv("RAZORPAY_KEY_ID", "rzp_test_key")
	config.Init()

	h := NewOrderHandler(nil)
	res, err := h.GetKey(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, "rzp_test_key", res.Key)
}

// Unimplemented stubs

func TestPaymentSuccessReturnsUnimplemented(t *testing.T) {
	h := NewOrderHandler(nil)
	_, err := h.PaymentSuccess(context.Background(), &pb.PaymentSuccessRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}

func TestPaymentFailureReturnsUnimplemented(t *testing.T) {
	h := NewOrderHandler(nil)
	_, err := h.PaymentFailure(context.Background(), &pb.PaymentFailureRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}

func TestPaymentCallbackReturnsUnimplemented(t *testing.T) {
	h := NewOrderHandler(nil)
	_, err := h.PaymentCallback(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}

// UpdateDeliveryAddress

func TestUpdateDeliveryAddressRejectsInvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.UpdateDeliveryAddress(context.Background(), &pb.UpdateDeliveryAddressRequest{Id: "bad-id"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestUpdateDeliveryAddressRejectsInvalidUserID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.UpdateDeliveryAddress(ctxWithUserID("bad-user"), &pb.UpdateDeliveryAddressRequest{Id: primitive.NewObjectID().Hex(), Address: "new"})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUpdateDeliveryAddressReturnsNotFoundWhenStoreMisses(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return &models.Order{ID: orderID, CustomerID: customerID}, nil
		},
		updateDeliveryAddressFn: func(primitive.ObjectID, string) error {
			return mongo.ErrNoDocuments
		},
	})

	_, err := h.UpdateDeliveryAddress(ctxWithUserID(customerID.String()), &pb.UpdateDeliveryAddressRequest{Id: orderID.Hex(), Address: "new"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateDeliveryAddressReturnsResponseOnSuccess(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(id primitive.ObjectID) (*models.Order, error) {
			return &models.Order{ID: id, CustomerID: customerID}, nil
		},
		updateDeliveryAddressFn: func(id primitive.ObjectID, address string) error {
			assert.Equal(t, orderID, id)
			assert.Equal(t, "new address", address)
			return nil
		},
	})

	res, err := h.UpdateDeliveryAddress(ctxWithUserID(customerID.String()), &pb.UpdateDeliveryAddressRequest{Id: orderID.Hex(), Address: "new address"})
	require.NoError(t, err)
	require.NotNil(t, res)
}

// CancelOrder

func TestCancelOrderRejectsInvalidOrderID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.CancelOrder(context.Background(), &pb.CancelOrderRequest{Id: "bad-id"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCancelOrderRejectsInvalidUserID(t *testing.T) {
	h := NewOrderHandler(nil)

	_, err := h.CancelOrder(ctxWithUserID("bad-user"), &pb.CancelOrderRequest{Id: primitive.NewObjectID().Hex()})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestCancelOrderReturnsNotFoundWhenOrderMissing(t *testing.T) {
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return nil, mongo.ErrNoDocuments
		},
	})

	_, err := h.CancelOrder(ctxWithUserID(uuid.NewString()), &pb.CancelOrderRequest{Id: primitive.NewObjectID().Hex()})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestCancelOrderReturnsInternalForUnexpectedError(t *testing.T) {
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return nil, errors.New("db failure")
		},
	})

	_, err := h.CancelOrder(ctxWithUserID(uuid.NewString()), &pb.CancelOrderRequest{Id: primitive.NewObjectID().Hex()})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCancelOrderReturnsCancelledResponse(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	h := newHandlerWithStore(handlerTestStore{
		getOrderByIDFn: func(id primitive.ObjectID) (*models.Order, error) {
			return &models.Order{ID: id, CustomerID: customerID}, nil
		},
		cancelOrderFn: func(id primitive.ObjectID) error {
			assert.Equal(t, orderID, id)
			return nil
		},
	})

	res, err := h.CancelOrder(ctxWithUserID(customerID.String()), &pb.CancelOrderRequest{Id: orderID.Hex()})
	require.NoError(t, err)
	assert.True(t, res.IsCancelled)
}
