package cache

import (
	"github.com/redis/go-redis/v9"
	"github.com/risbern21/runaway/orders-service/internal/config"
)

var rdb *redis.Client

func Client() *redis.Client {
	return rdb
}

func Connect() {
	cacheAddr := config.Config().CacheAddr
	cachePasswd := config.Config().CachePasswd

	rdb = redis.NewClient(&redis.Options{
		Addr:     cacheAddr,
		Password: cachePasswd,
		DB:       0,
	})
}

func Disconnect() error {
	return rdb.Close()
}
