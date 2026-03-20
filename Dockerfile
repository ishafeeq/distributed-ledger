# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy all source files
COPY . .

# Generate go.sum and build the binary
RUN go mod tidy && go build -o /bin/ledger ./cmd/ledger/main.go

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /app

# Install certificates
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /bin/ledger /app/ledger

# Ensure secrets directory exists
RUN mkdir -p /run/secrets

ENTRYPOINT ["/app/ledger"]
