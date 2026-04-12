package usecase

import (
	"errors"
	"order-service/internal/domain"
	"time"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderUseCase struct {
	repo       domain.OrderRepository
	gateway    domain.PaymentGateway
	updateChan chan *basepb.OrderStatusUpdate // Канал для стриминга статусов
}

func NewOrderUseCase(repo domain.OrderRepository, gateway domain.PaymentGateway, ch chan *basepb.OrderStatusUpdate) *OrderUseCase {
	return &OrderUseCase{repo: repo, gateway: gateway, updateChan: ch}
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
	if paymentStatus == "Failed" || paymentStatus == "Declined" {
		newStatus = "Failed"
	}

	uc.repo.UpdateStatus(order.ID, newStatus)
	order.Status = newStatus

	if uc.updateChan != nil {
		uc.updateChan <- &basepb.OrderStatusUpdate{
			OrderId:   order.ID,
			Status:    newStatus,
			UpdatedAt: timestamppb.Now(),
		}
	}

	return order, nil
}

func (uc *OrderUseCase) GetOrder(id string) (*domain.Order, error) {
	return uc.repo.GetByID(id)
}

func (uc *OrderUseCase) CancelOrder(id string) error {
	return uc.repo.UpdateStatus(id, "Cancelled")
}
