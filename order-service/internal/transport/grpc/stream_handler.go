package grpc

import (
	"log"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"
)

type OrderStreamHandler struct {
	servicepb.UnimplementedOrderTrackingServiceServer
	statusChan chan *basepb.OrderStatusUpdate
}

func NewOrderStreamHandler(ch chan *basepb.OrderStatusUpdate) *OrderStreamHandler {
	return &OrderStreamHandler{statusChan: ch}
}

func (h *OrderStreamHandler) SubscribeToOrderUpdates(req *basepb.OrderRequest, stream servicepb.OrderTrackingService_SubscribeToOrderUpdatesServer) error {
	log.Printf("Client subcscribed to updating order: %s", req.GetOrderId())

	for update := range h.statusChan {
		if update.OrderId == req.GetOrderId() {
			if err := stream.Send(update); err != nil {
				log.Printf("Stream Error: %v", err)
				return err
			}
		}
	}
	return nil
}
