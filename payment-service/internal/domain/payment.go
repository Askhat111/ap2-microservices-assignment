package domain

type Payment struct {
	ID            string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        string // "Authorized", "Declined"
}

type PaymentRepository interface {
	Save(payment *Payment) error
	GetByOrderID(orderID string) (*Payment, error)
}
