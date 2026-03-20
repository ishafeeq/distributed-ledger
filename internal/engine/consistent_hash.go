package engine

import (
	"hash/fnv"
)

// GetShard returns the shard index for a given account ID using FNV-1a.
func GetShard(accountID string, numShards int) int {
	h := fnv.New32a()
	h.Write([]byte(accountID))
	return int(h.Sum32()) % numShards
}
