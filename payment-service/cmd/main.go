package main

import (
	"database/sql"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"payment-service/internal/publisher"
	"payment-service/internal/repository"
	grpctransport "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"
)

func main() {
	_ = godotenv.Load()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is not set")
	}
	pub, err := publisher.NewRabbitMQPublisher(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer pub.Close()

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo, pub)
	handler := grpctransport.NewPaymentHandler(uc)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = ":50051"
	}
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", port, err)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpctransport.LoggingInterceptor),
	)
	servicepb.RegisterPaymentServiceServer(grpcServer, handler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down Payment Service...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Payment gRPC server running on %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}
