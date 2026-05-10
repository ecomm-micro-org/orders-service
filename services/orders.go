package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hudl/fargo"
	"github.com/razorpay/razorpay-go"
	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/risbern21/ecom/orders/internal/dto"
	"github.com/risbern21/ecom/orders/internal/kafka"
	"github.com/risbern21/ecom/orders/internal/notifications"
	"github.com/risbern21/ecom/orders/internal/token"
	"github.com/risbern21/ecom/orders/models"
	"github.com/risbern21/ecom/orders/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	orderCreated = "Your order has been created."
	// addressUpdated = "Your address has been successfully updated."
)

type OrderService struct {
	store      store.Storer
	eurekaConn fargo.EurekaConnection
	RzpClient  *razorpay.Client
	Notifier   *notifications.Notifier
}

func NewOrderService(rzpClient *razorpay.Client, notifier *notifications.Notifier, s store.Storer, eurekaConn fargo.EurekaConnection) *OrderService {
	return &OrderService{
		Notifier:   notifier,
		store:      s,
		eurekaConn: eurekaConn,
		RzpClient:  rzpClient,
	}
}

func NewOrderServiceWithRepo(s store.Storer) *OrderService {
	serviceRegistry := config.Config().ServiceRegistry

	return &OrderService{
		store:      s,
		eurekaConn: fargo.NewConn(serviceRegistry),
	}
}

func NewWithEurekaConn(c fargo.EurekaConnection) *OrderService {
	return &OrderService{
		eurekaConn: c,
	}
}

func getReliableServiceURL(eurekaConn fargo.EurekaConnection, appName string) (string, error) {
	app, err := eurekaConn.GetApp(appName)
	if err != nil {
		return "", fmt.Errorf("eureka lookup failed for %s: %w", appName, err)
	}
	if app == nil || len(app.Instances) == 0 {
		return "", fmt.Errorf("no instances registered for %s", appName)
	}

	// Shuffle instances to load balance + avoid always hitting a stale one
	instances := app.Instances
	rand.Shuffle(len(instances), func(i, j int) {
		instances[i], instances[j] = instances[j], instances[i]
	})

	for _, inst := range instances {
		if inst.Status != fargo.UP {
			continue
		}

		rawURL := inst.HomePageUrl
		if rawURL == "" {
			rawURL = fmt.Sprintf("%s:%d", inst.IPAddr, inst.Port)
		}
		if !strings.HasPrefix(rawURL, "http") {
			rawURL = "http://" + rawURL
		}

		if isReachable(rawURL) {
			return strings.TrimRight(rawURL, "/"), nil
		}
	}
	return "", fmt.Errorf("all instances of %s are unreachable", appName)
}

