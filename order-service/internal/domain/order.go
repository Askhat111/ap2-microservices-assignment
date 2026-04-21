package domain

import "time"

type Order struct {
	ID             string
	CustomerID     string
	ItemName       string
	Amount         int64  //cents
	Status         string // "Pending", "Paid", "Failed", "Cancelled"
	CreatedAt      time.Time
	IdempotencyKey string `json:"-"`
}

type OrderRepository interface {
	Create(order *Order) error
	GetByID(id string) (*Order, error)
	UpdateStatus(id, status string) error
	GetByIdempotencyKey(key string) (*Order, error)
}

type PaymentGateway interface {
	ProcessPayment(orderID string, amount int64) (string, error)
}
