package provider

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type SimulatedProvider struct {
	failureRate float64
	latency     time.Duration
}

func NewSimulatedProvider(failureRate float64, latency time.Duration) *SimulatedProvider {
	return &SimulatedProvider{failureRate: failureRate, latency: latency}
}

func (p *SimulatedProvider) Send(ctx context.Context, req EmailRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.latency):
	}

	if rand.Float64() < p.failureRate {
		return errors.New("provider: transient error — connection timeout")
	}

	fmt.Printf("[Email] To: %s | Subject: %s\n%s\n", req.To, req.Subject, req.Body)
	return nil
}
