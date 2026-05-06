package usecase

import (
	"fmt"
	"log"
	"notification-service/internal/domain"
	"sync"
)

type Notifier struct {
	mu           sync.Mutex
	processedIDs map[string]bool
}

func NewNotifier() *Notifier {
	return &Notifier{
		processedIDs: make(map[string]bool),
	}
}

func (n *Notifier) Handle(event domain.PaymentEvent) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.processedIDs[event.EventID] {
		log.Printf("[Notification] Duplicate event %s skipped (order: %s)", event.EventID, event.OrderID)
		return false
	}

	n.processedIDs[event.EventID] = true

	amountInDollars := float64(event.Amount) / 100.0
	fmt.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f. Status: %s\n",
		event.CustomerEmail, event.OrderID, amountInDollars, event.Status)

	return true
}
