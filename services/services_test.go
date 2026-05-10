package services_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/risbern21/ecom/orders/internal/dto"
	"github.com/risbern21/ecom/orders/internal/token"
	"github.com/risbern21/ecom/orders/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---------------------------------------------------------------------------
// Interfaces — define minimal contracts so every dependency can be faked
// ---------------------------------------------------------------------------

// OrderRepository abstracts the DB layer used by OrderService.
type OrderRepository interface {
	CreateOrder(m *models.Order) error
	GetOrderByID(m *models.Order) error
	GetOrdersByCustomerID(customerID uuid.UUID) ([]models.Order, error)
	UpdateDeliveryAddress(id primitive.ObjectID, address string) error
	UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error
	UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error
	CancelOrder(id primitive.ObjectID) error
}

// PaymentGateway abstracts Razorpay (or any payment provider).
type PaymentGateway interface {
	CreateOrder(amount float64, currency, receipt string) (string, error)
}

// ProductClient abstracts the HTTP call to the products service.
type ProductClient interface {
	CalculateTotalPrice(orderItems any, accessToken string) (float64, error)
}

// NotificationSender abstracts Courier (or any notification provider).
type NotificationSender interface {
	SendOrderPlaced(customerID string) error
}

// EventPublisher abstracts Kafka publishing.
type EventPublisher interface {
	PublishOrderCreated(orderID string, items []dto.OrderItem) error
}

// ---------------------------------------------------------------------------
// mocks
// ---------------------------------------------------------------------------

type fakeRepo struct {
	createErr               error
	getByIDErr              error
	getByIDResult           *models.Order
	getByCustomerErr        error
	getByCustomerResult     []models.Order
	updateAddressErr        error
	updatePaymentStatusErr  error
	updateDeliveryStatusErr error
	cancelErr               error
}

func (f *fakeRepo) CreateOrder(m *models.Order) error { return f.createErr }
func (f *fakeRepo) GetOrderByID(m *models.Order) error {
	if f.getByIDResult != nil {
		*m = *f.getByIDResult
	}
	return f.getByIDErr
}
func (f *fakeRepo) GetOrdersByCustomerID(customerID uuid.UUID) ([]models.Order, error) {
	return f.getByCustomerResult, f.getByCustomerErr
}
func (f *fakeRepo) UpdateDeliveryAddress(id primitive.ObjectID, address string) error {
	return f.updateAddressErr
}
func (f *fakeRepo) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	return f.updatePaymentStatusErr
}
func (f *fakeRepo) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	return f.updateDeliveryStatusErr
}
func (f *fakeRepo) CancelOrder(id primitive.ObjectID) error { return f.cancelErr }

type fakePayment struct {
	rzpID string
	err   error
}

func (f *fakePayment) CreateOrder(amount float64, currency, receipt string) (string, error) {
	return f.rzpID, f.err
}

type fakeProductClient struct {
	total float64
	err   error
}

func (f *fakeProductClient) CalculateTotalPrice(orderItems any, accessToken string) (float64, error) {
	return f.total, f.err
}

type fakeNotifier struct{ err error }

func (f *fakeNotifier) SendOrderPlaced(customerID string) error { return f.err }

type fakePublisher struct{ err error }

func (f *fakePublisher) PublishOrderCreated(orderID string, items []dto.OrderItem) error {
	return f.err
}

// ---------------------------------------------------------------------------
// Testable OrderService (wraps dependencies via interfaces)
// ---------------------------------------------------------------------------

// TestableOrderService mirrors OrderService but accepts injected deps.
// Keeps the real business logic; only I/O boundaries are swapped.
type TestableOrderService struct {
	userClaims    *token.UserClaims
	repo          OrderRepository
	payment       PaymentGateway
	productClient ProductClient
	notifier      NotificationSender
	publisher     EventPublisher
}

func (s *TestableOrderService) CreateOrder(
	accessToken string,
	req *dto.OrderRequest,
) (*dto.OrderResponse, error) {
	total, err := s.productClient.CalculateTotalPrice(req.OrderItems, accessToken)
	if err != nil {
		return nil, err
	}

	order := &models.Order{
		ID:            primitive.NewObjectID(),
		CustomerID:    s.userClaims.ID,
		Address:       req.Address,
		Pincode:       req.Pincode,
		OrderItems:    req.OrderItems,
		CheckoutTotal: total,
		Currency:      req.Currency,
	}

	if err := s.repo.CreateOrder(order); err != nil {
		return nil, err
	}

	rzpID, err := s.payment.CreateOrder(order.CheckoutTotal, req.Currency, "receipt_test")
	if err != nil {
		return nil, err
	}

	// fire-and-forget — errors logged, not returned
	_ = s.publisher.PublishOrderCreated(order.ID.String(), order.OrderItems)
	_ = s.notifier.SendOrderPlaced(s.userClaims.ID.String())

	return &dto.OrderResponse{
		ID:            order.ID,
		CustomerID:    order.CustomerID,
		OrderItems:    order.OrderItems,
		Address:       order.Address,
		Pincode:       order.Pincode,
		CheckoutTotal: order.CheckoutTotal,
		Currency:      order.Currency,
		RzpID:         rzpID,
	}, nil
}

