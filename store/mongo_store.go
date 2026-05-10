package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/risbern21/ecom/orders/internal/database"
	"github.com/risbern21/ecom/orders/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoStore struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewMongoStore() *MongoStore {
	return &MongoStore{
		client:     database.Client(),
		database:   "orderDB",
		collection: "orders",
	}
}

func (s *MongoStore) CreateOrder(o *models.Order) error {
	coll := s.client.Database(s.database).Collection(s.collection)
	_, err := coll.InsertOne(context.TODO(), o)

	return err
}

func (s *MongoStore) GetOrderByID(id primitive.ObjectID) (*models.Order, error) {
	coll := s.client.Database(s.database).Collection(s.collection)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	o := models.NewOrder()
	if err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *MongoStore) GetOrdersByCustomerID(customerID uuid.UUID) ([]models.Order, error) {
	coll := s.client.Database(s.database).Collection(s.collection)

	var orders []models.Order
	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()

	cursor, err := coll.Find(ctx1, bson.M{"customer_id": customerID})
	if err != nil {
		return nil, err
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	if err = cursor.All(ctx2, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *MongoStore) UpdateDeliveryAddress(id primitive.ObjectID, address string) error {
	coll := s.client.Database(s.database).Collection(s.collection)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"address": address}})
	if err != nil {
		return err
	}

	return nil
}

func (s *MongoStore) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	coll := s.client.Database(s.database).Collection(s.collection)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"is_paid": isPaid}})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no order found")
	}

	return nil
}

func (s *MongoStore) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	coll := s.client.Database(s.database).Collection(s.collection)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := coll.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"is_delivered": isDelivered}})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("no order found")
	}

	return nil
}

func (s *MongoStore) CancelOrder(id primitive.ObjectID) error {
	coll := s.client.Database(s.database).Collection(s.collection)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no order found")
	}

	return nil
}
