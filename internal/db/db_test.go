//go:build integration

package db

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/benniu/tradeengine/internal/config"
	"github.com/benniu/tradeengine/internal/models"
)

var (
	aliceID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	bobID   = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

func testPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	cfg := config.Load()
	p, err := NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { p.Close() })

	// Clean tables in FK order
	for _, table := range []string{"executions", "trades", "positions", "orders"} {
		if _, err := p.Exec(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
	// Reset seed users
	p.Exec(ctx, `UPDATE users SET balance = 10000.00 WHERE id = $1`, aliceID)
	p.Exec(ctx, `UPDATE users SET balance = 25000.00 WHERE id = $1`, bobID)

	return ctx, p
}

func TestBuyOrderHappyPath(t *testing.T) {
	ctx, p := testPool(t)

	// Seed bob with shares
	p.Exec(ctx, `INSERT INTO positions (user_id, symbol, quantity, avg_cost) VALUES ($1, 'AAPL', 100, 150.0000)`, bobID)

	sellOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: bobID, Symbol: "AAPL", Side: models.Sell, Quantity: 10,
		Price: decimal.NewFromFloat(150),
	})
	buyOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: aliceID, Symbol: "AAPL", Side: models.Buy, Quantity: 10,
		Price: decimal.NewFromFloat(150),
	})

	match := models.Match{
		BuyOrderID: buyOrder.ID, BuyUserID: aliceID,
		SellOrderID: sellOrder.ID, SellUserID: bobID,
		Quantity: 10, Price: decimal.NewFromFloat(150),
	}
	if err := ExecuteTrade(ctx, p, match, "AAPL"); err != nil {
		t.Fatalf("execute trade: %v", err)
	}

	// Alice: 10000 - 1500 = 8500
	alice, _ := GetUser(ctx, p, aliceID)
	if !alice.Balance.Equal(decimal.NewFromFloat(8500)) {
		t.Errorf("alice balance: want 8500, got %s", alice.Balance)
	}

	// Alice has 10 AAPL
	pos, _ := GetPosition(ctx, p, aliceID, "AAPL")
	if pos.Quantity != 10 {
		t.Errorf("alice position: want 10, got %d", pos.Quantity)
	}

	// Bob: 25000 + 1500 = 26500
	bob, _ := GetUser(ctx, p, bobID)
	if !bob.Balance.Equal(decimal.NewFromFloat(26500)) {
		t.Errorf("bob balance: want 26500, got %s", bob.Balance)
	}

	// Bob: 100 - 10 = 90 AAPL
	bobPos, _ := GetPosition(ctx, p, bobID, "AAPL")
	if bobPos.Quantity != 90 {
		t.Errorf("bob position: want 90, got %d", bobPos.Quantity)
	}
}

func TestSellOrderHappyPath(t *testing.T) {
	ctx, p := testPool(t)

	p.Exec(ctx, `INSERT INTO positions (user_id, symbol, quantity, avg_cost) VALUES ($1, 'AAPL', 50, 100.0000)`, aliceID)

	sellOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: aliceID, Symbol: "AAPL", Side: models.Sell, Quantity: 20,
		Price: decimal.NewFromFloat(120),
	})
	buyOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: bobID, Symbol: "AAPL", Side: models.Buy, Quantity: 20,
		Price: decimal.NewFromFloat(120),
	})

	match := models.Match{
		BuyOrderID: buyOrder.ID, BuyUserID: bobID,
		SellOrderID: sellOrder.ID, SellUserID: aliceID,
		Quantity: 20, Price: decimal.NewFromFloat(120),
	}
	if err := ExecuteTrade(ctx, p, match, "AAPL"); err != nil {
		t.Fatalf("execute trade: %v", err)
	}

	// Alice: 10000 + 2400 = 12400
	alice, _ := GetUser(ctx, p, aliceID)
	if !alice.Balance.Equal(decimal.NewFromFloat(12400)) {
		t.Errorf("alice balance: want 12400, got %s", alice.Balance)
	}

	// Alice position: 50 - 20 = 30
	pos, _ := GetPosition(ctx, p, aliceID, "AAPL")
	if pos.Quantity != 30 {
		t.Errorf("alice position: want 30, got %d", pos.Quantity)
	}
}

