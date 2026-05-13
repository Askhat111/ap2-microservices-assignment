package usecase

import (
	"fmt"
	"log"

	"github.com/google/uuid"

	"payment-service/internal/domain"
	"payment-service/internal/publisher"
)

type PaymentUseCase struct {
	repo      domain.PaymentRepository
	publisher publisher.Publisher
}

func NewPaymentUseCase(repo domain.PaymentRepository, pub publisher.Publisher) *PaymentUseCase {
	return &PaymentUseCase{repo: repo, publisher: pub}
}

func (uc *PaymentUseCase) Process(orderID string, amount int64) (*domain.Payment, error) {
	status := "Authorized"
	if amount > 100000 {
		status = "Declined"
	}

	payment := &domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		TransactionID: "txn_" + uuid.NewString(),
		Amount:        amount,
		Status:        status,
	}

	if err := uc.repo.Save(payment); err != nil {
		return nil, err
	}

	event := publisher.PaymentEvent{
		EventID:       uuid.NewString(),
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		CustomerEmail: fmt.Sprintf("customer_%s@gmail.com", orderID),
		Status:        payment.Status,
	}

	if err := uc.publisher.Publish(event); err != nil {
		log.Printf("[Warning] Failed to publish event for order %s: %v", orderID, err)
	}

	return payment, nil
}

func (uc *PaymentUseCase) GetPaymentStatus(orderID string) (*domain.Payment, error) {
	return uc.repo.GetByOrderID(orderID)
}

func (uc *PaymentUseCase) GetPaymentsByStatus(status string) ([]*domain.Payment, error) {
	return uc.repo.ListByStatus(status)
}
