package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/google/uuid"

	"github.com/benniu/tradeengine/internal/cache"
	"github.com/benniu/tradeengine/internal/config"
	"github.com/benniu/tradeengine/internal/db"
	"github.com/benniu/tradeengine/internal/kafka"
	"github.com/benniu/tradeengine/internal/matching"
	"github.com/benniu/tradeengine/internal/models"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	rdb := cache.NewClient(cfg.RedisAddr)
	defer rdb.Close()

	consumer := kafka.NewConsumer(cfg.KafkaBrokers, "orders", "order-processor")
	defer consumer.Close()

	execProducer := kafka.NewProducer(cfg.KafkaBrokers, "executions")
	defer execProducer.Close()

	// Initialize matching engine and rebuild order book from DB
	engine := matching.NewEngine()
	openOrders, err := db.GetOpenOrders(ctx, pool)
	if err != nil {
		log.Fatalf("rebuild order book: %v", err)
	}
	for _, o := range openOrders {
		entry := models.BookEntry{
			OrderID:      o.ID,
			UserID:       o.UserID,
			Price:        o.Price,
			RemainingQty: o.Quantity - o.FilledQuantity,
			CreatedAt:    o.CreatedAt,
		}
		engine.LoadOrder(o.Symbol, entry, o.Side)
	}
	log.Printf("order book rebuilt: loaded %d open/partial orders", len(openOrders))

	log.Println("processor started, consuming from 'orders' topic...")

	consumer.Consume(ctx, func(msg kafkago.Message) error {
		var event models.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("bad message key=%s: %v (skipping)", string(msg.Key), err)
			return nil
		}

		order := event.Order
		log.Printf("processing order %s: %s %d %s @ %s",
			order.ID, order.Side, order.Quantity, order.Symbol, order.Price)

		// Update status to validated
		if err := db.UpdateOrderStatus(ctx, pool, order.ID, models.StatusValidated); err != nil {
			return err
		}

		// Build book entry and submit to matching engine
		entry := models.BookEntry{
			OrderID:      order.ID,
			UserID:       order.UserID,
			Price:        order.Price,
			RemainingQty: order.Quantity,
			CreatedAt:    order.CreatedAt,
		}

		matches := engine.Submit(order.Symbol, entry, order.Side)

		// Execute each match against the database
		var trades []models.Trade
		affectedUsers := map[string]bool{order.UserID.String(): true}

		for _, m := range matches {
			if err := db.ExecuteTrade(ctx, pool, m, order.Symbol); err != nil {
				log.Printf("trade execution failed for order %s: %v", order.ID, err)
				// Remove the resting order from the book since it failed validation
				// (e.g., seller no longer has shares)
				if m.BuyOrderID == order.ID {
					engine.RemoveOrder(order.Symbol, m.SellOrderID)
				} else {
					engine.RemoveOrder(order.Symbol, m.BuyOrderID)
				}
				continue
			}

			trades = append(trades, models.Trade{
				Symbol:      order.Symbol,
				BuyOrderID:  m.BuyOrderID,
				SellOrderID: m.SellOrderID,
				Quantity:    m.Quantity,
				Price:       m.Price,
			})

			// Track all affected users for cache invalidation
			affectedUsers[m.BuyUserID.String()] = true
			affectedUsers[m.SellUserID.String()] = true
		}

		// If order has remaining quantity, set status to open
		if entry.RemainingQty > 0 && len(matches) == 0 {
			if err := db.UpdateOrderStatus(ctx, pool, order.ID, models.StatusOpen); err != nil {
				log.Printf("failed to set order %s to open: %v", order.ID, err)
			}
		}

		if len(trades) > 0 {
			log.Printf("order %s: %d trades executed", order.ID, len(trades))
		} else {
			log.Printf("order %s: resting in book (no matches)", order.ID)
		}

		// Invalidate cache for ALL affected users
		for uid := range affectedUsers {
			userID, _ := uuid.Parse(uid)
			cache.InvalidateUserBalance(ctx, rdb, userID)
			cache.InvalidatePositions(ctx, rdb, userID)
		}

		// Publish execution result
		event.Trades = trades
		updated, err := db.GetOrder(ctx, pool, order.ID)
		if err == nil {
			event.Order = *updated
		}

		data, _ := json.Marshal(event)
		if err := execProducer.Publish(ctx, order.ID.String(), data); err != nil {
			log.Printf("failed to publish execution result for order %s: %v", order.ID, err)
		}

		return nil
	})
}
