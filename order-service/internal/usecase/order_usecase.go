package usecase

import (
	"errors"
	"order-service/internal/domain"
	grpctransport "order-service/internal/transport/grpc"
	"time"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderUseCase struct {
	repo        domain.OrderRepository
	gateway     domain.PaymentGateway
	broadcaster *grpctransport.OrderBroadcaster
}

func NewOrderUseCase(repo domain.OrderRepository, gateway domain.PaymentGateway, b *grpctransport.OrderBroadcaster) *OrderUseCase {
	return &OrderUseCase{repo: repo, gateway: gateway, broadcaster: b}
}

func (uc *OrderUseCase) CreateOrder(customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	if idempotencyKey != "" {
		existingOrder, err := uc.repo.GetByIdempotencyKey(idempotencyKey)
		if err == nil && existingOrder != nil {
			return existingOrder, nil
		}
	}

	order := &domain.Order{
		ID:             uuid.NewString(),
		CustomerID:     customerID,
		ItemName:       itemName,
		Amount:         amount,
		Status:         "Pending",
		CreatedAt:      time.Now(),
		IdempotencyKey: idempotencyKey,
	}

	if err := uc.repo.Create(order); err != nil {
		return nil, err
	}

	paymentStatus, err := uc.gateway.ProcessPayment(order.ID, order.Amount)
	newStatus := "Paid"
	if err != nil || paymentStatus == "Failed" || paymentStatus == "Declined" {
		newStatus = "Failed"
	}

	uc.repo.UpdateStatus(order.ID, newStatus)
	order.Status = newStatus

	uc.broadcaster.Broadcast(&basepb.OrderStatusUpdate{
		OrderId:   order.ID,
		Status:    newStatus,
		UpdatedAt: timestamppb.Now(),
	})

	return order, nil
}

func (uc *OrderUseCase) GetOrder(id string) (*domain.Order, error) {
	return uc.repo.GetByID(id)
}

func (uc *OrderUseCase) CancelOrder(id string) error {
	return uc.repo.UpdateStatus(id, "Cancelled")
}
