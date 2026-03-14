package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/benniu/tradeengine/internal/cache"
	"github.com/benniu/tradeengine/internal/config"
	"github.com/benniu/tradeengine/internal/db"
	"github.com/benniu/tradeengine/internal/kafka"
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

	log.Println("processor started, consuming from 'orders' topic...")

	consumer.Consume(ctx, func(msg kafkago.Message) error {
		var event models.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("bad message key=%s: %v (skipping)", string(msg.Key), err)
			return nil // commit to avoid poison pill blocking
		}

		order := event.Order
		log.Printf("processing order %s: %s %d %s @ %s",
			order.ID, order.Side, order.Quantity, order.Symbol, order.Price)

		// Update status to validated
		if err := db.UpdateOrderStatus(ctx, pool, order.ID, models.StatusValidated); err != nil {
			return err // transient error — don't commit, will retry
		}

		// Execute the order transactionally
		reason, err := db.ExecuteOrder(ctx, pool, order)
		if err != nil {
			// Transient error — rollback happened, don't commit
			log.Printf("execution error for order %s: %v", order.ID, err)
			return err
		}

		if reason != "" {
			log.Printf("order %s rejected: %s", order.ID, reason)
			event.Reason = reason
		} else {
			log.Printf("order %s executed successfully", order.ID)
			// Refresh order from DB to get updated fields
			updated, err := db.GetOrder(ctx, pool, order.ID)
			if err == nil {
				event.Order = *updated
			}
		}

		// Invalidate cache for the affected user
		cache.InvalidateUserBalance(ctx, rdb, order.UserID)
		cache.InvalidatePositions(ctx, rdb, order.UserID)

		// Publish execution result
		data, _ := json.Marshal(event)
		if err := execProducer.Publish(ctx, order.ID.String(), data); err != nil {
			log.Printf("failed to publish execution result for order %s: %v", order.ID, err)
			// Non-fatal: order is already executed in DB
		}

		return nil
	})
}
