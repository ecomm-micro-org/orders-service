package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ Storer = (*MongoStore)(nil)

func TestNewMongoStoreSetsFields(t *testing.T) {
	client := new(mongo.Client)
	store := NewMongoStore(client, "ordersDB", "orders")

	assert.Same(t, client, store.client)
	assert.Equal(t, "ordersDB", store.database)
	assert.Equal(t, "orders", store.collection)
}
