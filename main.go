package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hudl/fargo"
	"github.com/op/go-logging"
	"github.com/razorpay/razorpay-go"
	"github.com/risbern21/ecom/orders/controllers"
	_ "github.com/risbern21/ecom/orders/docs"
	"github.com/risbern21/ecom/orders/internal/cache"
	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/risbern21/ecom/orders/internal/database"
	"github.com/risbern21/ecom/orders/internal/kafka"
	"github.com/risbern21/ecom/orders/internal/notifications"
	"github.com/risbern21/ecom/orders/internal/server"
	"github.com/risbern21/ecom/orders/services"
	"github.com/risbern21/ecom/orders/store"
)

func heartBeat(conn fargo.EurekaConnection, instance fargo.Instance, l *logging.Logger) {
	for {
		err := conn.HeartBeatInstance(&instance)
		if err != nil {
			l.Errorf("Heartbeat failed:", err)
		} else {
			l.Info("Heartbeat sent")
		}

		time.Sleep(30 * time.Second)
	}
}

func consumeFromKafka(s *services.OrderService) {
	errs := make(chan error, 10)

	pc := kafka.NewConsumer(kafka.TopicPayments)
	defer pc.Close()

	dc := kafka.NewConsumer(kafka.TopicDeliveries)
	defer dc.Close()

	go pc.ConsumeTopicPayments(s.UpdatePaymentStatus, errs)

	go dc.ConsumeTopicDeliveries(s.UpdateDeliveryStatus, errs)

	for err := range errs {
		log.Printf("error occurred :%v\n", err)
	}
}

// @title orders microservice API
// @version 1.0
// @description This is a orders server for ecomm micro project
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:42067
// @BasePath /
func main() {
	config.Init()
	f, err := os.OpenFile("/tmp/orders-service-eureka.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("unable to open log file /tmp/orders-service-eureka.log")
	}
	defer f.Close()

	backend := logging.NewLogBackend(f, "", 0)
	logging.SetBackend(backend)

	serviceRegistry := config.Config().ServiceRegistry
	c := fargo.NewConn(serviceRegistry)
	instance := fargo.Instance{
		InstanceId:       "orders-service",
		HostName:         config.Config().EurekaHostname,
		App:              "ORDERS-SERVICE",
		IPAddr:           "localhost",
		VipAddress:       "ORDERS-SERVICE",
		SecureVipAddress: "ORDERS-SERVICE",
		Status:           fargo.UP,
		Port:             42067,
		PortEnabled:      true,
		DataCenterInfo: fargo.DataCenterInfo{
			Name: fargo.MyOwn,
		},
		LeaseInfo: fargo.LeaseInfo{
			RenewalIntervalInSecs: 30,
			DurationInSecs:        90,
		},
	}

	// Register with Eureka
	err = c.RegisterInstance(&instance)
	if err != nil {
		log.Fatal("Failed to register:", err)
	}

	l := logging.MustGetLogger("products")
	go heartBeat(c, instance, l)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("deregistering from eureka")
		c.DeregisterInstance(&instance)

		database.Disconnect()
		cache.Disconnect()
		os.Exit(0)
	}()

	database.Connect()
	cache.Connect()

	kafka.Init(config.Config().Brokers)
	defer kafka.Close()

	secretKey, ok := os.LookupEnv("SECRET_KEY")
	if !ok {
		log.Fatalf("secret key not found")
	}

	// create service and contrlloer instances
	rc := razorpay.NewClient(config.Config().RazorpayKeyID, config.Config().RazorpaySecret)
	n := notifications.NewNotifier()
	ms := store.NewMongoStore()

	os := services.NewOrderService(rc, n, ms, c)

	oc := controllers.NewController(secretKey, os)

	server.SetUp(oc)

	go consumeFromKafka(os)

	app := server.New()

	if err := app.Listen(config.Config().Port); err != nil {
		log.Fatalf("err : %v", err)
	}
}
