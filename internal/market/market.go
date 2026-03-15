package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/benniu/tradeengine/internal/models"
)

const quoteTTL = 15 * time.Second

// Client fetches real-time stock quotes from Finnhub.
type Client struct {
	apiKey string
	http   *http.Client
	rdb    *redis.Client
}

func NewClient(apiKey string, rdb *redis.Client) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 5 * time.Second},
		rdb:    rdb,
	}
}

func quoteKey(symbol string) string {
	return fmt.Sprintf("quote:%s", symbol)
}

// GetQuote returns a cached or fresh quote for the given symbol.
func (c *Client) GetQuote(ctx context.Context, symbol string) (*models.Quote, error) {
	// Try cache first
	if val, err := c.rdb.Get(ctx, quoteKey(symbol)).Result(); err == nil {
		var q models.Quote
		if json.Unmarshal([]byte(val), &q) == nil {
			return &q, nil
		}
	}

	// Fetch from Finnhub
	url := fmt.Sprintf("https://finnhub.io/api/v1/quote?symbol=%s&token=%s", symbol, c.apiKey)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("finnhub request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("finnhub %d: %s", resp.StatusCode, string(body))
	}

	var q models.Quote
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return nil, fmt.Errorf("decode quote: %w", err)
	}

	// Cache the result
	if data, err := json.Marshal(q); err == nil {
		c.rdb.Set(ctx, quoteKey(symbol), data, quoteTTL)
	}

	return &q, nil
}
