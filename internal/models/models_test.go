package models

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestOrderEventSerializationRoundTrip(t *testing.T) {
	event := OrderEvent{
		Order: Order{
			ID:       uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			UserID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Symbol:   "AAPL",
			Side:     Buy,
			Quantity: 10,
			Price:    decimal.NewFromFloat(150.50),
			Status:   StatusPending,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded OrderEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Order.ID != event.Order.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.Order.ID, event.Order.ID)
	}
	if decoded.Order.Symbol != "AAPL" {
		t.Errorf("symbol mismatch: got %s, want AAPL", decoded.Order.Symbol)
	}
	if decoded.Order.Side != Buy {
		t.Errorf("side mismatch: got %s, want buy", decoded.Order.Side)
	}
	if !decoded.Order.Price.Equal(event.Order.Price) {
		t.Errorf("price mismatch: got %s, want %s", decoded.Order.Price, event.Order.Price)
	}
	if decoded.Reason != "" {
		t.Errorf("reason should be empty, got %s", decoded.Reason)
	}
}

func TestOrderEventWithRejectionReason(t *testing.T) {
	event := OrderEvent{
		Order: Order{
			ID:     uuid.New(),
			Status: StatusRejected,
		},
		Reason: "insufficient funds: need 1500.00, have 1000.00",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded OrderEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Reason != event.Reason {
		t.Errorf("reason mismatch: got %q, want %q", decoded.Reason, event.Reason)
	}
	if decoded.Order.Status != StatusRejected {
		t.Errorf("status mismatch: got %s, want rejected", decoded.Order.Status)
	}
}

func TestCreateOrderRequestValidation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, req CreateOrderRequest)
	}{
		{
			name:  "valid buy order",
			input: `{"user_id":"11111111-1111-1111-1111-111111111111","symbol":"AAPL","side":"buy","quantity":10,"price":"150.50"}`,
			check: func(t *testing.T, req CreateOrderRequest) {
				if req.Side != Buy {
					t.Errorf("expected buy, got %s", req.Side)
				}
				if req.Quantity != 10 {
					t.Errorf("expected 10, got %d", req.Quantity)
				}
			},
		},
		{
			name:  "valid sell order",
			input: `{"user_id":"11111111-1111-1111-1111-111111111111","symbol":"TSLA","side":"sell","quantity":5,"price":"200.00"}`,
			check: func(t *testing.T, req CreateOrderRequest) {
				if req.Side != Sell {
					t.Errorf("expected sell, got %s", req.Side)
				}
				if req.Symbol != "TSLA" {
					t.Errorf("expected TSLA, got %s", req.Symbol)
				}
			},
		},
		{
			name:  "with idempotency key",
			input: `{"user_id":"11111111-1111-1111-1111-111111111111","symbol":"GOOG","side":"buy","quantity":1,"price":"100","idempotency_key":"abc-123"}`,
			check: func(t *testing.T, req CreateOrderRequest) {
				if req.IdempotencyKey != "abc-123" {
					t.Errorf("expected abc-123, got %s", req.IdempotencyKey)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req CreateOrderRequest
			if err := json.Unmarshal([]byte(tc.input), &req); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tc.check(t, req)
		})
	}
}
