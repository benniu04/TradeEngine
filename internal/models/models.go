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
	StatusOpen      OrderStatus = "open"    // resting in order book, no fills yet
	StatusPartial   OrderStatus = "partial" // partially filled, still in book
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
	FilledQuantity int             `json:"filled_quantity"`
	Price          decimal.Decimal `json:"price"`
	Status         OrderStatus     `json:"status"`
	OrderType      string          `json:"order_type"`
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

// BookEntry represents a resting order in the matching engine's order book.
type BookEntry struct {
	OrderID      uuid.UUID
	UserID       uuid.UUID
	Price        decimal.Decimal
	RemainingQty int
	CreatedAt    time.Time
}

// Match represents a successful match between a buy and sell order.
type Match struct {
	BuyOrderID  uuid.UUID
	BuyUserID   uuid.UUID
	SellOrderID uuid.UUID
	SellUserID  uuid.UUID
	Quantity    int
	Price       decimal.Decimal // resting order's price
}

// Trade is a bilateral trade record persisted to the database.
type Trade struct {
	ID          uuid.UUID       `json:"id"`
	Symbol      string          `json:"symbol"`
	BuyOrderID  uuid.UUID       `json:"buy_order_id"`
	SellOrderID uuid.UUID       `json:"sell_order_id"`
	Quantity    int             `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	ExecutedAt  time.Time       `json:"executed_at"`
}

// OrderEvent is the Kafka message envelope for order processing.
type OrderEvent struct {
	Order  Order   `json:"order"`
	Trades []Trade `json:"trades,omitempty"`
	Reason string  `json:"reason,omitempty"`
}