func TestInsufficientFunds(t *testing.T) {
	ctx, p := testPool(t)

	p.Exec(ctx, `INSERT INTO positions (user_id, symbol, quantity, avg_cost) VALUES ($1, 'AAPL', 100, 150.0000)`, bobID)

	sellOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: bobID, Symbol: "AAPL", Side: models.Sell, Quantity: 100,
		Price: decimal.NewFromFloat(150),
	})
	buyOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: aliceID, Symbol: "AAPL", Side: models.Buy, Quantity: 100,
		Price: decimal.NewFromFloat(150),
	})

	match := models.Match{
		BuyOrderID: buyOrder.ID, BuyUserID: aliceID,
		SellOrderID: sellOrder.ID, SellUserID: bobID,
		Quantity: 100, Price: decimal.NewFromFloat(150),
	}

	err := ExecuteTrade(ctx, p, match, "AAPL")
	if err == nil {
		t.Fatal("expected error for insufficient funds (need 15000, have 10000)")
	}

	// Balance should be unchanged
	alice, _ := GetUser(ctx, p, aliceID)
	if !alice.Balance.Equal(decimal.NewFromFloat(10000)) {
		t.Errorf("alice balance should be unchanged at 10000, got %s", alice.Balance)
	}
}

func TestInsufficientShares(t *testing.T) {
	ctx, p := testPool(t)

	p.Exec(ctx, `INSERT INTO positions (user_id, symbol, quantity, avg_cost) VALUES ($1, 'AAPL', 5, 100.0000)`, bobID)

	sellOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: bobID, Symbol: "AAPL", Side: models.Sell, Quantity: 10,
		Price: decimal.NewFromFloat(100),
	})
	buyOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: aliceID, Symbol: "AAPL", Side: models.Buy, Quantity: 10,
		Price: decimal.NewFromFloat(100),
	})

	match := models.Match{
		BuyOrderID: buyOrder.ID, BuyUserID: aliceID,
		SellOrderID: sellOrder.ID, SellUserID: bobID,
		Quantity: 10, Price: decimal.NewFromFloat(100),
	}

	err := ExecuteTrade(ctx, p, match, "AAPL")
	if err == nil {
		t.Fatal("expected error for insufficient shares")
	}
}

func TestIdempotency(t *testing.T) {
	ctx, p := testPool(t)

	req := models.CreateOrderRequest{
		UserID: aliceID, Symbol: "AAPL", Side: models.Buy, Quantity: 10,
		Price: decimal.NewFromFloat(100), IdempotencyKey: "test-key-123",
	}

	o1, err := CreateOrder(ctx, p, req)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Duplicate key should fail
	_, err = CreateOrder(ctx, p, req)
	if err == nil {
		t.Fatal("expected error on duplicate idempotency key")
	}

	// Lookup by key returns the same order
	o2, err := GetOrderByIdempotencyKey(ctx, p, "test-key-123")
	if err != nil {
		t.Fatalf("get by key: %v", err)
	}
	if o1.ID != o2.ID {
		t.Errorf("idempotency mismatch: %s vs %s", o1.ID, o2.ID)
	}
}

func TestConcurrentBuyOrders(t *testing.T) {
	ctx, p := testPool(t)

	// Bob has lots of shares
	p.Exec(ctx, `INSERT INTO positions (user_id, symbol, quantity, avg_cost) VALUES ($1, 'AAPL', 1000, 10.0000)`, bobID)

	// 10 concurrent trades, each 10 shares @ $10 = $100. Total = $1000 out of $10000.
	const n = 10
	var wg sync.WaitGroup
	var succeeded atomic.Int32

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sellOrder, err := CreateOrder(ctx, p, models.CreateOrderRequest{
				UserID: bobID, Symbol: "AAPL", Side: models.Sell, Quantity: 10,
				Price: decimal.NewFromFloat(10),
			})
			if err != nil {
				return
			}
			buyOrder, err := CreateOrder(ctx, p, models.CreateOrderRequest{
				UserID: aliceID, Symbol: "AAPL", Side: models.Buy, Quantity: 10,
				Price: decimal.NewFromFloat(10),
			})
			if err != nil {
				return
			}
			match := models.Match{
				BuyOrderID: buyOrder.ID, BuyUserID: aliceID,
				SellOrderID: sellOrder.ID, SellUserID: bobID,
				Quantity: 10, Price: decimal.NewFromFloat(10),
			}
			if err := ExecuteTrade(ctx, p, match, "AAPL"); err == nil {
				succeeded.Add(1)
			}
		}()
	}
	wg.Wait()

	if succeeded.Load() != int32(n) {
		t.Errorf("expected all %d trades to succeed, got %d", n, succeeded.Load())
	}

	// Alice: 10000 - 1000 = 9000
	alice, _ := GetUser(ctx, p, aliceID)
	if !alice.Balance.Equal(decimal.NewFromFloat(9000)) {
		t.Errorf("alice balance: want 9000, got %s", alice.Balance)
	}
}

