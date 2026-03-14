package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/benniu/tradeengine/internal/models"
)

func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	cfg.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

func GetUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*models.User, error) {
	var u models.User
	err := pool.QueryRow(ctx,
		`SELECT id, username, balance, created_at FROM users WHERE id = $1`, userID,
	).Scan(&u.ID, &u.Username, &u.Balance, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func GetOrder(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) (*models.Order, error) {
	var o models.Order
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, symbol, side, quantity, price, status, idempotency_key,
		        created_at, executed_at, settled_at
		 FROM orders WHERE id = $1`, orderID,
	).Scan(&o.ID, &o.UserID, &o.Symbol, &o.Side, &o.Quantity, &o.Price,
		&o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.ExecutedAt, &o.SettledAt)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return &o, nil
}

func GetOrderByIdempotencyKey(ctx context.Context, pool *pgxpool.Pool, key string) (*models.Order, error) {
	var o models.Order
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, symbol, side, quantity, price, status, idempotency_key,
		        created_at, executed_at, settled_at
		 FROM orders WHERE idempotency_key = $1`, key,
	).Scan(&o.ID, &o.UserID, &o.Symbol, &o.Side, &o.Quantity, &o.Price,
		&o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.ExecutedAt, &o.SettledAt)
	if err != nil {
		return nil, fmt.Errorf("get order by idempotency key: %w", err)
	}
	return &o, nil
}

func GetOrdersByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]models.Order, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, user_id, symbol, side, quantity, price, status, idempotency_key,
		        created_at, executed_at, settled_at
		 FROM orders WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("get orders by user: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Symbol, &o.Side, &o.Quantity, &o.Price,
			&o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.ExecutedAt, &o.SettledAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func CreateOrder(ctx context.Context, pool *pgxpool.Pool, req models.CreateOrderRequest) (*models.Order, error) {
	var o models.Order
	err := pool.QueryRow(ctx,
		`INSERT INTO orders (user_id, symbol, side, quantity, price, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, symbol, side, quantity, price, status, idempotency_key,
		           created_at, executed_at, settled_at`,
		req.UserID, req.Symbol, req.Side, req.Quantity, req.Price, nilIfEmpty(req.IdempotencyKey),
	).Scan(&o.ID, &o.UserID, &o.Symbol, &o.Side, &o.Quantity, &o.Price,
		&o.Status, &o.IdempotencyKey, &o.CreatedAt, &o.ExecutedAt, &o.SettledAt)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	return &o, nil
}

func UpdateOrderStatus(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID, status models.OrderStatus) error {
	_, err := pool.Exec(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2`, status, orderID)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return nil
}

func GetPositionsByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]models.Position, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, user_id, symbol, quantity, avg_cost, updated_at
		 FROM positions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}
	defer rows.Close()

	var positions []models.Position
	for rows.Next() {
		var p models.Position
		if err := rows.Scan(&p.ID, &p.UserID, &p.Symbol, &p.Quantity, &p.AvgCost, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

func GetPosition(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, symbol string) (*models.Position, error) {
	var p models.Position
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, symbol, quantity, avg_cost, updated_at
		 FROM positions WHERE user_id = $1 AND symbol = $2`, userID, symbol,
	).Scan(&p.ID, &p.UserID, &p.Symbol, &p.Quantity, &p.AvgCost, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get position: %w", err)
	}
	return &p, nil
}

// ExecuteOrder runs the full order execution inside a single transaction.
// For buys: checks balance, debits user, upserts position, creates execution.
// For sells: checks position, decrements shares, credits user, creates execution.
// Returns a reason string if the order is rejected (empty on success).
func ExecuteOrder(ctx context.Context, pool *pgxpool.Pool, order models.Order) (string, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	total := order.Price.Mul(decimal.NewFromInt(int64(order.Quantity)))
	now := time.Now()

	switch order.Side {
	case models.Buy:
		// Lock user row and check balance
		var balance decimal.Decimal
		err := tx.QueryRow(ctx,
			`SELECT balance FROM users WHERE id = $1 FOR UPDATE`, order.UserID,
		).Scan(&balance)
		if err != nil {
			return "", fmt.Errorf("lock user for buy: %w", err)
		}
		if balance.LessThan(total) {
			// Reject: insufficient funds
			if _, err := tx.Exec(ctx,
				`UPDATE orders SET status = 'rejected' WHERE id = $1`, order.ID); err != nil {
				return "", fmt.Errorf("reject order: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return "", fmt.Errorf("commit rejection: %w", err)
			}
			return fmt.Sprintf("insufficient funds: need %s, have %s", total, balance), nil
		}

		// Debit balance
		if _, err := tx.Exec(ctx,
			`UPDATE users SET balance = balance - $1 WHERE id = $2`, total, order.UserID); err != nil {
			return "", fmt.Errorf("debit balance: %w", err)
		}

		// Upsert position with weighted average cost
		if _, err := tx.Exec(ctx,
			`INSERT INTO positions (user_id, symbol, quantity, avg_cost, updated_at)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (user_id, symbol) DO UPDATE SET
			   avg_cost = (positions.avg_cost * positions.quantity + $4 * $3) / (positions.quantity + $3),
			   quantity = positions.quantity + $3,
			   updated_at = $5`,
			order.UserID, order.Symbol, order.Quantity, order.Price, now); err != nil {
			return "", fmt.Errorf("upsert position: %w", err)
		}

	case models.Sell:
		// Lock position row and check quantity
		var posQty int
		err := tx.QueryRow(ctx,
			`SELECT quantity FROM positions WHERE user_id = $1 AND symbol = $2 FOR UPDATE`,
			order.UserID, order.Symbol,
		).Scan(&posQty)
		if err != nil {
			if err == pgx.ErrNoRows {
				// No position at all
				if _, err := tx.Exec(ctx,
					`UPDATE orders SET status = 'rejected' WHERE id = $1`, order.ID); err != nil {
					return "", fmt.Errorf("reject order: %w", err)
				}
				if err := tx.Commit(ctx); err != nil {
					return "", fmt.Errorf("commit rejection: %w", err)
				}
				return fmt.Sprintf("no position in %s", order.Symbol), nil
			}
			return "", fmt.Errorf("lock position for sell: %w", err)
		}
		if posQty < order.Quantity {
			if _, err := tx.Exec(ctx,
				`UPDATE orders SET status = 'rejected' WHERE id = $1`, order.ID); err != nil {
				return "", fmt.Errorf("reject order: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return "", fmt.Errorf("commit rejection: %w", err)
			}
			return fmt.Sprintf("insufficient shares: need %d, have %d", order.Quantity, posQty), nil
		}

		newQty := posQty - order.Quantity
		if newQty == 0 {
			if _, err := tx.Exec(ctx,
				`DELETE FROM positions WHERE user_id = $1 AND symbol = $2`,
				order.UserID, order.Symbol); err != nil {
				return "", fmt.Errorf("delete position: %w", err)
			}
		} else {
			if _, err := tx.Exec(ctx,
				`UPDATE positions SET quantity = $1, updated_at = $2 WHERE user_id = $3 AND symbol = $4`,
				newQty, now, order.UserID, order.Symbol); err != nil {
				return "", fmt.Errorf("update position: %w", err)
			}
		}

		// Credit balance
		if _, err := tx.Exec(ctx,
			`UPDATE users SET balance = balance + $1 WHERE id = $2`, total, order.UserID); err != nil {
			return "", fmt.Errorf("credit balance: %w", err)
		}

	default:
		return "", fmt.Errorf("unknown order side: %s", order.Side)
	}

	// Insert execution record
	if _, err := tx.Exec(ctx,
		`INSERT INTO executions (order_id, user_id, symbol, side, quantity, price, total, executed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.ID, order.UserID, order.Symbol, order.Side, order.Quantity, order.Price, total, now); err != nil {
		return "", fmt.Errorf("insert execution: %w", err)
	}

	// Mark order as executed
	if _, err := tx.Exec(ctx,
		`UPDATE orders SET status = 'executed', executed_at = $1 WHERE id = $2`, now, order.ID); err != nil {
		return "", fmt.Errorf("update order executed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit execution: %w", err)
	}
	return "", nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
