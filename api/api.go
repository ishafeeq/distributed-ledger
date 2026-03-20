package api

import (
	"context"
	"distributed-ledger/internal/engine"
	"distributed-ledger/internal/logger"
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

// Server encapsulates the HTTP API for the ledger.
type Server struct {
	workers   map[int]*engine.ShardWorker
	numShards int
	httpSrv   *http.Server
}

// NewServer creates a new API server with the given workers.
func NewServer(workers map[int]*engine.ShardWorker) *Server {
	return &Server{
		workers:   workers,
		numShards: len(workers),
	}
}

// Run starts the HTTP server and handles graceful shutdown.
func (s *Server) Run(ctx context.Context) {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/balance/", s.handleBalance)

	s.httpSrv = &http.Server{
		Addr:    ":9300",
		Handler: mux,
	}

	go func() {
		logger.Log.Info("HTTP API listening on :9300")
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Error("HTTP Server Error", zap.Error(err))
		}
	}()

	// Wait for context cancellation to shut down
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpSrv.Shutdown(shutdownCtx)
		logger.Log.Info("HTTP API shut down")
	}()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "UP"})
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	accountID := strings.TrimPrefix(r.URL.Path, "/balance/")
	if accountID == "" {
		http.Error(w, "accountID is required", http.StatusBadRequest)
		return
	}

	// Route to shard using FNV-1a
	shardID := engine.GetShard(accountID, s.numShards)
	worker, ok := s.workers[shardID]
	if !ok {
		http.Error(w, "shard not found", http.StatusInternalServerError)
		return
	}

	// Send query command to worker
	respChan := make(chan engine.CommandResult, 1)
	worker.CommandChan <- engine.Command{
		AccountID: accountID,
		Op:        "query",
		Response:  respChan,
	}

	// Wait for response from worker (single-threaded state access)
	result := <-respChan
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accountID": accountID,
		"balance":   result.Balance,
		"shardID":   shardID,
	})
}
