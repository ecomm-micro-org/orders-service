package controllers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/razorpay/razorpay-go/utils"
	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/risbern21/ecom/orders/internal/dto"
	"github.com/risbern21/ecom/orders/internal/token"
	"github.com/risbern21/ecom/orders/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Controller struct {
	extractor *token.JWTExtractor
}

func NewController(secretKey string) *Controller {
	return &Controller{
		extractor: token.NewJWTExtractor(secretKey),
	}
}

func extractAccessToken(c fiber.Ctx) (string, error) {
	accessToken := c.Get("Authorization")
	if accessToken == "" || !strings.HasPrefix(accessToken, "Bearer ") {
		return "", fmt.Errorf("malformed access token")
	}

	accessToken = strings.Split(accessToken, " ")[1]

	return accessToken, nil
}

func (con *Controller) GetKey(c fiber.Ctx) error {
	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	_, err = con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"key": config.Config().RazorpayKeyID,
	})
}

func (con *Controller) CreateOrder(c fiber.Ctx) error {
	var orderRequestDTO dto.OrderReqeust

	if err := c.Bind().Body(&orderRequestDTO); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err.Error())
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	s := services.New()
	s.UserClaims = userClaims
	s.OrderReqeustDTO = orderRequestDTO
	o, err := s.CreateOrder(accessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(o)
}

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

func (con *Controller) GetOrderByID(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON("invalid id")
	}

	pID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON("invalid id")
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	s := services.New()
	s.UserClaims = userClaims
	if err := s.GetOrderByID(pID); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(s.OrderResponseDTO)
}

func (con *Controller) GetOrdersByCustomerID(c fiber.Ctx) error {
	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	s := services.New()
	s.UserClaims = userClaims
	orders, err := s.GetOrdersByCustomerID()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON("internal server error")
	}

	return c.Status(fiber.StatusOK).JSON(orders)
}

func (con *Controller) UpdateDeliveryAddress(c fiber.Ctx) error {
	var request struct {
		Address string
	}

	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON("invalid id")
	}

	if err := c.Bind().JSON(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON("invalid input data")
	}

	if request.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON("no address sent")
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}
	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	s := services.New()
	s.UserClaims = userClaims
	if err := s.UpdateDeliveryAddress(id, request.Address); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (con *Controller) CancelOrder(c fiber.Ctx) error {
	var req struct {
		CustomerID uuid.UUID `json:"customer_id"`
	}

	orderID, err := primitive.ObjectIDFromHex(c.Params("order_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON("invalid order id")
	}

	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	accessToken, err := extractAccessToken(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}
	userClaims, err := con.extractor.ExtractUserClaims(accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(err)
	}

	s := services.New()
	s.UserClaims = userClaims
	if err := s.CancelOrder(orderID, req.CustomerID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON("internal server error")
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
