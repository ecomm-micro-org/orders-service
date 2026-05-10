package controllers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/razorpay/razorpay-go/utils"
	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/risbern21/ecom/orders/internal/dto"
	"github.com/risbern21/ecom/orders/internal/token"
	"github.com/risbern21/ecom/orders/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ErrorMsg struct {
	Message string `json:"message"`
}

type Controller struct {
	extractor *token.JWTExtractor
	s         *services.OrderService
}

func NewController(secretKey string, orderSvc *services.OrderService) *Controller {
	return &Controller{
		extractor: token.NewJWTExtractor(secretKey),
		s:         orderSvc,
	}
}

func NewControllerWithService(s *services.OrderService, secretKey string) *Controller {
	return &Controller{
		extractor: token.NewJWTExtractor(secretKey),
		s:         s,
	}
}

func extractAccessToken(c fiber.Ctx) (string, error) {
	accessToken := c.Get("Authorization")
	if accessToken == "" || !strings.HasPrefix(accessToken, "Bearer ") {
		return "", fmt.Errorf("malformed access token")
	}

	parts := strings.Split(accessToken, " ")
	if len(parts) != 2 {
		return "", fmt.Errorf("malformed access token")
	}

	accessToken = parts[1]
	return accessToken, nil
}

// GetKey returns the Razorpay public key for the authenticated user.
//
//	@Summary      Get Razorpay key
//	@Description  Returns the Razorpay public key ID. Requires a valid JWT bearer token.
//	@Tags         payments
//	@Produce      json
//	@Param        Authorization  header    string              true  "Bearer <token>"
//	@Success      200            {object}  map[string]string
//	@Failure      401            {object}  ErrorMsg
//	@Router       /orders/get-key [get]
func (con *Controller) GetKey(c fiber.Ctx) error {
	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	_, err = con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"key": config.Config().RazorpayKeyID,
	})
}

// CreateOrder creates a new order and initiates a Razorpay payment.
//
//	@Summary      Create order
//	@Description  Creates a new order for the authenticated customer and returns a Razorpay order to complete payment. Requires a valid JWT bearer token.
//	@Tags         orders
//	@Accept       json
//	@Produce      json
//	@Param        Authorization  header    string            true  "Bearer <token>"
//	@Param        body           body      dto.OrderRequest  true  "Order details"
//	@Success      201            {object}  dto.OrderResponse
//	@Failure      400            {object}  ErrorMsg
//	@Failure      401            {object}  ErrorMsg
//	@Failure      500            {object}  ErrorMsg
//	@Router       /orders/create [post]
func (con *Controller) CreateOrder(c fiber.Ctx) error {
	var orderRequest dto.OrderRequest

	if err := c.Bind().Body(&orderRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	o, err := con.s.CreateOrder(accessToken, userClaims, &orderRequest)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(o)
}

// PaymentCallback handles the Razorpay payment callback and verifies the signature.
//
//	@Summary      Razorpay payment callback
//	@Description  Receives the Razorpay payment callback form post, verifies the HMAC signature, and redirects to success or failure page.
//	@Tags         payments
//	@Accept       mpfd
//	@Param        razorpay_order_id    formData  string  true  "Razorpay Order ID"
//	@Param        razorpay_payment_id  formData  string  true  "Razorpay Payment ID"
//	@Param        razorpay_signature   formData  string  true  "Razorpay Signature"
//	@Success      302
//	@Failure      302
//	@Router       /orders/payment-callback [post]
func (con *Controller) PaymentCallback(c fiber.Ctx) error {
	razorpayOrderID := c.FormValue("razorpay_order_id")
	razorpayPaymentID := c.FormValue("razorpay_payment_id")
	razorpaySignature := c.FormValue("razorpay_signature")

	params := map[string]any{
		"razorpay_order_id":   razorpayOrderID,
		"razorpay_payment_id": razorpayPaymentID,
	}

	isValid := utils.VerifyPaymentSignature(params, razorpaySignature, config.Config().RazorpaySecret)

	if !isValid {
		return c.Redirect().To("/orders/failure.html")
	}

	return c.Redirect().To("/orders/success.html?orderId=" + razorpayOrderID + "&paymentId=" + razorpayPaymentID + "&signature=" + razorpaySignature)
}

// GetOrderByID fetches a single order by its ID.
//
//	@Summary      Get order by ID
//	@Description  Fetches a single order by its MongoDB ObjectID. Requires a valid JWT bearer token.
//	@Tags         orders
//	@Produce      json
//	@Param        Authorization  header    string            true  "Bearer <token>"
//	@Param        id             path      string            true  "Order ID (MongoDB ObjectID)"
//	@Success      200            {object}  dto.OrderResponse
//	@Failure      400            {object}  ErrorMsg
//	@Failure      401            {object}  ErrorMsg
//	@Failure      404            {object}  ErrorMsg
//	@Failure      500            {object}  ErrorMsg
//	@Router       /orders/order/{id} [get]
func (con *Controller) GetOrderByID(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorMsg{
			Message: "invalid order id",
		})
	}

	pID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorMsg{
			Message: "invalid order id",
		})
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	o, err := con.s.GetOrderByID(pID, userClaims)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(o)
}

