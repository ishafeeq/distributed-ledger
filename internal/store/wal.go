package store

import (
	"context"
	"distributed-ledger/internal/logger"
	"go.uber.org/zap"
)

// WAL (Write-Ahead Log) interface for durability.
type WAL interface {
	Append(ctx context.Context, accountID string, amount int64, op string) error
	Replay(ctx context.Context, fn func(accountID string, amount int64, op string)) error
}

// RedisWAL implements the WAL interface using Redis Streams.
type RedisWAL struct {
	// redis client placeholder
}

func NewRedisWAL() *RedisWAL {
	return &RedisWAL{}
}

func (r *RedisWAL) Append(ctx context.Context, accountID string, amount int64, op string) error {
	// TODO: implement xadd call to Redis Streams
	logger.Log.Debug("WAL: Appending", 
		zap.String("op", op), 
		zap.Int64("amount", amount), 
		zap.String("accountID", accountID))
	return nil
}

func (r *RedisWAL) Replay(ctx context.Context, fn func(accountID string, amount int64, op string)) error {
	// TODO: implement xread call to Redis Streams
	return nil
}
