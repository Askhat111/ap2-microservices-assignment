package domain

import "time"

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        string // "Authorized", "Declined"
	CreatedAt     time.Time
}

type PaymentRepository interface {
	Save(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
	ListByStatus(status string) ([]*Payment, error)
}
