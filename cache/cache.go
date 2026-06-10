package cache

import (
	"github.com/ecomm-micro-org/orders-service/internal/config"
	"github.com/redis/go-redis/v9"
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
