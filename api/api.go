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

// TransactionRequest defines the body for credit/debit operations.
type TransactionRequest struct {
	AccountID string `json:"accountID"`
	Amount    int64  `json:"amount"`
}

// TransactionResponse defines the body for successful transaction responses.
type TransactionResponse struct {
	AccountID string `json:"accountID"`
	Balance   int64  `json:"balance"`
	ShardID   int    `json:"shardID"`
}

// TransferRequest defines the body for account transfers.
type TransferRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
}

// TransferResponse defines the body for successful transfer responses.
type TransferResponse struct {
	FromBalance int64 `json:"fromBalance"`
	ToBalance   int64 `json:"toBalance"`
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
	mux.HandleFunc("/credit", s.handleCredit)
	mux.HandleFunc("/debit", s.handleDebit)
	mux.HandleFunc("/transfer", s.handleTransfer)

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

func (s *Server) handleCredit(w http.ResponseWriter, r *http.Request) {
	s.handleTransaction(w, r, "credit")
}

func (s *Server) handleDebit(w http.ResponseWriter, r *http.Request) {
	s.handleTransaction(w, r, "debit")
}

func (s *Server) handleTransaction(w http.ResponseWriter, r *http.Request, op string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.AccountID == "" || req.Amount <= 0 {
		http.Error(w, "accountID and positive amount are required", http.StatusBadRequest)
		return
	}

	shardID := engine.GetShard(req.AccountID, s.numShards)
	worker, ok := s.workers[shardID]
	if !ok {
		http.Error(w, "shard not found", http.StatusInternalServerError)
		return
	}

	respChan := make(chan engine.CommandResult, 1)
	worker.CommandChan <- engine.Command{
		AccountID: req.AccountID,
		Amount:    req.Amount,
		Op:        op,
		Response:  respChan,
	}

	result := <-respChan
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TransactionResponse{
		AccountID: req.AccountID,
		Balance:   result.Balance,
		ShardID:   shardID,
	})
}

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.From == "" || req.To == "" || req.Amount <= 0 {
		http.Error(w, "from, to, and positive amount are required", http.StatusBadRequest)
		return
	}

	shardIDFrom := engine.GetShard(req.From, s.numShards)
	shardIDTo := engine.GetShard(req.To, s.numShards)

	if shardIDFrom == shardIDTo {
		// Same-shard transfer: Atomic within worker
		worker := s.workers[shardIDFrom]
		respChan := make(chan engine.CommandResult, 1)
		worker.CommandChan <- engine.Command{
			AccountID:   req.From,
			ToAccountID: req.To,
			Amount:      req.Amount,
			Op:          "transfer",
			Response:    respChan,
		}

		result := <-respChan
		if result.Error != nil {
			http.Error(w, result.Error.Error(), http.StatusInternalServerError)
			return
		}

		// Query updated balance for 'To' account to return in response
		respChanTo := make(chan engine.CommandResult, 1)
		worker.CommandChan <- engine.Command{
			AccountID: req.To,
			Op:        "query",
			Response:  respChanTo,
		}
		resultTo := <-respChanTo

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TransferResponse{
			FromBalance: result.Balance,
			ToBalance:   resultTo.Balance,
		})
		return
	}

	// Cross-shard transfer: Coordinated by API
	// 1. Debit Source
	workerFrom := s.workers[shardIDFrom]
	respChanFrom := make(chan engine.CommandResult, 1)
	workerFrom.CommandChan <- engine.Command{
		AccountID: req.From,
		Amount:    req.Amount,
		Op:        "debit",
		Response:  respChanFrom,
	}

	resultFrom := <-respChanFrom
	if resultFrom.Error != nil {
		http.Error(w, resultFrom.Error.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Credit Destination
	workerTo := s.workers[shardIDTo]
	respChanTo := make(chan engine.CommandResult, 1)
	workerTo.CommandChan <- engine.Command{
		AccountID: req.To,
		Amount:    req.Amount,
		Op:        "credit",
		Response:  respChanTo,
	}

	resultTo := <-respChanTo
	if resultTo.Error != nil {
		// Critical: In a real system, we must roll back the debit here or use TCC.
		// For now, we log this inconsistency.
		logger.Log.Error("CRITICAL: Cross-shard transfer inconsistency!",
			zap.String("from", req.From),
			zap.String("to", req.To),
			zap.Int64("amount", req.Amount),
			zap.Error(resultTo.Error))
		http.Error(w, "coordinated transaction failed - destination credit failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TransferResponse{
		FromBalance: resultFrom.Balance,
		ToBalance:   resultTo.Balance,
	})
}
