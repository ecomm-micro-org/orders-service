package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hudl/fargo"
	"github.com/risbern21/ecom/orders/cmd/app"
	"github.com/risbern21/ecom/orders/internal/config"
	"github.com/risbern21/ecom/orders/internal/database"
)

func heartBeat(conn fargo.EurekaConnection, instance fargo.Instance) {
	for {
		err := conn.HeartBeatInstance(&instance)
		if err != nil {
			log.Println("Heartbeat failed:", err)
		} else {
			log.Println("Heartbeat sent")
		}

		time.Sleep(30 * time.Second)
	}
}

// TODO : implement swagger documentation
func main() {
	config.Init()

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
	err := c.RegisterInstance(&instance)
	if err != nil {
		log.Fatal("Failed to register:", err)
	}

	go heartBeat(c, instance)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("deregistering from eureka")
		c.DeregisterInstance(&instance)

		database.Disconnect()
		os.Exit(0)
	}()

	app.SetUp()
}
