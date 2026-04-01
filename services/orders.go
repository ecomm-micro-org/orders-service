package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hudl/fargo"
	"github.com/risbern21/ecom/orders/internal/cache"
	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/risbern21/ecom/orders/internal/dto"
	"github.com/risbern21/ecom/orders/internal/kafka"
	"github.com/risbern21/ecom/orders/internal/token"
	"github.com/risbern21/ecom/orders/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderService struct {
	UserClaims       *token.UserClaims
	OrderReqeustDTO  dto.OrderReqeustDTO
	OrderResponseDTO dto.OrderResponseDTO
}

func New() *OrderService {
	return &OrderService{}
}

func GetServiceURL(conn fargo.EurekaConnection, serviceName string) (string, error) {
	app, err := conn.GetApp(serviceName)
	if err != nil || len(app.Instances) == 0 {
		return "", fmt.Errorf("no instances for %s", serviceName)
	}
	instance := app.Instances[rand.Intn(len(app.Instances))]
	return fmt.Sprintf("http://%s:%d", instance.HostName, instance.Port), nil
}

func (o *OrderService) CreateOrder(accessToken string) (*dto.OrderResponseDTO, error) {
	var url string
	var err error

	ctx1, cancel1 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel1()
	url, err = cache.Client().Get(ctx1, "PRODUCTS-SERVICE").Result()
	if err != nil {
		serviceRegistry := config.Config().ServiceRegistry

		eurekaConn := fargo.NewConn(serviceRegistry)
		url, err = GetServiceURL(eurekaConn, "PRODUCTS-SERVICE")
		if err != nil {
			return nil, err
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel2()
		cache.Client().Set(ctx2, "PRODUCTS-SERVICE", url, 5*time.Minute)
	}

	body, err := json.Marshal(o.OrderReqeustDTO.OrderItems)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	//send request to product service to get fetch total order price  TODO: needs some fixes in the products service side
	req, err := http.NewRequest("POST", url+"/products/calculate_total_price", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	checkoutTotal, err := strconv.ParseFloat(string(respBody), 64)
	if err != nil {
		return nil, err
	}

	m := models.New()
	m.ID = primitive.NewObjectID()
	m.CustomerID = o.UserClaims.ID
	m.Address = o.OrderReqeustDTO.Address
	m.Pincode = o.OrderReqeustDTO.Pincode
	m.OrderItems = o.OrderReqeustDTO.OrderItems
	m.CheckoutTotal = checkoutTotal

	if err := m.CreateOrder(); err != nil {
		return nil, err
	}

	orderResponseDTO := &dto.OrderResponseDTO{}

	orderResponseDTO.ID = m.ID
	orderResponseDTO.CustomerID = m.CustomerID
	orderResponseDTO.OrderItems = m.OrderItems
	orderResponseDTO.Address = m.Address
	orderResponseDTO.CheckoutTotal = checkoutTotal
	orderResponseDTO.Pincode = m.Pincode
	orderResponseDTO.IsPaid = m.IsPaid
	orderResponseDTO.IsDelivered = m.IsDelivered

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := kafka.Client().Publish(ctx, kafka.TopicOrderCreated.String(), m.ID.String(), m.OrderItems); err != nil {
			log.Println("error occurred while publishing", err)
		}
	}()

	return orderResponseDTO, nil
}

func (o *OrderService) GetOrderByID(id primitive.ObjectID) error {
	m := models.New()
	m.ID = id

	if err := m.GetOrderByID(); err != nil {
		return err
	}

	if o.UserClaims.ID != m.CustomerID {
		return fmt.Errorf("you do not have authorization to access this resource")
	}

	o.OrderResponseDTO.ID = m.ID
	o.OrderResponseDTO.CustomerID = m.CustomerID
	o.OrderResponseDTO.OrderItems = m.OrderItems
	o.OrderResponseDTO.Address = m.Address
	o.OrderResponseDTO.Pincode = m.Pincode
	o.OrderResponseDTO.CheckoutTotal = m.CheckoutTotal
	o.OrderResponseDTO.IsDelivered = m.IsDelivered
	o.OrderResponseDTO.IsPaid = m.IsPaid

	return nil
}

func (o *OrderService) GetOrdersByCustomerID() ([]dto.OrderResponseDTO, error) {
	m := models.New()
	m.CustomerID = o.UserClaims.ID

	orders, err := m.GetOrdersByCustomerID()
	if err != nil {
		return nil, err
	}

	var orderResponses []dto.OrderResponseDTO

	for _, v := range orders {
		orderResponse := dto.OrderResponseDTO{}

		orderResponse.ID = v.ID
		orderResponse.CustomerID = v.CustomerID
		orderResponse.OrderItems = v.OrderItems
		orderResponse.Address = v.Address
		orderResponse.Pincode = v.Pincode
		orderResponse.CheckoutTotal = v.CheckoutTotal
		orderResponse.IsDelivered = v.IsDelivered
		orderResponse.IsPaid = v.IsPaid

		orderResponses = append(orderResponses, orderResponse)
	}

	return orderResponses, nil
}

func (o *OrderService) UpdateDeliveryAddress(id primitive.ObjectID, address string) error {
	m := models.New()
	m.ID = id
	m.Address = address
	return m.UpdateDeliveryAddress()
}

func (o *OrderService) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	m := models.New()
	m.ID = id
	m.IsPaid = isPaid
	return m.UpdatePaymentStatus()
}

func (o *OrderService) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	m := models.New()
	m.ID = id
	m.IsDelivered = isDelivered
	return m.UpdateDeliveryStatus()
}

func (O *OrderService) CancelOrder(id primitive.ObjectID, customerID uuid.UUID) error {
	m := models.New()
	m.ID = id
	if err := m.GetOrderByID(); err != nil {
		return err
	}

	if m.CustomerID != customerID {
		return fmt.Errorf("you do not have access to this order")
	}

	if err := m.CancelOrder(); err != nil {
		return err
	}

	return nil
}
