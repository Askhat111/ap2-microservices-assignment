package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type HTTPPaymentGateway struct {
	client  *http.Client
	baseURL string
}

func NewHTTPPaymentGateway(baseURL string) *HTTPPaymentGateway {
	return &HTTPPaymentGateway{
		client:  &http.Client{Timeout: 2 * time.Second},
		baseURL: baseURL,
	}
}

func (g *HTTPPaymentGateway) ProcessPayment(orderID string, amount int64) (string, error) {
	payload := map[string]interface{}{
		"order_id": orderID,
		"amount":   amount,
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := g.client.Post(g.baseURL+"/payments", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("payment service error")
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	return result["status"], nil
}
