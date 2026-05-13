package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"order-service/internal/cache"
	"order-service/internal/gateway"
	"order-service/internal/repository"
	grpctransport "order-service/internal/transport/grpc"
	httptransport "order-service/internal/transport/http"
	"order-service/internal/transport/http/middleware"
	"order-service/internal/usecase"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	defer db.Close()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer redisClient.Close()

	repo := repository.NewPostgresOrderRepository(db)
	orderCache := cache.NewRedisOrderCache(redisClient)

	paymentAddr := os.Getenv("PAYMENT_SERVICE_ADDR")
	if paymentAddr == "" {
		log.Fatal("PAYMENT_SERVICE_ADDR is not set")
	}
	paymentGateway, err := gateway.NewPaymentGateway(paymentAddr)
	if err != nil {
		log.Fatalf("Payment gateway error: %v", err)
	}

	broadcaster := grpctransport.NewOrderBroadcaster()
	uc := usecase.NewOrderUseCase(repo, paymentGateway, broadcaster, orderCache)

	r := gin.Default()
	r.Use(cors.Default())
	r.Use(middleware.RateLimiter(redisClient, 10, time.Minute))
	httptransport.RegisterRoutes(r, httptransport.NewOrderHandler(uc))

	restPort := os.Getenv("REST_PORT")
	if restPort == "" {
		restPort = ":8080"
	}
	httpServer := &http.Server{Addr: restPort, Handler: r}
	go func() {
		log.Printf("REST server running on %s", restPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("REST server error: %v", err)
		}
	}()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = ":50052"
	}
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("gRPC listen error: %v", err)
	}
	grpcServer := grpc.NewServer()
	servicepb.RegisterOrderTrackingServiceServer(grpcServer, grpctransport.NewOrderStreamHandler(broadcaster))
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC streaming server running on %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Order Service...")
	grpcServer.GracefulStop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}
