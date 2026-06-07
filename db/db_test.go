package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestClientReturnsConfiguredMongoClient(t *testing.T) {
	old := client
	expected := new(mongo.Client)
	client = expected
	t.Cleanup(func() {
		client = old
	})

	assert.Same(t, expected, Client())
}

func TestDisconnectWithNilClientReturnsNil(t *testing.T) {
	old := client
	client = nil
	t.Cleanup(func() {
		client = old
	})

	assert.NoError(t, Disconnect())
}
