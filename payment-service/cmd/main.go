package main

import (
	"database/sql"
	"log"
	"net"
	"os"

	"payment-service/internal/repository"
	grpctransport "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"

	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	//Database Configuration
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=55433 user=payment_user password=secretpassword dbname=payment_db sslmode=disable"
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo)
	handler := grpctransport.NewPaymentHandler(uc)

	//gRPC Server Configuration and Startup
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = ":50051"
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	//gRPC Server with Interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpctransport.LoggingInterceptor),
	)

	servicepb.RegisterPaymentServiceServer(grpcServer, handler)

	log.Printf("Payment gRPC Server is running on %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC Server startup error: %v", err)
	}
}
