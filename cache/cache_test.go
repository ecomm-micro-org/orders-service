package cache

import (
	"testing"

	"github.com/ecomm-micro-org/orders-service/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientReturnsCurrentRedisClient(t *testing.T) {
	old := rdb
	expected := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	rdb = expected
	t.Cleanup(func() {
		_ = expected.Close()
		rdb = old
	})

	assert.Same(t, expected, Client())
}

func TestConnectInitializesRedisClientAndDisconnectClosesIt(t *testing.T) {
	old := rdb
	rdb = nil
	t.Cleanup(func() {
		rdb = old
	})

	t.Setenv("BROKERS", "")
	t.Setenv("CACHE_ADDR", "localhost:6379")
	t.Setenv("CACHE_PASSWD", "secret")
	config.Init()

	Connect()
	require.NotNil(t, Client())
	assert.NoError(t, Disconnect())
}
