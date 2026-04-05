package config

import (
	"os"
	"strings"
	"sync"
)

var (
	defaultConfig *config
	once          sync.Once
)

type config struct {
	Port            string
	Brokers         []string
	DSN             string
	CacheAddr       string
	CachePasswd     string
	ServiceRegistry string
	EurekaHostname  string
	RazorpayKeyID   string
	RazorpaySecret  string
}

func Init() {
	once.Do(func() {
		brokers := os.Getenv("BROKERS")
		defaultConfig = &config{
			Brokers:         strings.Split(brokers, ","),
			Port:            os.Getenv("PORT"),
			DSN:             os.Getenv("DSN"),
			CacheAddr:       os.Getenv("CACHE_ADDR"),
			CachePasswd:     os.Getenv("CACHE_PASSWD"),
			ServiceRegistry: os.Getenv("SERVICE_REGISTRY"),
			EurekaHostname:  os.Getenv("EUREKA_HOSTNAME"),
			RazorpayKeyID:   os.Getenv("RAZORPAY_KEY_ID"),
			RazorpaySecret:  os.Getenv("RAZORPAY_SECRET"),
		}
	})
}

func Config() *config {
	return defaultConfig
}
