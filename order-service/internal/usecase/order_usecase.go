package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"order-service/internal/cache"
	"order-service/internal/domain"
	grpctransport "order-service/internal/transport/grpc"
)

const orderCacheTTL = 5 * time.Minute

type OrderUseCase struct {
	repo        domain.OrderRepository
	gateway     domain.PaymentGateway
	broadcaster *grpctransport.OrderBroadcaster
	cache       cache.OrderCache
}

func NewOrderUseCase(
	repo domain.OrderRepository,
	gateway domain.PaymentGateway,
	broadcaster *grpctransport.OrderBroadcaster,
	cache cache.OrderCache,
) *OrderUseCase {
	return &OrderUseCase{
		repo:        repo,
		gateway:     gateway,
		broadcaster: broadcaster,
		cache:       cache,
	}
}

func orderCacheKey(id string) string {
	return fmt.Sprintf("order:%s", id)
}

func (uc *OrderUseCase) CreateOrder(customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}

	if idempotencyKey != "" {
		if existing, err := uc.repo.GetByIdempotencyKey(idempotencyKey); err == nil && existing != nil {
			return existing, nil
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
	if err != nil || paymentStatus != "Authorized" {
		newStatus = "Failed"
	}

	uc.repo.UpdateStatus(order.ID, newStatus)
	order.Status = newStatus

	ctx := context.Background()

	uc.cache.Delete(ctx, orderCacheKey(order.ID))

	if err := uc.cache.Set(ctx, orderCacheKey(order.ID), order, orderCacheTTL); err != nil {
		log.Printf("[Cache] Failed to cache order %s: %v", order.ID, err)
	}

	uc.broadcaster.Broadcast(&basepb.OrderStatusUpdate{
		OrderId:   order.ID,
		Status:    order.Status,
		UpdatedAt: timestamppb.Now(),
	})

	return order, nil
}

func (uc *OrderUseCase) GetOrder(id string) (*domain.Order, error) {
	ctx := context.Background()
	var order domain.Order

	err := uc.cache.Get(ctx, orderCacheKey(id), &order)
	if err == nil {
		log.Printf("[Cache] HIT  order:%s", id)
		return &order, nil
	}

	if errors.Is(err, cache.ErrCacheMiss) {
		log.Printf("[Cache] MISS order:%s — querying DB", id)
	} else {
		log.Printf("[Cache] Redis error for order %s: %v", id, err)
	}

	dbOrder, err := uc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if err := uc.cache.Set(ctx, orderCacheKey(id), dbOrder, orderCacheTTL); err != nil {
		log.Printf("[Cache] Failed to cache order %s: %v", id, err)
	}

	return dbOrder, nil
}

func (uc *OrderUseCase) CancelOrder(id string) error {
	if err := uc.repo.UpdateStatus(id, "Cancelled"); err != nil {
		return err
	}

	ctx := context.Background()
	if err := uc.cache.Delete(ctx, orderCacheKey(id)); err != nil {
		log.Printf("[Cache] Failed to invalidate cancelled order %s: %v", id, err)
	}

	return nil
}
