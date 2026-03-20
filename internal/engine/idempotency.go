package engine

import (
	"sync"
)

// IdempotencyManager prevents double-processing of transactions from NATS retries.
type IdempotencyManager struct {
	mu     sync.RWMutex
	exists map[string]struct{} // TODO: replace with high-speed bitset or LRU
}

func NewIdempotencyManager() *IdempotencyManager {
	return &IdempotencyManager{
		exists: make(map[string]struct{}),
	}
}

// IsDuplicate returns true if the transaction ID has already been seen.
func (m *IdempotencyManager) IsDuplicate(txID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.exists[txID]
	return ok
}

// MarkProcessed records the transaction ID in the idempotency set.
func (m *IdempotencyManager) MarkProcessed(txID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exists[txID] = struct{}{}
}
