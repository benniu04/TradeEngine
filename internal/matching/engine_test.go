package matching

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/benniu/tradeengine/internal/models"
)

func makeEntry(price float64, qty int) models.BookEntry {
	return models.BookEntry{
		OrderID:      uuid.New(),
		UserID:       uuid.New(),
		Price:        decimal.NewFromFloat(price),
		RemainingQty: qty,
		CreatedAt:    time.Now(),
	}
}

func TestBasicMatch(t *testing.T) {
	e := NewEngine()

	sell := makeEntry(100, 10)
	buy := makeEntry(100, 10)

	// Sell rests in book
	matches := e.Submit("AAPL", sell, models.Sell)
	if len(matches) != 0 {
		t.Fatal("sell should not match against empty book")
	}

	// Buy matches the sell
	matches = e.Submit("AAPL", buy, models.Buy)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	m := matches[0]
	if m.Quantity != 10 {
		t.Errorf("expected qty 10, got %d", m.Quantity)
	}
	if !m.Price.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("expected price 100, got %s", m.Price)
	}
	if m.BuyOrderID != buy.OrderID {
		t.Error("buy order ID mismatch")
	}
	if m.SellOrderID != sell.OrderID {
		t.Error("sell order ID mismatch")
	}

	// Book should be empty
	bids, asks := e.GetBookDepth("AAPL")
	if len(bids) != 0 || len(asks) != 0 {
		t.Errorf("book should be empty, got %d bids, %d asks", len(bids), len(asks))
	}
}

func TestPartialFill(t *testing.T) {
	e := NewEngine()

	sell := makeEntry(100, 60)
	buy := makeEntry(100, 100)

	e.Submit("AAPL", sell, models.Sell)
	matches := e.Submit("AAPL", buy, models.Buy)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].Quantity != 60 {
		t.Errorf("expected partial fill of 60, got %d", matches[0].Quantity)
	}

	// Buy should remain in book with 40 remaining
	bids, asks := e.GetBookDepth("AAPL")
	if len(asks) != 0 {
		t.Errorf("asks should be empty, got %d", len(asks))
	}
	if len(bids) != 1 {
		t.Fatalf("expected 1 resting bid, got %d", len(bids))
	}
	if bids[0].RemainingQty != 40 {
		t.Errorf("expected remaining 40, got %d", bids[0].RemainingQty)
	}
}

func TestNoMatch(t *testing.T) {
	e := NewEngine()

	sell := makeEntry(150, 10) // ask at 150
	buy := makeEntry(100, 10) // bid at 100

	e.Submit("AAPL", sell, models.Sell)
	matches := e.Submit("AAPL", buy, models.Buy)

	if len(matches) != 0 {
		t.Fatalf("should not match, buy@100 < sell@150")
	}

	bids, asks := e.GetBookDepth("AAPL")
	if len(bids) != 1 || len(asks) != 1 {
		t.Errorf("both should rest, got %d bids, %d asks", len(bids), len(asks))
	}
}

func TestPriceTimePriority(t *testing.T) {
	e := NewEngine()

	// Two sells at same price, first one should match first
	sell1 := makeEntry(100, 10)
	sell1.CreatedAt = time.Now().Add(-time.Minute) // earlier
	sell2 := makeEntry(100, 10)
	sell2.CreatedAt = time.Now()

	e.Submit("AAPL", sell1, models.Sell)
	e.Submit("AAPL", sell2, models.Sell)

	buy := makeEntry(100, 10)
	matches := e.Submit("AAPL", buy, models.Buy)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].SellOrderID != sell1.OrderID {
		t.Error("should match sell1 (earlier) first, got sell2")
	}

	// sell2 should still be in book
	_, asks := e.GetBookDepth("AAPL")
	if len(asks) != 1 || asks[0].OrderID != sell2.OrderID {
		t.Error("sell2 should remain in book")
	}
}

func TestMultipleMatches(t *testing.T) {
	e := NewEngine()

	// Three sells: 30 + 40 + 30
	s1 := makeEntry(100, 30)
	s2 := makeEntry(100, 40)
	s3 := makeEntry(100, 30)

	e.Submit("AAPL", s1, models.Sell)
	e.Submit("AAPL", s2, models.Sell)
	e.Submit("AAPL", s3, models.Sell)

	// Buy for 100 — should match all three
	buy := makeEntry(100, 100)
	matches := e.Submit("AAPL", buy, models.Buy)

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}

	totalFilled := 0
	for _, m := range matches {
		totalFilled += m.Quantity
	}
	if totalFilled != 100 {
		t.Errorf("expected total fill of 100, got %d", totalFilled)
	}

	// Book should be empty
	bids, asks := e.GetBookDepth("AAPL")
	if len(bids) != 0 || len(asks) != 0 {
		t.Errorf("book should be empty, got %d bids, %d asks", len(bids), len(asks))
	}
}

func TestBuyMatchesCheaperAskFirst(t *testing.T) {
	e := NewEngine()

	// Ask at 90 and ask at 100
	cheap := makeEntry(90, 10)
	expensive := makeEntry(100, 10)

	e.Submit("AAPL", expensive, models.Sell)
	e.Submit("AAPL", cheap, models.Sell)

	// Buy at 100 should match cheap ask first (better price)
	buy := makeEntry(100, 10)
	matches := e.Submit("AAPL", buy, models.Buy)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if !matches[0].Price.Equal(decimal.NewFromFloat(90)) {
		t.Errorf("should execute at resting ask price 90, got %s", matches[0].Price)
	}
}

func TestSellMatchesHigherBidFirst(t *testing.T) {
	e := NewEngine()

	// Bid at 90 and bid at 100
	low := makeEntry(90, 10)
	high := makeEntry(100, 10)

	e.Submit("AAPL", low, models.Buy)
	e.Submit("AAPL", high, models.Buy)

	// Sell at 90 should match high bid first (better price)
	sell := makeEntry(90, 10)
	matches := e.Submit("AAPL", sell, models.Sell)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if !matches[0].Price.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("should execute at resting bid price 100, got %s", matches[0].Price)
	}
}

func TestLoadOrderDoesNotMatch(t *testing.T) {
	e := NewEngine()

	sell := makeEntry(100, 10)
	buy := makeEntry(100, 10)

	// LoadOrder should NOT trigger matching
	e.LoadOrder("AAPL", sell, models.Sell)
	e.LoadOrder("AAPL", buy, models.Buy)

	bids, asks := e.GetBookDepth("AAPL")
	if len(bids) != 1 || len(asks) != 1 {
		t.Errorf("both should rest without matching, got %d bids, %d asks", len(bids), len(asks))
	}
}

func TestRemoveOrder(t *testing.T) {
	e := NewEngine()

	entry := makeEntry(100, 10)
	e.Submit("AAPL", entry, models.Sell)

	removed := e.RemoveOrder("AAPL", entry.OrderID)
	if !removed {
		t.Error("should have removed the order")
	}

	_, asks := e.GetBookDepth("AAPL")
	if len(asks) != 0 {
		t.Error("book should be empty after removal")
	}
}

func TestDifferentSymbolsIndependent(t *testing.T) {
	e := NewEngine()

	aaplSell := makeEntry(100, 10)
	googlBuy := makeEntry(100, 10)

	e.Submit("AAPL", aaplSell, models.Sell)
	matches := e.Submit("GOOGL", googlBuy, models.Buy)

	if len(matches) != 0 {
		t.Fatal("different symbols should not match")
	}
}
