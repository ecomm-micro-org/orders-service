package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/risbern21/runaway/orders-service/gen/pb"
	"github.com/risbern21/runaway/orders-service/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/metadata"
)

type serviceTestStore struct {
	createOrderFn           func(*models.Order) error
	getOrderByIDFn          func(primitive.ObjectID) (*models.Order, error)
	getOrdersByCustomerIDFn func(uuid.UUID) ([]models.Order, error)
	updateDeliveryAddressFn func(primitive.ObjectID, string) error
	updatePaymentStatusFn   func(primitive.ObjectID, bool) error
	updateDeliveryStatusFn  func(primitive.ObjectID, bool) error
	cancelOrderFn           func(primitive.ObjectID) error
}

func (m serviceTestStore) CreateOrder(o *models.Order) error {
	if m.createOrderFn != nil {
		return m.createOrderFn(o)
	}
	return nil
}

func (m serviceTestStore) GetOrderByID(id primitive.ObjectID) (*models.Order, error) {
	if m.getOrderByIDFn != nil {
		return m.getOrderByIDFn(id)
	}
	return nil, nil
}

func (m serviceTestStore) GetOrdersByCustomerID(customerID uuid.UUID) ([]models.Order, error) {
	if m.getOrdersByCustomerIDFn != nil {
		return m.getOrdersByCustomerIDFn(customerID)
	}
	return nil, nil
}

func (m serviceTestStore) UpdateDeliveryAddress(id primitive.ObjectID, address string) error {
	if m.updateDeliveryAddressFn != nil {
		return m.updateDeliveryAddressFn(id, address)
	}
	return nil
}

func (m serviceTestStore) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	if m.updatePaymentStatusFn != nil {
		return m.updatePaymentStatusFn(id, isPaid)
	}
	return nil
}

func (m serviceTestStore) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	if m.updateDeliveryStatusFn != nil {
		return m.updateDeliveryStatusFn(id, isDelivered)
	}
	return nil
}

func (m serviceTestStore) CancelOrder(id primitive.ObjectID) error {
	if m.cancelOrderFn != nil {
		return m.cancelOrderFn(id)
	}
	return nil
}

type serviceOrdersStream struct {
	ctx     context.Context
	sent    []*pb.GetOrdersByCustomerIDResponse
	sendErr error
}

func (s *serviceOrdersStream) Send(res *pb.GetOrdersByCustomerIDResponse) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.sent = append(s.sent, res)
	return nil
}

func (s *serviceOrdersStream) SetHeader(metadata.MD) error  { return nil }
func (s *serviceOrdersStream) SendHeader(metadata.MD) error { return nil }
func (s *serviceOrdersStream) SetTrailer(metadata.MD)       {}
func (s *serviceOrdersStream) Context() context.Context     { return s.ctx }
func (s *serviceOrdersStream) SendMsg(any) error            { return nil }
func (s *serviceOrdersStream) RecvMsg(any) error            { return nil }

func TestGetOrderByIDReturnsMappedResponse(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(id primitive.ObjectID) (*models.Order, error) {
			assert.Equal(t, orderID, id)
			return &models.Order{
				ID:            orderID,
				CustomerID:    customerID,
				Address:       "Address",
				Pincode:       "123456",
				Currency:      "INR",
				CheckoutTotal: 999,
				IsPaid:        true,
				IsDelivered:   true,
			}, nil
		},
	}}

	res, err := svc.GetOrderByID(orderID, customerID)
	require.NoError(t, err)
	assert.Equal(t, orderID.Hex(), res.Id)
	assert.Equal(t, customerID.String(), res.CustomerId)
	assert.Equal(t, "Address", res.Address)
	assert.Equal(t, "INR", res.Currency)
	assert.True(t, res.IsPaid)
	assert.True(t, res.IsDelivered)
}

func TestGetOrderByIDPropagatesStoreError(t *testing.T) {
	expectedErr := errors.New("store failed")
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return nil, expectedErr
		},
	}}

	_, err := svc.GetOrderByID(primitive.NewObjectID(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestGetOrderByIDRejectsUnauthorizedUser(t *testing.T) {
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return &models.Order{CustomerID: uuid.New()}, nil
		},
	}}

	_, err := svc.GetOrderByID(primitive.NewObjectID(), uuid.New())
	require.Error(t, err)
	assert.EqualError(t, err, "you do not have authorization to access this resource")
}

