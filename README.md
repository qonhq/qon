# qon

High-performance HTTP/HTTPS networking engine written in Go, designed as the core for multi-language Qon bindings.

## Implemented

- HTTP client engine with GET/POST/PUT/DELETE/PATCH/HEAD support
- Header and query parameter handling
- Request/response body buffering and response streaming support
- Connection pooling and keep-alive tuning
- Priority-aware request scheduler with bounded concurrency
- Retry strategies with exponential backoff
- Circuit breaker per target host
- Optional rate limiting token bucket
- TLS configuration with certificate validation
- Proxy support and HTTP/2 toggle
- Structured errors by category
- Metrics API (total/success/fail/retry/bytes/latency)
- Request tracing via trace IDs and propagated header
- Access-key gate for server mode integrations
- Plugin hooks for request lifecycle extensibility
- Bridge mode via JSON over stdin/stdout
- Future bridge framing codec for binary protocol migration
- Optional server mode exposing /request, /metrics, /health

## Project Structure

- cmd/qon: CLI entrypoint for bridge/server execution modes
- internal/config: runtime configuration defaults and types
- internal/core: engine, scheduler, retries, circuit breaker, metrics, plugins
- internal/bridge: IPC bridge implementations
- internal/server: HTTP server mode wrapper
- pkg/qon: minimal public package wrapper

## Run

Build:

```bash
go build ./...
```

Bridge mode (stdin/stdout JSON lines):

```bash
go run ./cmd/qon -mode bridge
```

Server mode:

```bash
go run ./cmd/qon -mode server -addr :8080
```

## Bridge Request Example

Input line:

```json
{"method":"GET","url":"https://httpbin.org/get","timeout_ms":5000,"priority":1}
```

Output line:

```json
{"status":200,"headers":{"Content-Type":"application/json"},"body_base64":"...","duration_ms":42,"trace_id":"qon-1"}
```

## Tests

```bash
go test ./...
```

## Notes on Future Features

Qon already includes forward-compatible foundations for upcoming capabilities:

- Binary bridge framing codec for persistent multiplexed transports
- Priority scheduler that can be upgraded to weighted queues
- Plugin lifecycle hooks for middleware-style extensions
- Access-key and rate-limiting primitives for service mode hardening
- DNS cache module ready to be wired into custom dial path
