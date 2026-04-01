package main

import (
	"database/sql"
	"log"
	"order-service/internal/gateway"
	"order-service/internal/repository"
	"order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-contrib/cors"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=127.0.0.1 port=55432 user=order_user password=secretpassword dbname=order_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewPostgresOrderRepository(db)
	paymentGateway := gateway.NewHTTPPaymentGateway("http://localhost:8081")
	uc := usecase.NewOrderUseCase(repo, paymentGateway)
	handler := http.NewOrderHandler(uc)

	r := gin.Default()
	r.Use(cors.Default())

	http.RegisterRoutes(r, handler)
	r.Run(":8080")
}
