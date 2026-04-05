package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/risbern21/ecom/orders/internal/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Order struct {
	ID            primitive.ObjectID `bson:"_id" json:"id"`
	CustomerID    uuid.UUID          `bson:"customer_id" json:"customer_id"`
	Address       string             `bson:"address" json:"address"`
	Pincode       string             `bson:"pincode" json:"pincode"`
	CheckoutTotal float64            `bson:"checkout_total" json:"checkout_total"`
	Currency      string             `bson:"currency" json:"currency"`
	IsPaid        bool               `bson:"is_paid" json:"is_paid"`
	IsDelivered   bool               `bson:"is_delivered" json:"is_delivered"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
	DeletedAt     time.Time          `bson:"deleted_at" json:"deleted_at"`

	OrderItems []OrderItem
}

type OrderItem struct {
	ProductID uint `bson:"product_id"    json:"product_id"`
	Quantity  uint `bson:"quantity"      json:"quantity"`
}

func New() *Order {
	return &Order{}
}

func (o *Order) CreateOrder() error {
	_, err := database.Coll().InsertOne(context.TODO(), o)

	return err
}

func (o *Order) GetOrderByID() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return database.Coll().FindOne(ctx, bson.M{"_id": o.ID}).Decode(&o)
}

func (o *Order) GetOrdersByCustomerID() ([]Order, error) {
	var orders []Order
	ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel1()

	cursor, err := database.Coll().Find(ctx1, bson.M{"customer_id": o.CustomerID})
	if err != nil {
		return nil, err
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err = cursor.All(ctx2, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

func (o *Order) UpdateDeliveryAddress() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := database.Coll().UpdateOne(ctx, bson.M{"_id": o.ID}, bson.M{"$set": bson.M{"address": o.Address}})
	if err != nil {
		return err
	}

	return nil
}

func (o *Order) UpdatePaymentStatus() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := database.Coll().UpdateOne(ctx, bson.M{"_id": o.ID}, bson.M{"$set": bson.M{"is_paid": o.IsPaid}})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no order found")
	}

	return nil
}

func (o *Order) UpdateDeliveryStatus() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := database.Coll().UpdateOne(ctx, bson.M{"_id": o.ID}, bson.M{"$set": bson.M{"is_delivered": o.IsDelivered}})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no order found")
	}

	return nil
}

func (o *Order) CancelOrder() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := database.Coll().DeleteOne(ctx, bson.M{"_id": o.ID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no order found")
	}

	return nil
}
