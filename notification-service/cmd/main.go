package main

import (
	"log"
	"notification-service/internal/consumer"
	"notification-service/internal/usecase"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is not set")
	}

	notifier := usecase.NewNotifier()

	c, err := consumer.NewRabbitMQConsumer(rabbitURL, notifier)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down Notification Service...")
		c.Close()
	}()

	log.Println("Notification Service started.")
	if err := c.Start(); err != nil {
		log.Fatalf("Consumer error: %v", err)
	}
}
