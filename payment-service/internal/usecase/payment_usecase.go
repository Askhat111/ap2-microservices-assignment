package usecase

import (
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUseCase struct {
	repo domain.PaymentRepository
}

func NewPaymentUseCase(repo domain.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
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

	err := uc.repo.Save(payment)
	return payment, err
}

func (uc *PaymentUseCase) GetPaymentStatus(orderID string) (*domain.Payment, error) {
	return uc.repo.GetByOrderID(orderID)
}
