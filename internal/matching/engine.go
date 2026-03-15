package matching

import (
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/benniu/tradeengine/internal/models"
)

// OrderBook holds the bid and ask sides for a single symbol.
type OrderBook struct {
	Bids []models.BookEntry // sorted: price DESC, then CreatedAt ASC
	Asks []models.BookEntry // sorted: price ASC, then CreatedAt ASC
}

// Engine manages order books for all symbols.
type Engine struct {
	mu    sync.Mutex
	books map[string]*OrderBook
}

func NewEngine() *Engine {
	return &Engine{books: make(map[string]*OrderBook)}
}

func (e *Engine) getBook(symbol string) *OrderBook {
	book, ok := e.books[symbol]
	if !ok {
		book = &OrderBook{}
		e.books[symbol] = book
	}
	return book
}

// Submit places an incoming order into the book and attempts to match it
// against resting orders on the opposite side. Returns all matches produced.
func (e *Engine) Submit(symbol string, entry models.BookEntry, side models.OrderSide) []models.Match {
	e.mu.Lock()
	defer e.mu.Unlock()

	book := e.getBook(symbol)
	var matches []models.Match

	switch side {
	case models.Buy:
		// Match against asks (lowest price first)
		i := 0
		for i < len(book.Asks) && entry.RemainingQty > 0 {
			ask := &book.Asks[i]
			// Buy at entry.Price can match asks with price <= entry.Price
			if ask.Price.GreaterThan(entry.Price) {
				break // asks are sorted ASC, no more matches possible
			}

			matchQty := min(entry.RemainingQty, ask.RemainingQty)
			matches = append(matches, models.Match{
				BuyOrderID:  entry.OrderID,
				BuyUserID:   entry.UserID,
				SellOrderID: ask.OrderID,
				SellUserID:  ask.UserID,
				Quantity:    matchQty,
				Price:       ask.Price, // execute at resting order's price
			})

			entry.RemainingQty -= matchQty
			ask.RemainingQty -= matchQty

			if ask.RemainingQty == 0 {
				// Remove fully filled ask
				book.Asks = append(book.Asks[:i], book.Asks[i+1:]...)
			} else {
				i++
			}
		}

		// If remaining, add to bids
		if entry.RemainingQty > 0 {
			insertBid(book, entry)
		}

	case models.Sell:
		// Match against bids (highest price first)
		i := 0
		for i < len(book.Bids) && entry.RemainingQty > 0 {
			bid := &book.Bids[i]
			// Sell at entry.Price can match bids with price >= entry.Price
			if bid.Price.LessThan(entry.Price) {
				break // bids are sorted DESC, no more matches possible
			}

			matchQty := min(entry.RemainingQty, bid.RemainingQty)
			matches = append(matches, models.Match{
				BuyOrderID:  bid.OrderID,
				BuyUserID:   bid.UserID,
				SellOrderID: entry.OrderID,
				SellUserID:  entry.UserID,
				Quantity:    matchQty,
				Price:       bid.Price, // execute at resting order's price
			})

			entry.RemainingQty -= matchQty
			bid.RemainingQty -= matchQty

			if bid.RemainingQty == 0 {
				book.Bids = append(book.Bids[:i], book.Bids[i+1:]...)
			} else {
				i++
			}
		}

		// If remaining, add to asks
		if entry.RemainingQty > 0 {
			insertAsk(book, entry)
		}
	}

	return matches
}

// LoadOrder inserts an order into the book without matching.
// Used to rebuild the order book from the database on processor restart.
func (e *Engine) LoadOrder(symbol string, entry models.BookEntry, side models.OrderSide) {
	e.mu.Lock()
	defer e.mu.Unlock()

	book := e.getBook(symbol)
	switch side {
	case models.Buy:
		insertBid(book, entry)
	case models.Sell:
		insertAsk(book, entry)
	}
}

// RemoveOrder removes an order from the book (e.g., when a resting order
// fails validation during trade execution).
func (e *Engine) RemoveOrder(symbol string, orderID uuid.UUID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	book, ok := e.books[symbol]
	if !ok {
		return false
	}

	for i, b := range book.Bids {
		if b.OrderID == orderID {
			book.Bids = append(book.Bids[:i], book.Bids[i+1:]...)
			return true
		}
	}
	for i, a := range book.Asks {
		if a.OrderID == orderID {
			book.Asks = append(book.Asks[:i], book.Asks[i+1:]...)
			return true
		}
	}
	return false
}

// GetBookDepth returns the current bid/ask entries for a symbol (for debugging/API).
func (e *Engine) GetBookDepth(symbol string) (bids, asks []models.BookEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()

	book, ok := e.books[symbol]
	if !ok {
		return nil, nil
	}
	return append([]models.BookEntry{}, book.Bids...), append([]models.BookEntry{}, book.Asks...)
}

// insertBid adds an entry to bids in sorted order (price DESC, time ASC).
func insertBid(book *OrderBook, entry models.BookEntry) {
	idx := sort.Search(len(book.Bids), func(i int) bool {
		if book.Bids[i].Price.Equal(entry.Price) {
			return book.Bids[i].CreatedAt.After(entry.CreatedAt)
		}
		return book.Bids[i].Price.LessThan(entry.Price)
	})
	book.Bids = append(book.Bids, models.BookEntry{})
	copy(book.Bids[idx+1:], book.Bids[idx:])
	book.Bids[idx] = entry
}

// insertAsk adds an entry to asks in sorted order (price ASC, time ASC).
func insertAsk(book *OrderBook, entry models.BookEntry) {
	idx := sort.Search(len(book.Asks), func(i int) bool {
		if book.Asks[i].Price.Equal(entry.Price) {
			return book.Asks[i].CreatedAt.After(entry.CreatedAt)
		}
		return book.Asks[i].Price.GreaterThan(entry.Price)
	})
	book.Asks = append(book.Asks, models.BookEntry{})
	copy(book.Asks[idx+1:], book.Asks[idx:])
	book.Asks[idx] = entry
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