func TestConcurrentOverdraft(t *testing.T) {
	ctx, p := testPool(t)

	p.Exec(ctx, `INSERT INTO positions (user_id, symbol, quantity, avg_cost) VALUES ($1, 'AAPL', 1000, 100.0000)`, bobID)

	// 10 goroutines each try $2000 trade. Alice has $10000 → only 5 succeed.
	const n = 10
	var wg sync.WaitGroup
	var succeeded atomic.Int32

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sellOrder, err := CreateOrder(ctx, p, models.CreateOrderRequest{
				UserID: bobID, Symbol: "AAPL", Side: models.Sell, Quantity: 20,
				Price: decimal.NewFromFloat(100),
			})
			if err != nil {
				return
			}
			buyOrder, err := CreateOrder(ctx, p, models.CreateOrderRequest{
				UserID: aliceID, Symbol: "AAPL", Side: models.Buy, Quantity: 20,
				Price: decimal.NewFromFloat(100),
			})
			if err != nil {
				return
			}
			match := models.Match{
				BuyOrderID: buyOrder.ID, BuyUserID: aliceID,
				SellOrderID: sellOrder.ID, SellUserID: bobID,
				Quantity: 20, Price: decimal.NewFromFloat(100),
			}
			if err := ExecuteTrade(ctx, p, match, "AAPL"); err == nil {
				succeeded.Add(1)
			}
		}()
	}
	wg.Wait()

	if succeeded.Load() != 5 {
		t.Errorf("expected 5 successful trades, got %d", succeeded.Load())
	}

	// Balance should be exactly 0 (no negative)
	alice, _ := GetUser(ctx, p, aliceID)
	if !alice.Balance.Equal(decimal.Zero) {
		t.Errorf("alice balance should be 0, got %s", alice.Balance)
	}
}

func TestMatchedTrade(t *testing.T) {
	ctx, p := testPool(t)

	p.Exec(ctx, `INSERT INTO positions (user_id, symbol, quantity, avg_cost) VALUES ($1, 'AAPL', 50, 100.0000)`, bobID)

	sellOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: bobID, Symbol: "AAPL", Side: models.Sell, Quantity: 10,
		Price: decimal.NewFromFloat(150),
	})
	buyOrder, _ := CreateOrder(ctx, p, models.CreateOrderRequest{
		UserID: aliceID, Symbol: "AAPL", Side: models.Buy, Quantity: 10,
		Price: decimal.NewFromFloat(150),
	})

	match := models.Match{
		BuyOrderID: buyOrder.ID, BuyUserID: aliceID,
		SellOrderID: sellOrder.ID, SellUserID: bobID,
		Quantity: 10, Price: decimal.NewFromFloat(150),
	}
	if err := ExecuteTrade(ctx, p, match, "AAPL"); err != nil {
		t.Fatalf("execute trade: %v", err)
	}

	// Trade record exists
	var tradeCount int
	p.QueryRow(ctx, `SELECT COUNT(*) FROM trades WHERE buy_order_id = $1 AND sell_order_id = $2`,
		buyOrder.ID, sellOrder.ID).Scan(&tradeCount)
	if tradeCount != 1 {
		t.Errorf("expected 1 trade record, got %d", tradeCount)
	}

	// Two execution records
	var execCount int
	p.QueryRow(ctx, `SELECT COUNT(*) FROM executions WHERE order_id IN ($1, $2)`,
		buyOrder.ID, sellOrder.ID).Scan(&execCount)
	if execCount != 2 {
		t.Errorf("expected 2 execution records, got %d", execCount)
	}

	// Both orders marked executed
	buy, _ := GetOrder(ctx, p, buyOrder.ID)
	sell, _ := GetOrder(ctx, p, sellOrder.ID)
	if buy.Status != models.StatusExecuted {
		t.Errorf("buy status: want executed, got %s", buy.Status)
	}
	if sell.Status != models.StatusExecuted {
		t.Errorf("sell status: want executed, got %s", sell.Status)
	}
}
