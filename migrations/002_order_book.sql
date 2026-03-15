-- migrations/002_order_book.sql
-- Adds order book support: partial fills, order types, and bilateral trade records.

-- Track how many shares have been filled so far
ALTER TABLE orders ADD COLUMN filled_quantity INT NOT NULL DEFAULT 0;

-- Order type (limit only for now, extensible to market/stop later)
ALTER TABLE orders ADD COLUMN order_type VARCHAR(10) NOT NULL DEFAULT 'limit';

-- Index for rebuilding the order book on processor restart
CREATE INDEX idx_orders_open ON orders(symbol, side, price, created_at)
    WHERE status IN ('open', 'partial');

-- Bilateral trade records (each trade produces two execution rows, one per side)
CREATE TABLE trades (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol         VARCHAR(10) NOT NULL,
    buy_order_id   UUID NOT NULL REFERENCES orders(id),
    sell_order_id  UUID NOT NULL REFERENCES orders(id),
    quantity       INT NOT NULL,
    price          DECIMAL(15,4) NOT NULL,
    executed_at    TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_trades_symbol ON trades(symbol, executed_at DESC);
