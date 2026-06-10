package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/ecomm-micro-org/orders-service/models"
	"github.com/ecomm-micro-org/orders-service/pb"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MemoryStore struct {
	mu     sync.RWMutex
	orders map[primitive.ObjectID]*models.Order
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		orders: make(map[primitive.ObjectID]*models.Order),
	}
}

func (s *MemoryStore) CreateOrder(o *models.Order) error {
	if o == nil {
		return fmt.Errorf("order cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cp := cloneOrder(o)
	if cp.ID.IsZero() {
		cp.ID = primitive.NewObjectID()
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	cp.UpdatedAt = cp.CreatedAt

	s.orders[cp.ID] = cp
	return nil
}

func (s *MemoryStore) GetOrderByID(id primitive.ObjectID) (*models.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	o, ok := s.orders[id]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}

	return cloneOrder(o), nil
}

func (s *MemoryStore) GetOrdersByCustomerID(customerID uuid.UUID) ([]models.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orders := make([]models.Order, 0)
	for _, o := range s.orders {
		if o.CustomerID == customerID {
			orders = append(orders, *cloneOrder(o))
		}
	}

	return orders, nil
}

func (s *MemoryStore) UpdateDeliveryAddress(id primitive.ObjectID, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orders[id]
	if !ok {
		return nil
	}

	o.Address = address
	o.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orders[id]
	if !ok {
		return fmt.Errorf("no order found")
	}

	o.IsPaid = isPaid
	o.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	o, ok := s.orders[id]
	if !ok {
		return fmt.Errorf("no order found")
	}

	o.IsDelivered = isDelivered
	o.UpdatedAt = time.Now()
	return nil
}

func (s *MemoryStore) CancelOrder(id primitive.ObjectID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.orders[id]; !ok {
		return fmt.Errorf("no order found")
	}

	delete(s.orders, id)
	return nil
}

func cloneOrder(o *models.Order) *models.Order {
	if o == nil {
		return nil
	}

	cp := *o
	if o.OrderItems != nil {
		cp.OrderItems = make([]*pb.OrderItem, len(o.OrderItems))
		for i, item := range o.OrderItems {
			if item == nil {
				continue
			}
			itemCopy := *item
			cp.OrderItems[i] = &itemCopy
		}
	}

	return &cp
}
