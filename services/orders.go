package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/razorpay/razorpay-go"
	"github.com/risbern21/runaway/orders-service/gen/pb"
	"github.com/risbern21/runaway/orders-service/internal/kafka"
	"github.com/risbern21/runaway/orders-service/internal/messaging"
	"github.com/risbern21/runaway/orders-service/models"
	"github.com/risbern21/runaway/orders-service/store"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
)

type OrderService struct {
	store          store.Storer
	producer       *kafka.Producer
	messenger      messaging.Messenger
	productsClient pb.ProductsServiceClient
	rzpClient      *razorpay.Client
}

func NewOrderService(s store.Storer, p *kafka.Producer, m messaging.Messenger, pc pb.ProductsServiceClient, rzpClient *razorpay.Client) *OrderService {
	return &OrderService{
		store:          s,
		producer:       p,
		messenger:      m,
		productsClient: pc,
		rzpClient:      rzpClient,
	}
}

func (o *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	r, err := o.productsClient.CalculateTotalPrice(
		ctx, &pb.CalculateTotalPriceRequest{
			OrderItems: req.OrderItems,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("")
	}

	m := models.NewOrder()

	m.ID = primitive.NewObjectID()
	m.CustomerID = userID
	m.Address = req.Address
	m.Pincode = req.Pincode
	m.OrderItems = req.OrderItems
	m.CheckoutTotal = r.TotalPrice
	m.Currency = req.Currency

	data := map[string]any{
		"amount":   int(m.CheckoutTotal),
		"currency": req.Currency,
		"receipt":  "order_" + strconv.FormatInt(time.Now().Unix(), 10),
	}

	rzpOrder, err := o.rzpClient.Order.Create(data, nil)
	if err != nil {
		return nil, err
	}

	m.RzpID = rzpOrder["id"].(string)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(m.OrderItems)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal order items")
	}

	if err := o.producer.Produce(ctx, []byte(m.ID.String()), body); err != nil {
		log.Printf("unable to write to kafka : %v\n", err)
	}

	if err := o.store.CreateOrder(m); err != nil {
		return nil, err
	}

	res := &pb.CreateOrderResponse{}

	res.Id = m.ID.Hex()
	res.CustomerId = m.CustomerID.String()
	res.OrderItems = m.OrderItems
	res.Address = m.Address
	res.CheckoutTotal = r.TotalPrice
	res.Pincode = m.Pincode
	res.IsPaid = m.IsPaid
	res.IsDelivered = m.IsDelivered
	res.Currency = m.Currency

	res.RzpId = rzpOrder["id"].(string)

	if err := o.messenger.SendMsg(fmt.Sprintf("your order %s has been created", res.Id)); err != nil {
		log.Println("unable to send message ", err)
	}

	return res, nil
}

func (o *OrderService) GetOrderByID(id primitive.ObjectID, userID uuid.UUID) (*pb.GetOrderByIDResponse, error) {
	m, err := o.store.GetOrderByID(id)
	if err != nil {
		return nil, err
	}

	if userID != m.CustomerID {
		return nil, fmt.Errorf("you do not have authorization to access this resource")
	}

	res := &pb.GetOrderByIDResponse{}
	res.Id = m.ID.Hex()
	res.CustomerId = m.CustomerID.String()
	res.Currency = m.Currency
	res.OrderItems = m.OrderItems
	res.Address = m.Address
	res.Pincode = m.Pincode
	res.CheckoutTotal = m.CheckoutTotal
	res.IsDelivered = m.IsDelivered
	res.IsPaid = m.IsPaid

	return res, nil
}

func (o *OrderService) GetOrdersByCustomerID(userID uuid.UUID, stream grpc.ServerStreamingServer[pb.GetOrdersByCustomerIDResponse]) error {
	orders, err := o.store.GetOrdersByCustomerID(userID)
	if err != nil {
		return err
	}

	for _, v := range orders {
		r := &pb.GetOrdersByCustomerIDResponse{}

		r.Id = v.ID.Hex()
		r.CustomerId = v.CustomerID.String()
		r.Currency = v.Currency
		r.OrderItems = v.OrderItems
		r.Address = v.Address
		r.Pincode = v.Pincode
		r.CheckoutTotal = v.CheckoutTotal
		r.IsDelivered = v.IsDelivered
		r.IsPaid = v.IsPaid

		if err := stream.Send(r); err != nil {
			return err
		}
	}

	return nil
}

func (o *OrderService) UpdateDeliveryAddress(id primitive.ObjectID, userID uuid.UUID, address string) error {
	m, err := o.store.GetOrderByID(id)
	if err != nil {
		return err
	}
	if m.CustomerID != userID {
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

func (o *OrderService) CancelOrder(id primitive.ObjectID, userID uuid.UUID) error {
	m, err := o.store.GetOrderByID(id)
	if err != nil {
		return err
	}

	if m.CustomerID != userID {
		return fmt.Errorf("you do not have access to this order")
	}

	if err := o.store.CancelOrder(id); err != nil {
		return err
	}

	return nil
}
