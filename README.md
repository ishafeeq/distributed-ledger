# High-Throughput Distributed Ledger

A lock-free, shard-based distributed ledger built for extreme performance using Go, NATS JetStream, and Redis Streams (WAL).

## Architecture Overview

### 1. Lock-Free Concurrency
To eliminate `sync.Mutex` contention, the system uses a **Single-Writer Shard Worker** model. Each account is strictly owned by exactly one worker thread (goroutine).
- **Routing**: Uses **FNV-1a Consistent Hashing** to map an `AccountID` to its owner `ShardID`.
- **Command Dispatch**: Commands (credit, debit, query) are sent to the owner's `CommandChan` (buffered).
- **Serialized Execution**: The worker processes commands one-at-a-time, removing the need for internal locking.

### 2. Durability & Recovery
- **Write-Ahead Log (WAL)**: All state-changing operations are appended to **Redis Streams** before being applied to the in-memory state.
- **Bootstrapping**: On startup, the `Bootstrapper` replays the Redis Streams to reconstruct the in-memory account map for each shard.

## Transaction Flows

### Credit / Debit Transactions
1.  **Ingress**: A transaction request arrives (via the `/balance` API or NATS).
2.  **Routing**: `engine.GetShard(accountID)` calculates the destination shard.
3.  **Validation**: The command is sent to the worker. For a `debit`, the worker checks the in-memory balance.
4.  **WAL Append**: (Ongoing Integration) The operation is appended to the Redis WAL for durability.
5.  **State Update**: The worker updates the `map[string]int64` in-memory state.
6.  **Response**: The worker returns the new balance.

### Balance Inquiries
1.  **Request**: `GET /balance/{accountID}` on port **9300**.
2.  **Routing**: The API layer calculates the owner shard.
3.  **Query Command**: A "query" op is sent to the worker's queue.
4.  **Response**: The worker reads its local map and returns the balance safely.

## Project Structure

- `/api`: HTTP handlers and server orchestration.
- `/cmd/ledger`: Application entry point.
- `/internal/engine`: Core logic (ShardWorker, Consistent Hashing, Reaper).
- `/internal/store`: Write-Ahead Log (WAL) implementation.
- `/internal/config`: Secret loading (Zero-Leak Policy).
- `/internal/logger`: High-performance Zap structured logging.

## Deployment & Verification

Deploying to EC2 (Ubuntu) is handled via **Taskfile.yml**:

```bash
# Deploy code and start infrastructure (NATS, Redis, Ledger)
task deploy

# Test remote health
task test:api
```

The application runs inside Docker (see `docker-compose.yml`) and uses **Docker Secrets** for sensitive data.
