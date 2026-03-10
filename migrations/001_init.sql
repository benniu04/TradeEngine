-- migrations/001_init.sql

-- Users with cash balances
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username    VARCHAR(100) UNIQUE NOT NULL,
    balance     DECIMAL(15,2) NOT NULL DEFAULT 10000.00,  -- start with $10k
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- Every order submitted
CREATE TABLE orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    symbol          VARCHAR(10) NOT NULL,        -- e.g. "AAPL"
    side            VARCHAR(4) NOT NULL,          -- "buy" or "sell"
    quantity         INT NOT NULL,
    price           DECIMAL(15,4) NOT NULL,       -- limit price
    status          VARCHAR(20) DEFAULT 'pending',
    -- pending -> validated -> executed -> settled
    -- pending -> rejected
    idempotency_key VARCHAR(255) UNIQUE,          -- prevent double-submit
    created_at      TIMESTAMPTZ DEFAULT now(),
    executed_at     TIMESTAMPTZ,
    settled_at      TIMESTAMPTZ
);

CREATE INDEX idx_orders_user ON orders(user_id, created_at DESC);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_idempotency ON orders(idempotency_key);

-- Current holdings per user
CREATE TABLE positions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    symbol      VARCHAR(10) NOT NULL,
    quantity    INT NOT NULL DEFAULT 0,
    avg_cost    DECIMAL(15,4) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, symbol)
);

CREATE INDEX idx_positions_user ON positions(user_id);

-- Executed trades (immutable ledger)
CREATE TABLE executions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL REFERENCES orders(id),
    user_id     UUID NOT NULL REFERENCES users(id),
    symbol      VARCHAR(10) NOT NULL,
    side        VARCHAR(4) NOT NULL,
    quantity    INT NOT NULL,
    price       DECIMAL(15,4) NOT NULL,
    total       DECIMAL(15,2) NOT NULL,           -- price * quantity
    executed_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_executions_user ON executions(user_id, executed_at DESC);

-- Seed some users
INSERT INTO users (id, username, balance) VALUES
    ('11111111-1111-1111-1111-111111111111', 'alice', 10000.00),
    ('22222222-2222-2222-2222-222222222222', 'bob', 25000.00);