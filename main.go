package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ecomm-micro-org/orders-service/api"
	"github.com/ecomm-micro-org/orders-service/db"
	"github.com/ecomm-micro-org/orders-service/interceptors"
	"github.com/ecomm-micro-org/orders-service/internal/auth"
	"github.com/ecomm-micro-org/orders-service/internal/config"
	kafkaProducer "github.com/ecomm-micro-org/orders-service/internal/kafka"
	"github.com/ecomm-micro-org/orders-service/internal/messaging"
	"github.com/ecomm-micro-org/orders-service/pb"
	"github.com/ecomm-micro-org/orders-service/services"
	"github.com/ecomm-micro-org/orders-service/store"
	"github.com/razorpay/razorpay-go"
	"github.com/segmentio/kafka-go"
	"github.com/slack-go/slack"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	infoLogger *zap.Logger
)

func main() {
	config.Init()
	db.Connect()
	if err := initLogger(); err != nil {
		log.Fatalf("unable to init zap logger : %v\n", err)
	}

	am, err := auth.NewAuthManager(config.Config().SecretKey)
	if err != nil {
		infoLogger.Fatal("unable to create auth manager\n")
	}

	//interceptors
	li := interceptors.NewLoggingInterceptor(infoLogger)
	ai, err := interceptors.NewAuthInterceptor(am)
	if err != nil {
		infoLogger.Error("unable to create auth interceptor")
	}

	p := kafkaProducer.NewProducer(
		kafkaProducer.ProducerConfig{
			Brokers:      config.Config().Brokers,
			Topic:        kafkaProducer.TopicOrderCreated.String(),
			BatchSize:    100,
			BatchTimeout: 50,
			Async:        false,
			RequiredAcks: kafka.RequireAll,
		},
	)

	rzpClient := razorpay.NewClient(config.Config().RazorpayKeyID, config.Config().RazorpaySecret)

	sc := slack.New(config.Config().SlackToken)
	m, err := messaging.NewSlackMessenger(sc, config.Config().SlackChannel)
	if err != nil {
		infoLogger.Sugar().Fatalf("unable to create slack client : %v\n", err)
	}

	conn, err := grpc.NewClient(
		config.Config().ProductsClient,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(ai.ForwardMetadataInterceptor()),
		grpc.WithChainUnaryInterceptor(li.UnaryClientLoggingInterceptor()),
	)
	if err != nil {
		infoLogger.Sugar().Fatalf("unable to create grpc products client : %v\n", err)
	}

	pc := pb.NewProductsServiceClient(conn)

	s := store.NewMongoStore(db.Client(), "ordersDB", "orders")
	o := services.NewOrderService(s, p, m, pc, rzpClient)

	grpcServer := api.NewGRPCServer(o, li, ai)

	if err := runServer(context.Background(), grpcServer); err != nil {
		infoLogger.Sugar().Errorf("unable to run server : %v\n", err)
	}

	if err := conn.Close(); err != nil {
		infoLogger.Sugar().Errorf("couldnt close the grpc client connection : %v\n", err)
	}
	infoLogger.Sugar().Infoln("successfully closed the grpc client connection")
	if err := p.Close(); err != nil {
		infoLogger.Sugar().Errorf("couldnt close the kafka connection : %v\n", err)
	}
	infoLogger.Sugar().Infoln("successfully closed the kafka connection")
	if err := db.Disconnect(); err != nil {
		infoLogger.Sugar().Errorf("Failed to disconnect from MongoDB: %v\n", err)
	}
	infoLogger.Sugar().Infoln("successfully disconnected from MongoDB")
}

func runServer(ctx context.Context, grpcServer *grpc.Server) error {
	serverErr := make(chan error, 1)

	go func() {
		infoLogger.Sugar().Infoln("orders service running on port :42067")
		lis, err := net.Listen("tcp", ":42067")
		if err != nil {
			infoLogger.Sugar().Fatalf("unable to listen on port :42067\n")
		}

		if err := grpcServer.Serve(lis); errors.Is(err, grpc.ErrServerStopped) {
			serverErr <- err
		}
		close(serverErr)
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErr:
		return err
	case <-shutdown:
		infoLogger.Sugar().Infoln("shutdown signal received")
	case <-ctx.Done():
		infoLogger.Sugar().Infoln("parent context cancelled")
	}

	grpcServer.GracefulStop()

	infoLogger.Info("server exited successfully")
	return nil
}

func initLogger() error {
	var err error
	infoLogger, err = zap.NewProduction()
	if err != nil {
		return err
	}
	return nil
}
