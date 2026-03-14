package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderSide string

const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusValidated OrderStatus = "validated"
	StatusExecuted  OrderStatus = "executed"
	StatusSettled   OrderStatus = "settled"
	StatusRejected  OrderStatus = "rejected"
)

type User struct {
	ID        uuid.UUID       `json:"id"`
	Username  string          `json:"username"`
	Balance   decimal.Decimal `json:"balance"`
	CreatedAt time.Time       `json:"created_at"`
}

type Order struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	Symbol         string          `json:"symbol"`
	Side           OrderSide       `json:"side"`
	Quantity       int             `json:"quantity"`
	Price          decimal.Decimal `json:"price"`
	Status         OrderStatus     `json:"status"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	ExecutedAt     *time.Time      `json:"executed_at,omitempty"`
	SettledAt      *time.Time      `json:"settled_at,omitempty"`
}

type Position struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Symbol    string          `json:"symbol"`
	Quantity  int             `json:"quantity"`
	AvgCost   decimal.Decimal `json:"avg_cost"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Execution struct {
	ID         uuid.UUID       `json:"id"`
	OrderID    uuid.UUID       `json:"order_id"`
	UserID     uuid.UUID       `json:"user_id"`
	Symbol     string          `json:"symbol"`
	Side       OrderSide       `json:"side"`
	Quantity   int             `json:"quantity"`
	Price      decimal.Decimal `json:"price"`
	Total      decimal.Decimal `json:"total"`
	ExecutedAt time.Time       `json:"executed_at"`
}

// CreateOrderRequest is the JSON body for POST /orders.
type CreateOrderRequest struct {
	UserID         uuid.UUID       `json:"user_id"`
	Symbol         string          `json:"symbol"`
	Side           OrderSide       `json:"side"`
	Quantity       int             `json:"quantity"`
	Price          decimal.Decimal `json:"price"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

// OrderEvent is the Kafka message envelope for order processing.
type OrderEvent struct {
	Order  Order  `json:"order"`
	Reason string `json:"reason,omitempty"`
}
