package gateway

import (
	"context"
	"log"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentGateway struct {
	client servicepb.PaymentServiceClient
}

func NewPaymentGateway(paymentURL string) (*PaymentGateway, error) {
	conn, err := grpc.Dial(paymentURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &PaymentGateway{client: servicepb.NewPaymentServiceClient(conn)}, nil
}

func (g *PaymentGateway) ProcessPayment(orderID string, amount int64) (string, error) {
	req := &basepb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	}
	resp, err := g.client.ProcessPayment(context.Background(), req)
	if err != nil {
		log.Printf("Payment gRPC Error: %v", err)
		return "Failed", err
	}
	return resp.Status, nil
}
