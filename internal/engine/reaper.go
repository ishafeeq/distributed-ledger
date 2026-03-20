package engine

import (
	"context"
	"distributed-ledger/internal/logger"
	"time"
)

// TCCState represents the phase of a Try-Confirm-Cancel transaction.
type TCCState string

const (
	TCC_TRY     TCCState = "TRY"
	TCC_CONFIRM TCCState = "CONFIRM"
	TCC_CANCEL  TCCState = "CANCEL"
)

// Transaction tracks cross-shard transfers for distributed consistency.
type Transaction struct {
	ID        string
	From      string
	To        string
	Amount    int64
	State     TCCState
	Expiry    time.Time
}

// Reaper is a background process that reconciles stuck distributed transactions.
type Reaper struct {
	Interval time.Duration
}

// NewReaper initializes a new Reaper with the given interval.
func NewReaper(interval time.Duration) *Reaper {
	return &Reaper{Interval: interval}
}

// Start runs the Reaper's scan loop.
func (r *Reaper) Start(ctx context.Context) {
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.reconcile()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Reaper) reconcile() {
	// 1. Fetch pending TCC transactions from persistent storage
	// 2. Resolve based on the source shard worker's local state
	// 3. Cleanup or roll back
	logger.Log.Info("Reaper: Scanning for pending distributed transactions")
}
