package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/risbern21/runaway/orders-service/gen/pb"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Order struct {
	ID            primitive.ObjectID `bson:"_id" json:"id"`
	CustomerID    uuid.UUID          `bson:"customer_id" json:"customer_id"`
	RzpID         string             `bson:"rzp_id" json:"rzp_id"`
	Address       string             `bson:"address" json:"address"`
	Pincode       string             `bson:"pincode" json:"pincode"`
	CheckoutTotal float64            `bson:"checkout_total" json:"checkout_total"`
	Currency      string             `bson:"currency" json:"currency"`
	IsPaid        bool               `bson:"is_paid" json:"is_paid"`
	IsDelivered   bool               `bson:"is_delivered" json:"is_delivered"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt     time.Time          `bson:"deleted_at" json:"deleted_at"`

	OrderItems []*pb.OrderItem `bson:"order_items" json:"order_items"`
}

func NewOrder() *Order {
	return &Order{}
}
