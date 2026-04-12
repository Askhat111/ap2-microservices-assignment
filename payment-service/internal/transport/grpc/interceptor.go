package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("[gRPC] Method: %s | Duration: %v | Err: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}