// TCP reachability check
func isReachable(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Host
	if u.Port() == "" {
		host += ":80"
	}
	conn, err := net.DialTimeout("tcp", host, 300*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (o *OrderService) CreateOrder(accessToken string, userClaims *token.UserClaims, orderRequest *dto.OrderRequest) (*dto.OrderResponse, error) {
	var url string
	var err error

	// ctx1, cancel1 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	// defer cancel1()

	// url, err = cache.Client().Get(ctx1, "PRODUCTS-SERVICE").Result()
	// if err != nil {

	// Retry up to 3 times — Eureka can be flaky on first call
	for attempt := 1; attempt <= 3; attempt++ {
		url, err = getReliableServiceURL(o.eurekaConn, "PRODUCTS-SERVICE")
		if err == nil {
			break
		}
		log.Printf("attempt %d: failed to resolve PRODUCTS-SERVICE: %v", attempt, err)
		time.Sleep(time.Duration(attempt) * 100 * time.Millisecond) // backoff
	}
	if err != nil || url == "" {
		return nil, fmt.Errorf("could not resolve PRODUCTS-SERVICE after retries: %w", err)
	}

	// 	ctx2, cancel2 := context.WithTimeout(context.Background(), 300*time.Millisecond)
	// 	defer cancel2()
	// 	cache.Client().Set(ctx2, "PRODUCTS-SERVICE", url, 10*time.Second)
	// }
	//
	body, err := json.Marshal(orderRequest.OrderItems)
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

	m := models.NewOrder()
	m.ID = primitive.NewObjectID()
	m.CustomerID = userClaims.ID
	m.Address = orderRequest.Address
	m.Pincode = orderRequest.Pincode
	m.OrderItems = orderRequest.OrderItems
	m.CheckoutTotal = checkoutTotal
	m.Currency = orderRequest.Currency

	data := map[string]any{
		"amount":   m.CheckoutTotal,
		"currency": orderRequest.Currency,
		"receipt":  "order_" + strconv.FormatInt(time.Now().Unix(), 10),
	}

	rzpOrder, err := o.RzpClient.Order.Create(data, nil)
	if err != nil {
		return nil, err
	}

	m.RzpID = rzpOrder["id"].(string)

	if err := o.store.CreateOrder(m); err != nil {
		return nil, err
	}

	orderResponse := &dto.OrderResponse{}

	orderResponse.ID = m.ID
	orderResponse.CustomerID = m.CustomerID
	orderResponse.OrderItems = m.OrderItems
	orderResponse.Address = m.Address
	orderResponse.CheckoutTotal = checkoutTotal
	orderResponse.Pincode = m.Pincode
	orderResponse.IsPaid = m.IsPaid
	orderResponse.IsDelivered = m.IsDelivered
	orderResponse.Currency = m.Currency

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := kafka.Client().Publish(ctx, kafka.TopicOrderCreated.String(), m.ID.String(), m.OrderItems); err != nil {
			log.Println("error occurred while publishing", err)
		}
	}()

	orderResponse.RzpID = rzpOrder["id"].(string)

	if err := o.Notifier.SendNotificationToUser("order-placed", orderCreated); err != nil {
		log.Println("error while placing order ", err)
	}

	return orderResponse, nil
}

func (o *OrderService) GetOrderByID(id primitive.ObjectID, userClaims *token.UserClaims) (*dto.OrderResponse, error) {
	m, err := o.store.GetOrderByID(id)
	if err != nil {
		return nil, err
	}

	if userClaims.ID != m.CustomerID {
		return nil, fmt.Errorf("you do not have authorization to access this resource")
	}

	orderResponse := &dto.OrderResponse{}
	orderResponse.ID = m.ID
	orderResponse.CustomerID = m.CustomerID
	orderResponse.RzpID = m.RzpID
	orderResponse.Currency = m.Currency
	orderResponse.OrderItems = m.OrderItems
	orderResponse.Address = m.Address
	orderResponse.Pincode = m.Pincode
	orderResponse.CheckoutTotal = m.CheckoutTotal
	orderResponse.IsDelivered = m.IsDelivered
	orderResponse.IsPaid = m.IsPaid

	return orderResponse, nil
}

func (o *OrderService) GetOrdersByCustomerID(userClaims *token.UserClaims) ([]dto.OrderResponse, error) {
	orders, err := o.store.GetOrdersByCustomerID(userClaims.ID)
	if err != nil {
		return nil, err
	}

	var orderResponses []dto.OrderResponse

	for _, v := range orders {
		orderResponse := dto.OrderResponse{}

		orderResponse.ID = v.ID
		orderResponse.CustomerID = v.CustomerID
		orderResponse.RzpID = v.RzpID
		orderResponse.Currency = v.Currency
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

func (o *OrderService) UpdateDeliveryAddress(id primitive.ObjectID, userClaims *token.UserClaims, address string) error {
	m, err := o.store.GetOrderByID(id)
	if err != nil {
		return err
	}
	if m.CustomerID != userClaims.ID {
		return fmt.Errorf("you do not have enough permissions to access this resource")
	}

	return o.store.UpdateDeliveryAddress(id, address)
}

func (o *OrderService) UpdatePaymentStatus(id primitive.ObjectID, isPaid bool) error {
	return o.store.UpdatePaymentStatus(id, isPaid)
}

func (o *OrderService) UpdateDeliveryStatus(id primitive.ObjectID, isDelivered bool) error {
	return o.store.UpdateDeliveryStatus(id, isDelivered)
}

func (o *OrderService) CancelOrder(id primitive.ObjectID, customerID uuid.UUID) error {
	m, err := o.store.GetOrderByID(id)
	if err != nil {
		return err
	}

	if m.CustomerID != customerID {
		return fmt.Errorf("you do not have access to this order")
	}

	if err := o.store.CancelOrder(id); err != nil {
		return err
	}

	return nil
}
