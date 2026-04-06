package server

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/risbern21/ecom/orders/controllers"
	"github.com/risbern21/ecom/orders/internal/kafka"
	"github.com/risbern21/ecom/orders/routes"
	"github.com/risbern21/ecom/orders/services"
)

var app *fiber.App

func New() *fiber.App {
	return app
}

func consume() {
	errs := make(chan error, 10)

	pc := kafka.NewConsumer(kafka.TopicPayments)
	defer pc.Close()

	dc := kafka.NewConsumer(kafka.TopicDeliveries)
	defer dc.Close()

	s := services.New()
	go pc.ConsumeTopicPayments(s.UpdatePaymentStatus, errs)

	go dc.ConsumeTopicDeliveries(s.UpdateDeliveryStatus, errs)

	for err := range errs {
		log.Printf("error occurred :%v\n", err)
	}
}

func SetUp() {
	config := fiber.Config{
		BodyLimit:    10 * 1024 * 1024,
		ErrorHandler: errorHandler,
	}

	app = fiber.New(config)
	app.Use(logger.New())

	defer app.Use(notFoundHandler)
	defer app.Use(recover.New())

	go consume()

	app.Get("/health", func(c fiber.Ctx) {
		c.SendStatus(fiber.StatusOK)
	})
	addRoutes(app)
}

func errorHandler(c fiber.Ctx, e error) error {
	err := e.Error()
	return c.Status(fiber.StatusInternalServerError).JSON(err)
}

var notFoundHandler = func(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotFound)
}

func addRoutes(app *fiber.App) {
	baseRouter := app.Group("/orders")

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		log.Fatal("secret key not defined")
	}

	c := controllers.NewController(secretKey)

	routes.OrderRoutes(baseRouter, c)
}