func TestGetOrdersByCustomerIDPropagatesStoreError(t *testing.T) {
	expectedErr := errors.New("store failed")
	svc := &OrderService{store: serviceTestStore{
		getOrdersByCustomerIDFn: func(uuid.UUID) ([]models.Order, error) {
			return nil, expectedErr
		},
	}}

	err := svc.GetOrdersByCustomerID(uuid.New(), &serviceOrdersStream{ctx: context.Background()})
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestGetOrdersByCustomerIDStreamsEachOrder(t *testing.T) {
	customerID := uuid.New()
	firstID := primitive.NewObjectID()
	secondID := primitive.NewObjectID()
	svc := &OrderService{store: serviceTestStore{
		getOrdersByCustomerIDFn: func(id uuid.UUID) ([]models.Order, error) {
			assert.Equal(t, customerID, id)
			return []models.Order{
				{ID: firstID, CustomerID: customerID, Address: "A", Currency: "INR"},
				{ID: secondID, CustomerID: customerID, Address: "B", Currency: "USD"},
			}, nil
		},
	}}
	stream := &serviceOrdersStream{ctx: context.Background()}

	err := svc.GetOrdersByCustomerID(customerID, stream)
	require.NoError(t, err)
	require.Len(t, stream.sent, 2)
	assert.Equal(t, firstID.Hex(), stream.sent[0].Id)
	assert.Equal(t, secondID.Hex(), stream.sent[1].Id)
	assert.Equal(t, "USD", stream.sent[1].Currency)
}

func TestGetOrdersByCustomerIDReturnsSendError(t *testing.T) {
	expectedErr := errors.New("send failed")
	svc := &OrderService{store: serviceTestStore{
		getOrdersByCustomerIDFn: func(uuid.UUID) ([]models.Order, error) {
			return []models.Order{{ID: primitive.NewObjectID(), CustomerID: uuid.New()}}, nil
		},
	}}

	err := svc.GetOrdersByCustomerID(uuid.New(), &serviceOrdersStream{ctx: context.Background(), sendErr: expectedErr})
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestUpdateDeliveryAddressPropagatesGetOrderError(t *testing.T) {
	expectedErr := errors.New("lookup failed")
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return nil, expectedErr
		},
	}}

	err := svc.UpdateDeliveryAddress(primitive.NewObjectID(), uuid.New(), "new address")
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestUpdateDeliveryAddressRejectsUnauthorizedUser(t *testing.T) {
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return &models.Order{CustomerID: uuid.New()}, nil
		},
	}}

	err := svc.UpdateDeliveryAddress(primitive.NewObjectID(), uuid.New(), "new address")
	require.Error(t, err)
	assert.EqualError(t, err, "you do not have enough permissions to access this resource")
}

func TestUpdateDeliveryAddressDelegatesToStore(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	called := false
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(id primitive.ObjectID) (*models.Order, error) {
			return &models.Order{ID: id, CustomerID: customerID}, nil
		},
		updateDeliveryAddressFn: func(id primitive.ObjectID, address string) error {
			called = true
			assert.Equal(t, orderID, id)
			assert.Equal(t, "updated", address)
			return nil
		},
	}}

	err := svc.UpdateDeliveryAddress(orderID, customerID, "updated")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestUpdatePaymentStatusDelegatesToStore(t *testing.T) {
	orderID := primitive.NewObjectID()
	called := false
	svc := &OrderService{store: serviceTestStore{
		updatePaymentStatusFn: func(id primitive.ObjectID, isPaid bool) error {
			called = true
			assert.Equal(t, orderID, id)
			assert.True(t, isPaid)
			return nil
		},
	}}

	err := svc.UpdatePaymentStatus(orderID, true)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestUpdateDeliveryStatusDelegatesToStore(t *testing.T) {
	orderID := primitive.NewObjectID()
	called := false
	svc := &OrderService{store: serviceTestStore{
		updateDeliveryStatusFn: func(id primitive.ObjectID, isDelivered bool) error {
			called = true
			assert.Equal(t, orderID, id)
			assert.True(t, isDelivered)
			return nil
		},
	}}

	err := svc.UpdateDeliveryStatus(orderID, true)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestCancelOrderPropagatesLookupError(t *testing.T) {
	expectedErr := errors.New("lookup failed")
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return nil, expectedErr
		},
	}}

	err := svc.CancelOrder(primitive.NewObjectID(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestCancelOrderRejectsUnauthorizedUser(t *testing.T) {
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(primitive.ObjectID) (*models.Order, error) {
			return &models.Order{CustomerID: uuid.New()}, nil
		},
	}}

	err := svc.CancelOrder(primitive.NewObjectID(), uuid.New())
	require.Error(t, err)
	assert.EqualError(t, err, "you do not have access to this order")
}

func TestCancelOrderDelegatesToStore(t *testing.T) {
	customerID := uuid.New()
	orderID := primitive.NewObjectID()
	called := false
	svc := &OrderService{store: serviceTestStore{
		getOrderByIDFn: func(id primitive.ObjectID) (*models.Order, error) {
			return &models.Order{ID: id, CustomerID: customerID}, nil
		},
		cancelOrderFn: func(id primitive.ObjectID) error {
			called = true
			assert.Equal(t, orderID, id)
			return nil
		},
	}}

	err := svc.CancelOrder(orderID, customerID)
	require.NoError(t, err)
	assert.True(t, called)
}
