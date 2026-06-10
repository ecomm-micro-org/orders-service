package api

import (
	"github.com/ecomm-micro-org/orders-service/handlers"
	"github.com/ecomm-micro-org/orders-service/interceptors"
	"github.com/ecomm-micro-org/orders-service/pb"
	"github.com/ecomm-micro-org/orders-service/services"
	"google.golang.org/grpc"
)

func NewGRPCServer(os *services.OrderService, li *interceptors.LoggingInterceptor, ai *interceptors.AuthInterceptor) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(li.UnaryLoggingInterceptor()),
		grpc.ChainUnaryInterceptor(ai.UnaryAuthInterceptor()),
		grpc.ChainStreamInterceptor(li.StreamLoggingInterceptor()),
		grpc.ChainStreamInterceptor(ai.StreamAuthInterceptor()),
	)

	pb.RegisterOrdersServiceServer(
		grpcServer,
		handlers.NewOrderHandler(os),
	)

	return grpcServer
}
