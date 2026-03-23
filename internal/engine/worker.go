package engine

import (
	"context"
	"distributed-ledger/internal/logger"
	"fmt"
	"go.uber.org/zap"
)

// Command represents a transaction command sent to a shard worker.
type Command struct {
	AccountID   string
	ToAccountID string // Used for transfers
	Amount      int64
	Op          string // "credit", "debit", "query", "transfer"
	Response    chan CommandResult
}

// CommandResult contains the outcome of a command.
type CommandResult struct {
	Error   error
	Balance int64
}

// ShardWorker is a single-threaded processor for a specific partition of accounts.
type ShardWorker struct {
	ID          int
	State       map[string]int64
	CommandChan chan Command
}

// NewShardWorker creates a new lock-free shard worker.
func NewShardWorker(id int) *ShardWorker {
	// Using Go 1.26 hypothetical 'new(expr)' if applicable, 
	// but standard composite literal is idiomatic.
	return &ShardWorker{
		ID:          id,
		State:       make(map[string]int64),
		CommandChan: make(chan Command, 1024),
	}
}

// Run starts the main execution loop for the worker.
func (w *ShardWorker) Run(ctx context.Context) {
	logger.Log.Info("Shard Worker started", zap.Int("id", w.ID))
	for {
		select {
		case cmd := <-w.CommandChan:
			w.handleCommand(cmd)
		case <-ctx.Done():
			return
		}
	}
}

func (w *ShardWorker) handleCommand(cmd Command) {
	// TODO: Integrate with Redis WAL in internal/store
	
	switch cmd.Op {
	case "credit":
		w.State[cmd.AccountID] += cmd.Amount
		cmd.Response <- CommandResult{Balance: w.State[cmd.AccountID]}
	case "debit":
		if w.State[cmd.AccountID] < cmd.Amount {
			cmd.Response <- CommandResult{Error: fmt.Errorf("insufficient balance")}
			return
		}
		w.State[cmd.AccountID] -= cmd.Amount
		cmd.Response <- CommandResult{Balance: w.State[cmd.AccountID]}
	case "transfer":
		if w.State[cmd.AccountID] < cmd.Amount {
			cmd.Response <- CommandResult{Error: fmt.Errorf("insufficient balance")}
			return
		}
		w.State[cmd.AccountID] -= cmd.Amount
		w.State[cmd.ToAccountID] += cmd.Amount
		cmd.Response <- CommandResult{Balance: w.State[cmd.AccountID]}
	case "query":
		cmd.Response <- CommandResult{Balance: w.State[cmd.AccountID]}
	default:
		cmd.Response <- CommandResult{Error: fmt.Errorf("unknown operation: %s", cmd.Op)}
	}
}
