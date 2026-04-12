package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"

	"order-service/internal/gateway"
	"order-service/internal/repository"
	grpctransport "order-service/internal/transport/grpc"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	//Database Configuration
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=55432 user=order_user password=secretpassword dbname=order_db sslmode=disable"
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	repo := repository.NewPostgresOrderRepository(db)

	//gRPC Client Configuration (Connection to Payment Service)
	paymentAddr := os.Getenv("PAYMENT_SERVICE_ADDR")
	if paymentAddr == "" {
		paymentAddr = "localhost:50051"
	}
	paymentGateway, err := gateway.NewPaymentGateway(paymentAddr)
	if err != nil {
		log.Fatalf("Failed to initialize Payment Gateway: %v", err)
	}

	//Channel for Real-time Status Updates (Streaming)
	updateChan := make(chan *basepb.OrderStatusUpdate, 100)

	//UseCase Initialization
	uc := usecase.NewOrderUseCase(repo, paymentGateway, updateChan)

	//REST Server Startup (Gin) in a separate goroutine
	handler := httptransport.NewOrderHandler(uc)
	r := gin.Default()
	r.Use(cors.Default())
	httptransport.RegisterRoutes(r, handler)

	go func() {
		restPort := os.Getenv("REST_PORT")
		if restPort == "" {
			restPort = ":8080"
		}
		log.Printf("REST Server is running on %s", restPort)
		if err := r.Run(restPort); err != nil && err != http.ErrServerClosed {
			log.Fatalf("REST Server error: %v", err)
		}
	}()

	//gRPC Server Startup (Order Tracking Streaming)
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = ":50052"
	}
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpctransport.LoggingInterceptor),
	)

	streamHandler := grpctransport.NewOrderStreamHandler(updateChan)
	servicepb.RegisterOrderTrackingServiceServer(grpcServer, streamHandler)

	log.Printf("Order gRPC Streaming Server is running on %s", grpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC Server error: %v", err)
	}
}
