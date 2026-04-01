package usecase

import (
	"errors"
	"order-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

type OrderUseCase struct {
	repo    domain.OrderRepository
	gateway domain.PaymentGateway
}

func NewOrderUseCase(repo domain.OrderRepository, gateway domain.PaymentGateway) *OrderUseCase {
	return &OrderUseCase{repo: repo, gateway: gateway}
}

func (uc *OrderUseCase) CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	order := &domain.Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     "Pending",
		CreatedAt:  time.Now(),
	}

	if err := uc.repo.Create(order); err != nil {
		return nil, err
	}

	paymentStatus, err := uc.gateway.ProcessPayment(order.ID, order.Amount)
	if err != nil {
		uc.repo.UpdateStatus(order.ID, "Failed")
		return nil, errors.New("503 Service Unavailable")
	}

	newStatus := "Paid"
	if paymentStatus == "Declined" {
		newStatus = "Failed"
	}

	uc.repo.UpdateStatus(order.ID, newStatus)
	order.Status = newStatus

	return order, nil
}

func (uc *OrderUseCase) GetOrder(id string) (*domain.Order, error) {
	return uc.repo.GetByID(id)
}

func (uc *OrderUseCase) CancelOrder(id string) error {
	order, err := uc.repo.GetByID(id)
	if err != nil {
		return err
	}

	if order.Status == "Paid" {
		return errors.New("paid orders cannot be cancelled")
	}
	if order.Status != "Pending" {
		return errors.New("only pending orders can be cancelled")
	}

	return uc.repo.UpdateStatus(id, "Cancelled")
}
