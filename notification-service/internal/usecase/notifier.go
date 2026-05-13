package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/redis/go-redis/v9"

	"notification-service/internal/domain"
	"notification-service/internal/provider"
)

const (
	notifierMaxRetries = 3
	idempotencyTTL     = 24 * time.Hour
)

type Notifier struct {
	sender      provider.EmailSender
	redisClient *redis.Client
}

func NewNotifier(sender provider.EmailSender, redisClient *redis.Client) *Notifier {
	return &Notifier{sender: sender, redisClient: redisClient}
}

func notificationKey(eventID string) string {
	return fmt.Sprintf("notification:processed:%s", eventID)
}

func exponentialBackoff(n int) time.Duration {
	return time.Duration(math.Pow(2, float64(n))) * time.Second
}

func (n *Notifier) Handle(ctx context.Context, event domain.PaymentEvent) {
	key := notificationKey(event.EventID)

	if status, err := n.redisClient.Get(ctx, key).Result(); !errors.Is(err, redis.Nil) {
		log.Printf("[Notifier] Duplicate event %s skipped (status: %s)", event.EventID, status)
		return
	}

	n.redisClient.Set(ctx, key, "processing", idempotencyTTL)

	req := provider.EmailRequest{
		To:      event.CustomerEmail,
		Subject: fmt.Sprintf("Your order #%s", event.OrderID),
		Body: fmt.Sprintf(
			"Your payment for order #%s has been processed.\nAmount: $%.2f\nStatus: %s",
			event.OrderID, float64(event.Amount)/100.0, event.Status,
		),
	}

	var lastErr error
	for attempt := 0; attempt <= notifierMaxRetries; attempt++ {
		if attempt > 0 {
			delay := exponentialBackoff(attempt)
			log.Printf("[Notifier] Retry %d/%d for event %s, waiting %v...",
				attempt, notifierMaxRetries, event.EventID, delay)
			select {
			case <-ctx.Done():
				n.redisClient.Set(ctx, key, "failed", idempotencyTTL)
				return
			case <-time.After(delay):
			}
		}

		lastErr = n.sender.Send(ctx, req)
		if lastErr == nil {
			n.redisClient.Set(ctx, key, "sent", idempotencyTTL)
			log.Printf("[Notifier] Event %s sent successfully", event.EventID)
			return
		}

		log.Printf("[Notifier] Attempt %d failed for event %s: %v", attempt+1, event.EventID, lastErr)
	}

	n.redisClient.Set(ctx, key, "failed", idempotencyTTL)
	log.Printf("[Notifier] Event %s permanently failed after %d retries: %v",
		event.EventID, notifierMaxRetries, lastErr)
}
