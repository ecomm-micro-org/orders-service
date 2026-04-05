package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/risbern21/ecom/orders/controllers"
)

func OrderRoutes(r fiber.Router, c *controllers.Controller) {
	r.Get("/get_my_orders", c.GetOrdersByCustomerID)
	r.Get("/order/:id", c.GetOrderByID)

	r.Get("/get-key", c.GetKey)

	r.Get("/success.html", c.PaymentSuccess)
	r.Get("/failure.html", c.PaymentFailure)

	r.Post("/payment-callback", c.PaymentCallback)

	r.Post("/create", c.CreateOrder)

	r.Put("/order/:id", c.UpdateDeliveryAddress)

	r.Delete("/order/:id", c.CancelOrder)
}
