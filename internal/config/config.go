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
	Port           string
	Brokers        []string
	DSN            string
	CacheAddr      string
	CachePasswd    string
	SecretKey      string
	RazorpayKeyID  string
	RazorpaySecret string
	CourierKey     string
	ProductsClient string
	SlackToken     string
	SlackChannel   string
}

func Init() {
	once.Do(func() {
		brokers := os.Getenv("BROKERS")
		defaultConfig = &config{
			Brokers:        strings.Split(brokers, ","),
			Port:           os.Getenv("PORT"),
			DSN:            os.Getenv("DSN"),
			CacheAddr:      os.Getenv("CACHE_ADDR"),
			CachePasswd:    os.Getenv("CACHE_PASSWD"),
			SecretKey:      os.Getenv("SECRET_KEY"),
			RazorpayKeyID:  os.Getenv("RAZORPAY_KEY_ID"),
			RazorpaySecret: os.Getenv("RAZORPAY_SECRET"),
			CourierKey:     os.Getenv("COURIER_KEY"),
			ProductsClient: os.Getenv("PRODUCTS_CLIENT"),
			SlackToken:     os.Getenv("SLACK_TOKEN"),
			SlackChannel:   os.Getenv("SLACK_CHANNEL"),
		}
	})
}

func Config() *config {
	return defaultConfig
}
