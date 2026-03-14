package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/benniu/tradeengine/internal/models"
)

const (
	balanceTTL     = 30 * time.Second
	positionsTTL   = 30 * time.Second
	idempotencyTTL = 1 * time.Hour
)

func NewClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}

func balanceKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:balance", userID)
}

func positionsKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:positions", userID)
}

func idempotencyKey(key string) string {
	return fmt.Sprintf("idempotency:%s", key)
}

func GetUserBalance(ctx context.Context, rdb *redis.Client, userID uuid.UUID) (*decimal.Decimal, error) {
	val, err := rdb.Get(ctx, balanceKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d, err := decimal.NewFromString(val)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func SetUserBalance(ctx context.Context, rdb *redis.Client, userID uuid.UUID, balance decimal.Decimal) error {
	return rdb.Set(ctx, balanceKey(userID), balance.String(), balanceTTL).Err()
}

func InvalidateUserBalance(ctx context.Context, rdb *redis.Client, userID uuid.UUID) error {
	return rdb.Del(ctx, balanceKey(userID)).Err()
}

func GetPositions(ctx context.Context, rdb *redis.Client, userID uuid.UUID) ([]models.Position, error) {
	val, err := rdb.Get(ctx, positionsKey(userID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var positions []models.Position
	if err := json.Unmarshal([]byte(val), &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

func SetPositions(ctx context.Context, rdb *redis.Client, userID uuid.UUID, positions []models.Position) error {
	data, err := json.Marshal(positions)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, positionsKey(userID), data, positionsTTL).Err()
}

func InvalidatePositions(ctx context.Context, rdb *redis.Client, userID uuid.UUID) error {
	return rdb.Del(ctx, positionsKey(userID)).Err()
}

func CheckIdempotency(ctx context.Context, rdb *redis.Client, key string) (string, bool, error) {
	val, err := rdb.Get(ctx, idempotencyKey(key)).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func SetIdempotency(ctx context.Context, rdb *redis.Client, key string, orderID string) error {
	return rdb.Set(ctx, idempotencyKey(key), orderID, idempotencyTTL).Err()
}
