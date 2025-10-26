# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`circuitbreaker` is a firewall for Lightning Network nodes that protects against HTLC flooding and spam attacks. It acts as an interceptor between LND and the network, enforcing per-peer limits on in-flight HTLCs and applying rate limiting using token buckets.

**Key capabilities:**
- Per-peer HTLC slot limits to prevent channel liquidity lockup
- Rate limiting for HTLC forwarding (token bucket implementation)
- Multiple operating modes: fail, queue, queue_peer_initiated, block
- Web UI for configuration and monitoring
- Stub mode for demos/testing without LND

## Development Commands

### Go Backend

```bash
# Install and run
go install
circuitbreaker --help

# Run with stub mode (no LND required)
circuitbreaker --stub --httplisten 0.0.0.0:9235

# Run tests
go test -v ./...

# Run specific test
go test -v -run TestProcess

# Lint
golangci-lint run
```

### Frontend (Next.js)

Located in `/web` directory:

```bash
cd web
yarn install
yarn dev              # Development server
yarn build            # Production build
yarn export           # Export static files to ../webui-build
yarn build-export     # Build and export
yarn check-types      # TypeScript type checking
yarn check-lint       # ESLint
```

**Note:** The production build uses `build-frontend.sh` which builds via Docker and copies the static files to `webui-build/` directory. The Go binary embeds these files at compile time.

### Protocol Buffers

Located in `/circuitbreakerrpc` directory:

```bash
cd circuitbreakerrpc
make buf-gen                    # Generate protobuf files
make protoc-go-docker           # Generate using Docker (recommended)
```

The `.proto` file defines the gRPC API. Generated files (`*.pb.go`, `*.pb.gw.go`) should be committed.

## Architecture

### Component Structure

**Main process flow:**
- `main.go` - CLI entry point, parses flags
- `run.go` - Application initialization, starts servers and process
- `process.go` - Core HTLC processing engine, manages peer controllers
- `peer_controller.go` - Per-peer rate limiting and HTLC queue management
- `lndclient.go` - LND gRPC client wrapper, handles subscriptions to HTLC events
- `server.go` - gRPC server implementation for CircuitBreaker API
- `db.go` - SQLite database for configuration and HTLC history

**Key interfaces:**
- `lndclient` interface - Abstraction for LND operations (enables stub mode)
- `htlcEventsClient` - Streams HTLC resolution events from LND
- `htlcInterceptorClient` - Intercepts HTLCs before forwarding

### Data Flow

1. **HTLC Interception:** LND forwards HTLCs to circuitbreaker via gRPC stream
2. **Process Distribution:** Main process routes events to appropriate peer controller
3. **Limit Enforcement:** Peer controller checks limits, queues or fails HTLCs
4. **Resolution Tracking:** HTLC events stream back resolution status (settled/failed)
5. **Database Updates:** HTLC history and counters persisted to SQLite

### Operating Modes

Configured per-peer (with default fallback):
- `FAIL` - Immediately fail HTLCs exceeding limits
- `QUEUE` - Queue HTLCs until slots available and rate limit allows (unlimited queue)
- `QUEUE_PEER_INITIATED` - Queue only for peer-opened channels, fail for own channels
- `BLOCK` - Block all HTLCs from peer
- `QUEUE_LIMITED` - Queue HTLCs up to `max_queue_size` limit, then fail for safety (hybrid mode)

### Frontend Integration

The web UI is a Next.js static export embedded in the Go binary at compile time:
- Built files stored in `webui-build/` directory
- Embedded using `//go:embed all:webui-build` in `run.go`
- Served via HTTP alongside gRPC-Gateway REST API
- API calls proxied to gRPC backend

### Database Schema

SQLite database with migrations managed by `rubenv/sql-migrate`:
- `limits` table - Per-peer configuration (max pending, rate limits, mode, max queue size)
  - `max_pending` - Maximum number of in-flight HTLCs
  - `max_hourly_rate` - Maximum HTLCs per hour (rate limiting)
  - `mode` - Operating mode (FAIL, QUEUE, QUEUE_PEER_INITIATED, BLOCK, QUEUE_LIMITED)
  - `max_queue_size` - Maximum queue size for QUEUE_LIMITED mode (0 = unlimited)
- `htlc_info` table - HTLC forwarding history for monitoring
- Default peer (`000...000`) stores global defaults

## Important Implementation Notes

### LND Version Requirements

Requires LND 0.15.4-beta or above. Queue modes depend on LND's auto-fail protection (0.16+) to prevent force-closes.

### Channel Map Tracking

The process maintains a channel map (`chanMap`) mapping channel IDs to peer pubkeys. This is needed because LND's HTLC events only provide channel IDs, not peer identities. The map is refreshed periodically and on new peer detection.

### Rate Limiting Implementation

Uses `golang.org/x/time/rate.Limiter` with token bucket algorithm:
- Burst size configurable (default: 10)
- Rate derived from `MaxHourlyRate` setting
- Tokens consumed per HTLC forward attempt

### Circuit Keys

HTLCs identified by `circuitKey` struct: `{channel uint64, htlc uint64}`. Both incoming and outgoing circuit keys tracked for each in-flight HTLC to properly match settlement/failure events.

### Testing

Test files use:
- `lndclientMock` - Mock LND client for unit tests
- `setupTestDb(t, limit)` - Creates temporary SQLite database
- `Timeout()` - Test timeout helper (in `timeout_test.go`)
- `zaptest.NewLogger(t)` - Structured logging in tests

Standard Go testing with `github.com/stretchr/testify/require` for assertions.

### Docker

Dockerfile multi-stage build:
1. `build_frontend` stage - Builds Next.js web UI
2. `builder` stage - Compiles Go binary with embedded frontend
3. Final stage - Minimal runtime image

Published to `ghcr.io/lightningequipment/circuitbreaker`.

## Configuration

Runtime configuration via command-line flags (see `main.go`):
- `--rpcserver` - LND gRPC address (default: localhost:10009)
- `--lnddir` - LND data directory for TLS/macaroon
- `--network` - Bitcoin network (mainnet/testnet/regtest/simnet)
- `--httplisten` - HTTP server address (default: 127.0.0.1:9235)
- `--listen` - gRPC server address (default: 127.0.0.1:9234)
- `--configdir` - CircuitBreaker config directory (default: ~/.circuitbreaker)
- `--fwdhistorylimit` - Max HTLC history entries to persist
- `--stub` - Enable stub mode

Per-peer limits configured via web UI at http://127.0.0.1:9235, stored in SQLite database.
