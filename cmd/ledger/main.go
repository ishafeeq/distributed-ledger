package main

import (
	"context"
	"distributed-ledger/api"
	"distributed-ledger/internal/config"
	"distributed-ledger/internal/engine"
	"distributed-ledger/internal/logger"
	"distributed-ledger/internal/store"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger.Init()
	defer logger.Sync()
	logger.Log.Info("Starting High-Throughput Distributed Ledger")

	// 1. Load Secrets (Zero-Leak Policy)
	redisPass, err := config.LoadSecret("redis_password")
	if err != nil {
		logger.Log.Warn("Could not load redis_password", zap.Error(err))
	}
	_ = redisPass // To be used in Redis WAL init

	natsToken, err := config.LoadSecret("nats_token")
	if err != nil {
		logger.Log.Warn("Could not load nats_token", zap.Error(err))
	}
	_ = natsToken // To be used in NATS init

	// 2. Initialize Infrastructure Stubs
	wal := store.NewRedisWAL()
	bootstrapper := engine.NewBootstrapper(wal)
	
	// 3. Initialize Shard Workers (e.g., 3 shards)
	numShards := 3
	workers := make(map[int]*engine.ShardWorker)
	for i := 0; i < numShards; i++ {
		workers[i] = engine.NewShardWorker(i)
	}

	// 4. Recovery Phase
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := bootstrapper.Recover(ctx, workers); err != nil {
		logger.Log.Fatal("Startup Error", zap.Error(err))
	}

	// 5. Start Shard Workers
	for _, worker := range workers {
		go worker.Run(ctx)
	}

	// 6. Start Reaper for TCC reconciliation
	reaper := engine.NewReaper(10 * time.Second)
	go reaper.Start(ctx)

	// 7. Start HTTP API
	apiSrv := api.NewServer(workers)
	apiSrv.Run(ctx)

	// 8. Wait for Termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Log.Info("Shutting down ledger")
}
