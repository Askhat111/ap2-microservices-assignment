package grpc

import (
	"log"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"
)

type OrderStreamHandler struct {
	servicepb.UnimplementedOrderTrackingServiceServer
	broadcaster *OrderBroadcaster
}

func NewOrderStreamHandler(b *OrderBroadcaster) *OrderStreamHandler {
	return &OrderStreamHandler{broadcaster: b}
}

func (h *OrderStreamHandler) SubscribeToOrderUpdates(req *basepb.OrderRequest, stream servicepb.OrderTrackingService_SubscribeToOrderUpdatesServer) error {
	log.Println("[Stream] Postman connected to gRPC Stream!")
	ch := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(ch)

	for {
		select {
		case <-stream.Context().Done():
			log.Println("[Stream] Postman disconnected.")
			return nil
		case update := <-ch:
			if err := stream.Send(update); err != nil {
				log.Printf("Stream Send Error: %v", err)
				return err
			}
			log.Printf("[Stream] Successfully pushed Order %s to Postman!", update.OrderId)
		}
	}
}