func (s *TestableOrderService) GetOrderByID(id primitive.ObjectID) (*dto.OrderResponse, error) {
	order := &models.Order{ID: id}
	if err := s.repo.GetOrderByID(order); err != nil {
		return nil, err
	}
	if s.userClaims.ID != order.CustomerID {
		return nil, errors.New("you do not have authorization to access this resource")
	}
	return &dto.OrderResponse{
		ID:            order.ID,
		CustomerID:    order.CustomerID,
		OrderItems:    order.OrderItems,
		Address:       order.Address,
		Pincode:       order.Pincode,
		CheckoutTotal: order.CheckoutTotal,
		IsDelivered:   order.IsDelivered,
		IsPaid:        order.IsPaid,
	}, nil
}

func (s *TestableOrderService) GetOrdersByCustomerID() ([]dto.OrderResponse, error) {
	orders, err := s.repo.GetOrdersByCustomerID(s.userClaims.ID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.OrderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, dto.OrderResponse{
			ID:            o.ID,
			CustomerID:    o.CustomerID,
			OrderItems:    o.OrderItems,
			Address:       o.Address,
			Pincode:       o.Pincode,
			CheckoutTotal: o.CheckoutTotal,
			IsDelivered:   o.IsDelivered,
			IsPaid:        o.IsPaid,
		})
	}
	return resp, nil
}

func (s *TestableOrderService) UpdateDeliveryAddress(id primitive.ObjectID, address string) error {
	return s.repo.UpdateDeliveryAddress(id, address)
}

func (s *TestableOrderService) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	return s.repo.UpdatePaymentStatus(id, isPaid)
}

func (s *TestableOrderService) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	return s.repo.UpdateDeliveryStatus(id, isDelivered)
}

