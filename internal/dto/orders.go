package dto

import (
	"github.com/google/uuid"
	"github.com/risbern21/ecom/orders/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderReqeustDTO struct {
	OrderItems []models.OrderItem `json:"order_items" validate:"required"`
	Address    string             `json:"address" validate:"required"`
	Pincode    string             `json:"pincode" validate:"required"`
}

type OrderResponseDTO struct {
	ID            primitive.ObjectID `json:"id"`
	CustomerID    uuid.UUID          `json:"customer_id"`
	OrderItems    []models.OrderItem `json:"order_items"`
	Address       string             `json:"address"`
	Pincode       string             `json:"pincode"`
	CheckoutTotal float64            `json:"checkout_total"`
	IsPaid        bool               `json:"is_paid"`
	IsDelivered   bool               `json:"is_delivered"`
}
