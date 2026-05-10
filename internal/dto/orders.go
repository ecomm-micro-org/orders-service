package dto

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderItem struct {
	ProductID uint `json:"product_id" validate:"required" bson:"product_id"`
	Quantity  uint `json:"quantity" validate:"required" bson:"quantity"`
}

type OrderRequest struct {
	OrderItems []OrderItem `json:"order_items" validate:"required"`
	Address    string      `json:"address" validate:"required"`
	Pincode    string      `json:"pincode" validate:"required"`
	Currency   string      `json:"currency" validate:"required"`
}

type OrderResponse struct {
	ID            primitive.ObjectID `json:"id"`
	RzpID         string             `json:"rzp_id"`
	CustomerID    uuid.UUID          `json:"customer_id"`
	OrderItems    []OrderItem        `json:"order_items"`
	Address       string             `json:"address"`
	Pincode       string             `json:"pincode"`
	CheckoutTotal float64            `json:"checkout_total"`
	Currency      string             `json:"currency"`
	IsPaid        bool               `json:"is_paid"`
	IsDelivered   bool               `json:"is_delivered"`
}
