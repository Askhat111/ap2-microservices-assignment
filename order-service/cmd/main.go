package main

import (
	"database/sql"
	"log"
	"net"
	"os"

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
	"google.golang.org/grpc/reflection"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=127.0.0.1 port=55432 user=order_user password=secretpassword dbname=order_db sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("DB Error: %v", err)
	}
	repo := repository.NewPostgresOrderRepository(db)

	paymentAddr := os.Getenv("PAYMENT_SERVICE_ADDR")
	if paymentAddr == "" {
		paymentAddr = "localhost:50051"
	}
	paymentGateway, err := gateway.NewPaymentGateway(paymentAddr)
	if err != nil {
		log.Fatalf("Payment Gateway Error: %v", err)
	}

	broadcaster := grpctransport.NewOrderBroadcaster()
	uc := usecase.NewOrderUseCase(repo, paymentGateway, broadcaster)

	//REST API
	r := gin.Default()
	r.Use(cors.Default())
	restHandler := httptransport.NewOrderHandler(uc)
	httptransport.RegisterRoutes(r, restHandler)

	go func() {
		restPort := os.Getenv("REST_PORT")
		if restPort == "" {
			restPort = ":8080"
		}
		log.Printf("REST Server running on %s", restPort)
		r.Run(restPort)
	}()

	//gRPC Server
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = ":50052"
	}
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("gRPC Listen Error: %v", err)
	}

	grpcServer := grpc.NewServer()
	streamHandler := grpctransport.NewOrderStreamHandler(broadcaster)
	servicepb.RegisterOrderTrackingServiceServer(grpcServer, streamHandler)
	reflection.Register(grpcServer)

	log.Printf("gRPC Streaming Server running on %s", grpcPort)
	grpcServer.Serve(lis)
}