// GetOrdersByCustomerID returns all orders for the authenticated customer.
//
//	@Summary      Get my orders
//	@Description  Returns all orders belonging to the currently authenticated customer. Requires a valid JWT bearer token.
//	@Tags         orders
//	@Produce      json
//	@Param        Authorization  header    string              true  "Bearer <token>"
//	@Success      200            {array}   dto.OrderResponse
//	@Failure      401            {object}  ErrorMsg
//	@Failure      500            {object}  ErrorMsg
//	@Router       /orders/get_my_orders [get]
func (con *Controller) GetOrdersByCustomerID(c fiber.Ctx) error {
	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	orders, err := con.s.GetOrdersByCustomerID(userClaims)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(&ErrorMsg{
			Message: "internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(orders)
}

// UpdateDeliveryAddress updates the delivery address of an existing order.
//
//	@Summary      Update delivery address
//	@Description  Updates the delivery address for a specific order. Only the owner of the order may do this. Requires a valid JWT bearer token.
//	@Tags         orders
//	@Accept       json
//	@Produce      json
//	@Param        Authorization  header    string  true  "Bearer <token>"
//	@Param        id             path      string  true  "Order ID (MongoDB ObjectID)"
//	@Param        body           body      object  true  "New delivery address"
//	@Success      204
//	@Failure      400            {object}  ErrorMsg
//	@Failure      401            {object}  ErrorMsg
//	@Failure      500            {object}  ErrorMsg
//	@Router       /orders/order/{id} [put]
func (con *Controller) UpdateDeliveryAddress(c fiber.Ctx) error {
	var request struct {
		Address string
	}

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorMsg{
			Message: "invalid order id",
		})
	}

	if err := c.Bind().JSON(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorMsg{
			Message: "invalid input",
		})
	}

	if request.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorMsg{
			Message: "invalid or no address",
		})
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{Message: err.Error()})
	}
	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{Message: err.Error()})
	}

	if err := con.s.UpdateDeliveryAddress(id, userClaims, request.Address); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// CancelOrder cancels an existing order by ID.
//
//	@Summary      Cancel an order
//	@Description  Cancels a specific order. Only the owner of the order can cancel it. Requires a valid JWT bearer token.
//	@Tags         orders
//	@Produce      json
//	@Param        Authorization  header    string  true  "Bearer <token>"
//	@Param        order_id       path      string  true  "Order ID (MongoDB ObjectID)"
//	@Success      204
//	@Failure      400            {object}  ErrorMsg
//	@Failure      401            {object}  ErrorMsg
//	@Failure      500            {object}  ErrorMsg
//	@Router       /orders/order/{order_id} [delete]
func (con *Controller) CancelOrder(c fiber.Ctx) error {
	orderID, err := primitive.ObjectIDFromHex(c.Params("order_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&ErrorMsg{
			Message: "invalid order id",
		})
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}
	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(&ErrorMsg{
			Message: err.Error(),
		})
	}

	if err := con.s.CancelOrder(orderID, userClaims.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(&ErrorMsg{
			Message: "internal server error",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (con *Controller) PaymentSuccess(c fiber.Ctx) error {
	rzpOrderID := c.Query("orderId")
	paymentID := c.Query("paymentId")
	signature := c.Query("signature")

	data := fiber.Map{
		"OrderID":   rzpOrderID,
		"PaymentID": paymentID,
		"Signature": signature,
	}

	return c.Render("success", data)
}

func (con *Controller) PaymentFailure(c fiber.Ctx) error {
	return c.Render("failure", nil)
}
