package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/risbern21/runaway/orders-service/gen/pb"
	"github.com/risbern21/runaway/orders-service/internal/config"
	"github.com/risbern21/runaway/orders-service/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type OrderHandler struct {
	pb.UnimplementedOrdersServiceServer
	orderService *services.OrderService
}

func NewOrderHandler(os *services.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: os,
	}
}

func (h *OrderHandler) GetOrderByID(ctx context.Context, req *pb.GetOrderByIDRequest) (*pb.GetOrderByIDResponse, error) {
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}

	userID, err := uuid.Parse(ctx.Value("userID").(string))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user id")
	}

	res, err := h.orderService.GetOrderByID(id, userID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, "something went wrong")
	}
	return res, nil
}

func (h *OrderHandler) GetOrdersByCustomerID(_ *emptypb.Empty, stream grpc.ServerStreamingServer[pb.GetOrdersByCustomerIDResponse]) error {
	ctx := stream.Context()
	userID, err := uuid.Parse(ctx.Value("userID").(string))
	if err != nil {
		return status.Error(codes.Unauthenticated, "invalid user id")
	}

	return h.orderService.GetOrdersByCustomerID(userID, stream)
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	userID, err := uuid.Parse(ctx.Value("userID").(string))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user id")
	}

	res, err := h.orderService.CreateOrder(ctx, userID, req)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("something went wrong try again later %v", err.Error()))
	}
	return res, nil
}

func (h *OrderHandler) GetKey(ctx context.Context, req *emptypb.Empty) (*pb.GetKeyResponse, error) {
	return &pb.GetKeyResponse{
		Key: config.Config().RazorpayKeyID,
	}, nil
}

func (h *OrderHandler) PaymentSuccess(ctx context.Context, req *pb.PaymentSuccessRequest) (*pb.PaymentSuccessResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method PaymentSuccess not implemented")
}

func (h *OrderHandler) PaymentFailure(ctx context.Context, req *pb.PaymentFailureRequest) (*pb.PaymentFailureResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method PaymentFailure not implemented")
}

func (h *OrderHandler) PaymentCallback(ctx context.Context, req *emptypb.Empty) (*pb.PaymentCallbackResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method PaymentCallback not implemented")
}

func (h *OrderHandler) UpdateDeliveryAddress(ctx context.Context, req *pb.UpdateDeliveryAddressRequest) (*emptypb.Empty, error) {
	id, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order id")
	}

	userID, err := uuid.Parse(ctx.Value("userID").(string))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user id")
	}

	if err := h.orderService.UpdateDeliveryAddress(id, userID, req.Address); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Error(codes.Internal, "something went wrong")
	}

	return &emptypb.Empty{}, nil
}

func (h *OrderHandler) CancelOrder(ctx context.Context, req *emptypb.Empty) (*pb.CancelOrderResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method CancelOrder not implemented")
}
