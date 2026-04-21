package grpc

import (
	"sync"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
)

type OrderBroadcaster struct {
	mu       sync.Mutex
	channels map[chan *basepb.OrderStatusUpdate]struct{}
}

func NewOrderBroadcaster() *OrderBroadcaster {
	return &OrderBroadcaster{
		channels: make(map[chan *basepb.OrderStatusUpdate]struct{}),
	}
}

func (b *OrderBroadcaster) Broadcast(update *basepb.OrderStatusUpdate) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.channels {
		select {
		case ch <- update:
		default:
		}
	}
}

func (b *OrderBroadcaster) Subscribe() chan *basepb.OrderStatusUpdate {
	ch := make(chan *basepb.OrderStatusUpdate, 10)
	b.mu.Lock()
	b.channels[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *OrderBroadcaster) Unsubscribe(ch chan *basepb.OrderStatusUpdate) {
	b.mu.Lock()
	delete(b.channels, ch)
	close(ch)
	b.mu.Unlock()
}
