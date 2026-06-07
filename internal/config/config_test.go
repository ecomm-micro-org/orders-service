package config

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetConfigForTest() {
	defaultConfig = nil
	once = sync.Once{}
}

func TestInitLoadsEnvironmentVariables(t *testing.T) {
	resetConfigForTest()
	t.Cleanup(resetConfigForTest)

	t.Setenv("BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("PORT", "42067")
	t.Setenv("DSN", "mongodb://localhost:27017")
	t.Setenv("CACHE_ADDR", "localhost:6379")
	t.Setenv("CACHE_PASSWD", "cache-secret")
	t.Setenv("SECRET_KEY", "secret")
	t.Setenv("RAZORPAY_KEY_ID", "key-id")
	t.Setenv("RAZORPAY_SECRET", "rzp-secret")
	t.Setenv("COURIER_KEY", "courier")
	t.Setenv("PRODUCTS_CLIENT", "localhost:50051")
	t.Setenv("SLACK_TOKEN", "slack-token")
	t.Setenv("SLACK_CHANNEL", "channel-id")

	Init()
	cfg := Config()
	require.NotNil(t, cfg)
	assert.Equal(t, []string{"broker1:9092", "broker2:9092"}, cfg.Brokers)
	assert.Equal(t, "42067", cfg.Port)
	assert.Equal(t, "mongodb://localhost:27017", cfg.DSN)
	assert.Equal(t, "localhost:6379", cfg.CacheAddr)
	assert.Equal(t, "cache-secret", cfg.CachePasswd)
	assert.Equal(t, "secret", cfg.SecretKey)
	assert.Equal(t, "key-id", cfg.RazorpayKeyID)
	assert.Equal(t, "rzp-secret", cfg.RazorpaySecret)
	assert.Equal(t, "courier", cfg.CourierKey)
	assert.Equal(t, "localhost:50051", cfg.ProductsClient)
	assert.Equal(t, "slack-token", cfg.SlackToken)
	assert.Equal(t, "channel-id", cfg.SlackChannel)
}

func TestInitRunsOnlyOnce(t *testing.T) {
	resetConfigForTest()
	t.Cleanup(resetConfigForTest)

	t.Setenv("BROKERS", "first-broker")
	t.Setenv("PORT", "1111")
	Init()

	t.Setenv("BROKERS", "second-broker")
	t.Setenv("PORT", "2222")
	Init()

	cfg := Config()
	require.NotNil(t, cfg)
	assert.Equal(t, []string{"first-broker"}, cfg.Brokers)
	assert.Equal(t, "1111", cfg.Port)
}
