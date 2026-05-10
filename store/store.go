package store

import (
	"github.com/google/uuid"
	"github.com/risbern21/ecom/orders/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Storer interface {
	CreateOrder(o *models.Order) error
	GetOrderByID(id primitive.ObjectID) (*models.Order, error)
	GetOrdersByCustomerID(customerID uuid.UUID) ([]models.Order, error)
	UpdateDeliveryAddress(id primitive.ObjectID, address string) error
	UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error
	UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error
	CancelOrder(id primitive.ObjectID) error
}
