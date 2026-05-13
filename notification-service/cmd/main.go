package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"notification-service/internal/consumer"
	"notification-service/internal/provider"
	"notification-service/internal/usecase"
)

func main() {
	_ = godotenv.Load()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is not set")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()

	var emailSender provider.EmailSender
	switch os.Getenv("PROVIDER_MODE") {
	case "REAL":
		log.Println("[Provider] Mode: REAL (10% failure rate)")
		emailSender = provider.NewSimulatedProvider(0.1, 200*time.Millisecond)
	default:
		log.Println("[Provider] Mode: SIMULATED (30% failure rate — tests retries)")
		emailSender = provider.NewSimulatedProvider(0.3, 100*time.Millisecond)
	}

	notifier := usecase.NewNotifier(emailSender, redisClient)

	c, err := consumer.NewRabbitMQConsumer(rabbitURL, notifier)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down Notification Service...")
		cancel()
		c.Close()
	}()

	log.Println("Notification Service started.")
	if err := c.Start(ctx); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
