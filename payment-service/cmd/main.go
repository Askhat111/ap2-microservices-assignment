package main

import (
	"database/sql"
	"log"
	"payment-service/internal/repository"
	"payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=127.0.0.1 port=55433 user=payment_user password=secretpassword dbname=payment_db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo)
	handler := http.NewPaymentHandler(uc)

	r := gin.Default()
	http.RegisterRoutes(r, handler)

	r.Run(":8081")
}
