package app

import (
	"log"

	"github.com/risbern21/ecom/orders/internal/cache"
	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/risbern21/ecom/orders/internal/database"
	"github.com/risbern21/ecom/orders/internal/kafka"
	"github.com/risbern21/ecom/orders/internal/server"
)

func SetUp() {
	database.Connect()
	cache.Connect()

	kafka.Init(config.Config().Brokers)
	defer kafka.Close()

	server.SetUp()

	app := server.New()

	if err := app.Listen(config.Config().Port); err != nil {
		log.Fatalf("err : %v", err)
	}
}
