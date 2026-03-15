package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/benniu/tradeengine/internal/cache"
	"github.com/benniu/tradeengine/internal/config"
	"github.com/benniu/tradeengine/internal/db"
	"github.com/benniu/tradeengine/internal/kafka"
	"github.com/benniu/tradeengine/internal/models"
	"github.com/benniu/tradeengine/internal/ws"
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

	producer := kafka.NewProducer(cfg.KafkaBrokers, "orders")
	defer producer.Close()

	h := &handler{pool: pool, rdb: rdb, producer: producer}

	// WebSocket hub + Kafka execution consumer
	hub := ws.NewHub()
	go ws.StartExecutionConsumer(ctx, cfg.KafkaBrokers, hub)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/ws", hub.HandleWS)
	r.Post("/orders", h.createOrder)
	r.Get("/orders/{id}", h.getOrder)
	r.Get("/orders", h.listOrders)
	r.Get("/positions", h.listPositions)
	r.Get("/users/{id}", h.getUser)

	srv := &http.Server{
		Addr:    ":" + cfg.GatewayPort,
		Handler: r,
	}

	go func() {
		log.Printf("gateway listening on :%s", cfg.GatewayPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gateway...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

type handler struct {
	pool     *pgxpool.Pool
	rdb      *redis.Client
	producer *kafka.Producer
}

func (h *handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate required fields
	if req.UserID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}
	if req.Side != models.Buy && req.Side != models.Sell {
		writeError(w, http.StatusBadRequest, "side must be 'buy' or 'sell'")
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}
	if req.Price.IsNegative() || req.Price.IsZero() {
		writeError(w, http.StatusBadRequest, "price must be positive")
		return
	}

	ctx := r.Context()

	// Idempotency check
	if req.IdempotencyKey != "" {
		// Check Redis first (fast path)
		if orderID, found, err := cache.CheckIdempotency(ctx, h.rdb, req.IdempotencyKey); err == nil && found {
			id, _ := uuid.Parse(orderID)
			existing, err := db.GetOrder(ctx, h.pool, id)
			if err == nil {
				writeJSON(w, http.StatusOK, existing)
				return
			}
		}
		// Check DB (durable path)
		if existing, err := db.GetOrderByIdempotencyKey(ctx, h.pool, req.IdempotencyKey); err == nil {
			cache.SetIdempotency(ctx, h.rdb, req.IdempotencyKey, existing.ID.String())
			writeJSON(w, http.StatusOK, existing)
			return
		}
	}

	// Create order in DB
	order, err := db.CreateOrder(ctx, h.pool, req)
	if err != nil {
		log.Printf("create order: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	// Cache idempotency key
	if req.IdempotencyKey != "" {
		cache.SetIdempotency(ctx, h.rdb, req.IdempotencyKey, order.ID.String())
	}

	// Publish to Kafka
	event := models.OrderEvent{Order: *order}
	data, _ := json.Marshal(event)
	if err := h.producer.Publish(ctx, order.ID.String(), data); err != nil {
		log.Printf("publish order: %v", err)
		// Order is created in DB but not published — processor won't pick it up.
		// In production, add a retry mechanism or outbox pattern.
		writeError(w, http.StatusInternalServerError, "order created but failed to queue for processing")
		return
	}

	writeJSON(w, http.StatusAccepted, order)
}

func (h *handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := db.GetOrder(r.Context(), h.pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (h *handler) listOrders(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id query param is required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	orders, err := db.GetOrdersByUser(r.Context(), h.pool, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}
	if orders == nil {
		orders = []models.Order{}
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *handler) listPositions(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeError(w, http.StatusBadRequest, "user_id query param is required")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	ctx := r.Context()

	// Try cache first
	if positions, err := cache.GetPositions(ctx, h.rdb, userID); err == nil && positions != nil {
		writeJSON(w, http.StatusOK, positions)
		return
	}

	// Fallback to DB
	positions, err := db.GetPositionsByUser(ctx, h.pool, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list positions")
		return
	}
	if positions == nil {
		positions = []models.Position{}
	}

	// Populate cache
	cache.SetPositions(ctx, h.rdb, userID, positions)

	writeJSON(w, http.StatusOK, positions)
}

func (h *handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	ctx := r.Context()

	// Try cache for balance
	user, dbErr := db.GetUser(ctx, h.pool, id)
	if dbErr != nil {
		if errors.Is(dbErr, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Cache balance for next time
	cache.SetUserBalance(ctx, h.rdb, id, user.Balance)

	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
