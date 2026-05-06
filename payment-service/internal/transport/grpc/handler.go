package grpc

import (
	"context"
	"payment-service/internal/usecase"

	basepb "github.com/Askhat111/converted-proto/base/frontend/v1"
	servicepb "github.com/Askhat111/converted-proto/service/frontend/client/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentHandler struct {
	servicepb.UnimplementedPaymentServiceServer
	useCase *usecase.PaymentUseCase
}

func NewPaymentHandler(uc *usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{useCase: uc}
}

func (h *PaymentHandler) ProcessPayment(ctx context.Context, req *basepb.PaymentRequest) (*basepb.PaymentResponse, error) {
	payment, err := h.useCase.Process(req.OrderId, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "DB error: %v", err)
	}

	return &basepb.PaymentResponse{
		TransactionId: payment.TransactionID,
		Status:        payment.Status,
	}, nil
}

func (h *PaymentHandler) ListPayments(ctx context.Context, req *basepb.ListPaymentsRequest) (*basepb.ListPaymentsResponse, error) {
	payments, err := h.useCase.GetPaymentsByStatus(req.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list payments: %v", err)
	}

	var pbPayments []*basepb.PaymentResponse
	for _, p := range payments {
		pbPayments = append(pbPayments, &basepb.PaymentResponse{
			TransactionId: p.TransactionID,
			Status:        p.Status,
		})
	}

	return &basepb.ListPaymentsResponse{
		Payments: pbPayments,
	}, nil
}
