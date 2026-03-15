# TradeEngine

A real-time stock trading platform built with Go microservices, Kafka event streaming, and a React dashboard. Features an in-memory order matching engine with price-time priority, WebSocket live updates, and Finnhub market data integration.

## Architecture

```
                         ┌─────────────┐
                         │  Dashboard   │
                         │  React/TS    │
                         └──────┬───────┘
                                │ HTTP + WebSocket
                                ▼
┌──────────────────────────────────────────────────────┐
│                    Gateway (:8080)                     │
│  REST API · WebSocket Hub · Redis Cache · Market Data │
└──────────┬──────────────────────────────┬─────────────┘
           │ Kafka: "orders"              │ Kafka: "executions"
           ▼                              │
┌─────────────────────────┐               │
│       Processor          │──────────────┘
│  Matching Engine · Trade │
│  Settlement · Validation │
└──────────┬───────────────┘
           │
     ┌─────┴─────┐
     ▼           ▼
┌─────────┐ ┌─────────┐
│ Postgres │ │  Redis  │
│  (5433)  │ │ (6379)  │
└─────────┘ └─────────┘
```

**Order lifecycle:** Client submits order via REST (returns 202) → Gateway publishes to Kafka → Processor validates, runs matching engine, settles trades atomically in Postgres → publishes result to Kafka → Gateway broadcasts to connected WebSocket clients in real-time.

## Key Features

- **Order Matching Engine** — In-memory price-time priority order book with partial fill support. Bids sorted price DESC/time ASC, asks sorted price ASC/time ASC. Executes at the resting order's price.
- **Atomic Trade Settlement** — Single Postgres transaction per trade: debits buyer, credits seller, upserts positions with weighted average cost, inserts bilateral trade records. Deadlock prevention via UUID-sorted row locking (`SELECT ... FOR UPDATE`).
- **Async Pipeline** — Orders are accepted immediately (202) and processed asynchronously through Kafka. Decouples API latency from matching/settlement.
- **Idempotent Orders** — Deduplication via Redis `SETNX` + Postgres `UNIQUE` constraint on idempotency key. Safe to retry without double-execution.
- **WebSocket Streaming** — Real-time order status and trade notifications pushed to connected clients via a Kafka → WebSocket bridge.
- **Live Market Data** — Finnhub API integration with Redis-cached quotes (15s TTL) to stay within rate limits. API key kept server-side.
- **Redis Caching** — Read-through cache for user balances (30s), positions (30s), and market quotes (15s) with invalidation on writes.
- **Order Book Rebuild** — Processor rebuilds its in-memory order book from the database on restart, loading all `open` and `partial` orders.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| API Gateway | Go, chi/v5, nhooyr.io/websocket |
| Message Broker | Apache Kafka (segmentio/kafka-go) |
| Database | PostgreSQL 15 (pgx/v5, connection pooling) |
| Cache | Redis 7 (go-redis/v9) |
| Matching Engine | Pure Go, in-memory sorted order books |
| Financial Math | shopspring/decimal (arbitrary precision) |
| Dashboard | React 19, TypeScript, Vite, Tailwind CSS 4 |
| Market Data | Finnhub REST API |
| Infrastructure | Docker Compose |

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 18+
- Docker & Docker Compose

### Setup

```bash
# Start infrastructure (Postgres, Redis, Kafka)
make docker-up

# Apply database migrations
psql "postgres://trade:trade@localhost:5433/tradeengine?sslmode=disable" \
  -f migrations/001_init.sql -f migrations/002_order_book.sql

# (Optional) Add a Finnhub API key for live market data
# Get a free key at https://finnhub.io — add it to .env
cp .env.example .env

# Start the processor (Terminal 1)
make run-processor

# Start the gateway (Terminal 2)
make run-gateway

# Start the dashboard (Terminal 3)
cd dashboard && npm install && npm run dev
```

The dashboard will be available at `http://localhost:3000`.

### Seed Users

| User | ID | Balance |
|------|----|---------|
| alice | `11111111-1111-1111-1111-111111111111` | $10,000 |
| bob | `22222222-2222-2222-2222-222222222222` | $25,000 |

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/orders` | Submit a new order (async, returns 202) |
| `GET` | `/orders/{id}` | Get order by ID |
| `GET` | `/orders?user_id=` | List orders for a user |
| `GET` | `/positions?user_id=` | List positions for a user |
| `GET` | `/users/{id}` | Get user profile and balance |
| `GET` | `/quote/{symbol}` | Live market quote (Finnhub) |
| `GET` | `/book/{symbol}` | Order book depth (aggregated) |
| `GET` | `/ws?user_id=` | WebSocket for real-time updates |

### Example: Place an Order

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "11111111-1111-1111-1111-111111111111",
    "symbol": "AAPL",
    "side": "buy",
    "quantity": 10,
    "price": "150.50",
    "idempotency_key": "my-unique-key"
  }'
```

### Example: Match Two Orders

```bash
# Alice buys 10 AAPL at $150
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"11111111-1111-1111-1111-111111111111","symbol":"AAPL","side":"buy","quantity":10,"price":"150.00"}'

# Bob sells 10 AAPL at $150 — triggers a match
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":"22222222-2222-2222-2222-222222222222","symbol":"AAPL","side":"sell","quantity":10,"price":"150.00"}'
```

## Testing

```bash
# Unit tests (matching engine)
make test-unit

# Integration tests (requires Docker infrastructure running)
make test-integration

# All tests with race detector
make test
```

The integration tests cover: order happy paths, insufficient funds/shares rejection, idempotency deduplication, concurrent order processing, concurrent overdraft prevention, and bilateral trade settlement.

## Project Structure

```
cmd/
  gateway/       REST API server, WebSocket hub, market data proxy
  processor/     Kafka consumer, matching engine, trade settlement
internal/
  cache/         Redis caching layer (balances, positions, idempotency)
  config/        Environment configuration
  db/            PostgreSQL queries and transactions
  kafka/         Kafka producer and consumer
  market/        Finnhub API client with Redis caching
  matching/      In-memory order book matching engine
  models/        Domain types (orders, trades, positions, quotes)
  ws/            WebSocket connection manager and Kafka bridge
dashboard/       React + TypeScript + Tailwind CSS frontend
migrations/      PostgreSQL schema migrations
```
