package engine

import (
	"context"
	"distributed-ledger/internal/logger"
	"distributed-ledger/internal/store"
)

// Bootstrapper handles state reconstruction on startup by replaying the WAL.
type Bootstrapper struct {
	WAL store.WAL
}

func NewBootstrapper(wal store.WAL) *Bootstrapper {
	return &Bootstrapper{WAL: wal}
}

// Recover replays from Redis Streams to populate in-memory states for all shard workers.
func (b *Bootstrapper) Recover(ctx context.Context, workers map[int]*ShardWorker) error {
	logger.Log.Info("Recovery: Replaying Redis Streams WAL")
	
	err := b.WAL.Replay(ctx, func(accountID string, amount int64, op string) {
		shardID := GetShard(accountID, len(workers))
		if worker, ok := workers[shardID]; ok {
			switch op {
			case "credit":
				worker.State[accountID] += amount
			case "debit":
				worker.State[accountID] -= amount
			}
		}
	})
	
	if err != nil {
		return err // Wrap with logger at caller or here if needed
	}
	
	logger.Log.Info("Recovery: State reconstruction complete")
	return nil
}
