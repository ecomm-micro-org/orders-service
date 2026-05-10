package server

import (
	"github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/risbern21/ecom/orders/controllers"
	"github.com/risbern21/ecom/orders/routes"
)

var app *fiber.App

func New() *fiber.App {
	return app
}

func SetUp(c *controllers.Controller) {
	config := fiber.Config{
		BodyLimit:    10 * 1024 * 1024,
		ErrorHandler: errorHandler,
	}

	app = fiber.New(config)
	app.Use(logger.New())

	defer app.Use(notFoundHandler)
	defer app.Use(recover.New())

	app.Get("/health", func(c fiber.Ctx) {
		c.SendStatus(fiber.StatusOK)
	})

	app.Get("/swagger/*", swaggo.HandlerDefault)

	addRoutes(app, c)
}

func errorHandler(c fiber.Ctx, e error) error {
	err := e.Error()
	return c.Status(fiber.StatusInternalServerError).JSON(err)
}

var notFoundHandler = func(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNotFound)
}

func addRoutes(app *fiber.App, c *controllers.Controller) {
	baseRouter := app.Group("/orders")

	routes.OrderRoutes(baseRouter, c)
}