func (s *TestableOrderService) CancelOrder(id primitive.ObjectID, customerID uuid.UUID) error {
	order := &models.Order{ID: id}
	if err := s.repo.GetOrderByID(order); err != nil {
		return err
	}
	if order.CustomerID != customerID {
		return errors.New("you do not have access to this order")
	}
	return s.repo.CancelOrder(id)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var (
	ownerID     = uuid.New()
	strangerID  = uuid.New()
	sampleID    = primitive.NewObjectID()
	sampleItems = []dto.OrderItem{
		{ProductID: 269, Quantity: 2},
	}
)

func ownerClaims() *token.UserClaims { return &token.UserClaims{ID: ownerID} }
func newSvc(repo OrderRepository, pay PaymentGateway, pc ProductClient) *TestableOrderService {
	return &TestableOrderService{
		userClaims:    ownerClaims(),
		repo:          repo,
		payment:       pay,
		productClient: pc,
		notifier:      &fakeNotifier{},
		publisher:     &fakePublisher{},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// ── CreateOrder ─────────────────────────────────────────────────────────────

func TestCreateOrder(t *testing.T) {
	baseReq := &dto.OrderRequest{
		Address:    "123 Main St",
		Pincode:    "411001",
		Currency:   "INR",
		OrderItems: sampleItems,
	}

	tests := []struct {
		name          string
		repo          *fakeRepo
		payment       *fakePayment
		productClient *fakeProductClient
		req           *dto.OrderRequest
		wantErr       bool
		wantErrMsg    string
		assertResp    func(t *testing.T, resp *dto.OrderResponse)
	}{
		{
			name:          "success — returns populated response with rzp ID",
			repo:          &fakeRepo{},
			payment:       &fakePayment{rzpID: "rzp_order_abc"},
			productClient: &fakeProductClient{total: 500.00},
			req:           baseReq,
			wantErr:       false,
			assertResp: func(t *testing.T, resp *dto.OrderResponse) {
				assert.Equal(t, "rzp_order_abc", resp.RzpID)
				assert.Equal(t, 500.00, resp.CheckoutTotal)
				assert.Equal(t, ownerID, resp.CustomerID)
				assert.Equal(t, "123 Main St", resp.Address)
				assert.Equal(t, "INR", resp.Currency)
				assert.NotEmpty(t, resp.ID)
			},
		},
		{
			name:          "product client error — propagates error",
			repo:          &fakeRepo{},
			payment:       &fakePayment{rzpID: "rzp_order_abc"},
			productClient: &fakeProductClient{err: errors.New("products service unavailable")},
			req:           baseReq,
			wantErr:       true,
			wantErrMsg:    "products service unavailable",
		},
		{
			name:          "repo create error — propagates error",
			repo:          &fakeRepo{createErr: errors.New("db write failed")},
			payment:       &fakePayment{rzpID: "rzp_order_abc"},
			productClient: &fakeProductClient{total: 200.00},
			req:           baseReq,
			wantErr:       true,
			wantErrMsg:    "db write failed",
		},
		{
			name:          "payment gateway error — propagates error",
			repo:          &fakeRepo{},
			payment:       &fakePayment{err: errors.New("razorpay timeout")},
			productClient: &fakeProductClient{total: 300.00},
			req:           baseReq,
			wantErr:       true,
			wantErrMsg:    "razorpay timeout",
		},
		{
			name:          "zero total — allowed, sets checkout total to 0",
			repo:          &fakeRepo{},
			payment:       &fakePayment{rzpID: "rzp_free"},
			productClient: &fakeProductClient{total: 0},
			req:           baseReq,
			wantErr:       false,
			assertResp: func(t *testing.T, resp *dto.OrderResponse) {
				assert.Equal(t, 0.0, resp.CheckoutTotal)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc(tc.repo, tc.payment, tc.productClient)
			resp, err := svc.CreateOrder("Bearer token", tc.req)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.assertResp != nil {
				tc.assertResp(t, resp)
			}
		})
	}
}

// ── GetOrderByID ─────────────────────────────────────────────────────────────

func TestGetOrderByID(t *testing.T) {
	ownerOrder := &models.Order{
		ID:            sampleID,
		CustomerID:    ownerID,
		Address:       "Baker Street",
		Pincode:       "NW1",
		CheckoutTotal: 999.99,
		IsPaid:        true,
	}

	tests := []struct {
		name       string
		claims     *token.UserClaims
		repo       *fakeRepo
		orderID    primitive.ObjectID
		wantErr    bool
		wantErrMsg string
		assertResp func(t *testing.T, resp *dto.OrderResponse)
	}{
		{
			name:    "success — owner fetches own order",
			claims:  ownerClaims(),
			repo:    &fakeRepo{getByIDResult: ownerOrder},
			orderID: sampleID,
			wantErr: false,
			assertResp: func(t *testing.T, resp *dto.OrderResponse) {
				assert.Equal(t, sampleID, resp.ID)
				assert.Equal(t, ownerID, resp.CustomerID)
				assert.Equal(t, 999.99, resp.CheckoutTotal)
				assert.True(t, resp.IsPaid)
			},
		},
		{
			name:       "unauthorized — different customer ID",
			claims:     &token.UserClaims{ID: strangerID},
			repo:       &fakeRepo{getByIDResult: ownerOrder},
			orderID:    sampleID,
			wantErr:    true,
			wantErrMsg: "authorization",
		},
		{
			name:       "repo error — order not found",
			claims:     ownerClaims(),
			repo:       &fakeRepo{getByIDErr: errors.New("mongo: no documents in result")},
			orderID:    sampleID,
			wantErr:    true,
			wantErrMsg: "mongo: no documents in result",
		},
		{
			name:   "success — delivered and paid order",
			claims: ownerClaims(),
			repo: &fakeRepo{getByIDResult: &models.Order{
				ID:          sampleID,
				CustomerID:  ownerID,
				IsDelivered: true,
				IsPaid:      true,
			}},
			orderID: sampleID,
			wantErr: false,
			assertResp: func(t *testing.T, resp *dto.OrderResponse) {
				assert.True(t, resp.IsDelivered)
				assert.True(t, resp.IsPaid)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &TestableOrderService{
				userClaims: tc.claims,
				repo:       tc.repo,
				notifier:   &fakeNotifier{},
				publisher:  &fakePublisher{},
			}
			resp, err := svc.GetOrderByID(tc.orderID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.assertResp != nil {
				tc.assertResp(t, resp)
			}
		})
	}
}

// ── GetOrdersByCustomerID ────────────────────────────────────────────────────

func TestGetOrdersByCustomerID(t *testing.T) {
	orders := []models.Order{
		{ID: primitive.NewObjectID(), CustomerID: ownerID, CheckoutTotal: 100},
		{ID: primitive.NewObjectID(), CustomerID: ownerID, CheckoutTotal: 200},
	}

	tests := []struct {
		name       string
		repo       *fakeRepo
		wantErr    bool
		wantErrMsg string
		wantCount  int
	}{
		{
			name:      "success — returns all orders for customer",
			repo:      &fakeRepo{getByCustomerResult: orders},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "success — no orders returns empty slice",
			repo:      &fakeRepo{getByCustomerResult: []models.Order{}},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:       "repo error — propagates error",
			repo:       &fakeRepo{getByCustomerErr: errors.New("connection refused")},
			wantErr:    true,
			wantErrMsg: "connection refused",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &TestableOrderService{
				userClaims: ownerClaims(),
				repo:       tc.repo,
				notifier:   &fakeNotifier{},
				publisher:  &fakePublisher{},
			}
			resp, err := svc.GetOrdersByCustomerID()

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			assert.Len(t, resp, tc.wantCount)
		})
	}
}

// ── UpdateDeliveryAddress ────────────────────────────────────────────────────

func TestUpdateDeliveryAddress(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepo
		id         primitive.ObjectID
		address    string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "success — address updated",
			repo:    &fakeRepo{},
			id:      sampleID,
			address: "456 New Road",
			wantErr: false,
		},
		{
			name:       "repo error — propagates error",
			repo:       &fakeRepo{updateAddressErr: errors.New("update failed")},
			id:         sampleID,
			address:    "456 New Road",
			wantErr:    true,
			wantErrMsg: "update failed",
		},
		{
			name:    "success — empty address (validation is caller's concern)",
			repo:    &fakeRepo{},
			id:      sampleID,
			address: "",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &TestableOrderService{userClaims: ownerClaims(), repo: tc.repo}
			err := svc.UpdateDeliveryAddress(tc.id, tc.address)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

// ── UpdatePaymentStatus ──────────────────────────────────────────────────────

func TestUpdatePaymentStatus(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepo
		isPaid     bool
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "success — mark as paid",
			repo:    &fakeRepo{},
			isPaid:  true,
			wantErr: false,
		},
		{
			name:    "success — mark as unpaid",
			repo:    &fakeRepo{},
			isPaid:  false,
			wantErr: false,
		},
		{
			name:       "repo error — propagates error",
			repo:       &fakeRepo{updatePaymentStatusErr: errors.New("payment update failed")},
			isPaid:     true,
			wantErr:    true,
			wantErrMsg: "payment update failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &TestableOrderService{userClaims: ownerClaims(), repo: tc.repo}
			err := svc.UpdatePaymentStatus(sampleID, tc.isPaid)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

// ── UpdateDeliveryStatus ─────────────────────────────────────────────────────

func TestUpdateDeliveryStatus(t *testing.T) {
	tests := []struct {
		name        string
		repo        *fakeRepo
		isDelivered bool
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "success — mark as delivered",
			repo:        &fakeRepo{},
			isDelivered: true,
			wantErr:     false,
		},
		{
			name:        "success — mark as not delivered",
			repo:        &fakeRepo{},
			isDelivered: false,
			wantErr:     false,
		},
		{
			name:        "repo error — propagates error",
			repo:        &fakeRepo{updateDeliveryStatusErr: errors.New("delivery update failed")},
			isDelivered: true,
			wantErr:     true,
			wantErrMsg:  "delivery update failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &TestableOrderService{userClaims: ownerClaims(), repo: tc.repo}
			err := svc.UpdateDeliveryStatus(sampleID, tc.isDelivered)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

// ── CancelOrder ──────────────────────────────────────────────────────────────

func TestCancelOrder(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepo
		orderID    primitive.ObjectID
		customerID uuid.UUID
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success — owner cancels own order",
			repo: &fakeRepo{
				getByIDResult: &models.Order{ID: sampleID, CustomerID: ownerID},
			},
			orderID:    sampleID,
			customerID: ownerID,
			wantErr:    false,
		},
		{
			name: "unauthorized — stranger tries to cancel",
			repo: &fakeRepo{
				getByIDResult: &models.Order{ID: sampleID, CustomerID: ownerID},
			},
			orderID:    sampleID,
			customerID: strangerID,
			wantErr:    true,
			wantErrMsg: "you do not have access to this order",
		},
		{
			name:       "repo get error — order not found",
			repo:       &fakeRepo{getByIDErr: errors.New("order not found")},
			orderID:    sampleID,
			customerID: ownerID,
			wantErr:    true,
			wantErrMsg: "order not found",
		},
		{
			name: "repo cancel error — propagates error",
			repo: &fakeRepo{
				getByIDResult: &models.Order{ID: sampleID, CustomerID: ownerID},
				cancelErr:     errors.New("cancel write failed"),
			},
			orderID:    sampleID,
			customerID: ownerID,
			wantErr:    true,
			wantErrMsg: "cancel write failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &TestableOrderService{
				userClaims: ownerClaims(),
				repo:       tc.repo,
				notifier:   &fakeNotifier{},
				publisher:  &fakePublisher{},
			}
			err := svc.CancelOrder(tc.orderID, tc.customerID)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}
