---
title: Production
weight: 8
description: "Production-ready patterns for Loom services - observability, security, and common deployment patterns."
llm_optimized: true
aliases:
---

This guide covers essential patterns for running Loom services in production, including observability, security, and common deployment patterns.

## Observability

Modern distributed systems require comprehensive observability. Loom includes
small Clue packages for structured logging, HTTP/gRPC logging middleware, health
checks, and debug endpoints:

- `github.com/CaliLuke/loom/clue/log`
- `github.com/CaliLuke/loom/clue/health`
- `github.com/CaliLuke/loom/clue/debug`
- `github.com/CaliLuke/loom/clue/mock`

Applications can either configure OpenTelemetry directly or use Loom's
`observability/otel` bootstrap helpers. The Clue packages provide
framework-adjacent logging and health helpers that work with generated Loom
servers.

### Logging Context

```go
import (
    "context"

    "github.com/CaliLuke/loom/clue/log"
)

func main() {
    format := log.FormatJSON
    if log.IsTerminal() {
        format = log.FormatTerminal
    }

    ctx := log.Context(context.Background(),
        log.WithFormat(format),
        log.WithFunc(log.Span))

    _ = ctx
}
```

### Logging

```go
import "github.com/CaliLuke/loom/clue/log"

func (s *Service) CreateOrder(ctx context.Context, order *Order) error {
    log.Info(ctx,
        log.KV{K: "msg", V: "processing order"},
        log.KV{K: "order_id", V: order.ID},
        log.KV{K: "amount", V: order.Amount})

    if err := s.processOrder(ctx, order); err != nil {
        log.Error(ctx, err,
            log.KV{K: "msg", V: "failed to process order"},
            log.KV{K: "order_id", V: order.ID})
        return err
    }

    return nil
}
```

### OpenTelemetry Runtime

Use `github.com/CaliLuke/loom/observability/otel` when a service wants
framework-owned OpenTelemetry provider setup plus Loom-aware HTTP and gRPC
transport instrumentation:

```go
import (
    "context"

    loomhttp "github.com/CaliLuke/loom/http"
    loomotel "github.com/CaliLuke/loom/observability/otel"
)

func main() {
    rt, err := loomotel.New(context.Background(), loomotel.Config{
        ServiceName:    "orders",
        ServiceVersion: "2026.05.11",
        Environment:    "production",
        Traces: loomotel.TraceConfig{
            Enabled:     true,
            Endpoint:    "otel-collector:4318",
            Insecure:    true,
            SampleRatio: 0.25,
        },
        Metrics: loomotel.MetricConfig{
            Enabled:  true,
            Endpoint: "otel-collector:4318",
            Insecure: true,
        },
        Logs: loomotel.LogConfig{
            Enabled:  true,
            Endpoint: "otel-collector:4318",
            Insecure: true,
        },
    })
    if err != nil {
        panic(err)
    }
    defer rt.Shutdown(context.Background())

    mux := loomhttp.NewMuxer()
    mux.Use(loomotel.HTTPMiddleware(loomotel.HTTPConfig{
        ServiceName:    "orders",
        MetricMode:     loomotel.HTTPMetricModeBoth,
        TracerProvider: rt.TracerProvider,
        MeterProvider:  rt.MeterProvider,
        Propagators:    rt.Propagators,
    }))
}
```

`HTTPMetricMode` controls whether HTTP transport metrics come from the official
OpenTelemetry HTTP instrumentation, a custom recorder, both, or neither. Use
`loomotel.AddHTTPAttributes(ctx, ...)` from downstream middleware to attach
request-scoped transport attributes after the span has started. For gRPC,
prefer `loomotel.GRPCServerOption(...)` and `loomotel.GRPCClientOption(...)`.
If a service uses logrus and wants OTLP log export, use
`github.com/CaliLuke/loom/observability/otel/logrusbridge`.

If the application already owns OpenTelemetry providers, use
`github.com/CaliLuke/loom/http/middleware/otel` and
`github.com/CaliLuke/loom/grpc/middleware/otel` directly as thin wrappers around
the official OpenTelemetry HTTP and gRPC instrumentation.

### Generated Transport Observer

Generated HTTP, JSON-RPC, and Loom-MCP servers emit safe, classified events
at decode, dispatch, handler, panic, response-write, and stream-write
boundaries through the dependency-free
`github.com/CaliLuke/loom/observability/transport` contract. Enablement is
context-based, so generated constructor signatures stay unchanged — turning
observability on or off is purely a wiring choice at the application
boundary. Generated code never emits raw bodies, JSON-RPC params, MCP tool
arguments, credentials, or result payloads; events carry only
low-cardinality classification fields safe for metric labels and log
enrichment.

```go
import (
    loomhttp "github.com/CaliLuke/loom/http"
    "github.com/CaliLuke/loom/observability/transport"
)

func main() {
    obs := transport.ObserverFunc(func(_ context.Context, e transport.Event) {
        log.Printf("%s/%s %s reason=%s status=%d bytes=%d duration=%s",
            e.Transport, e.Kind, e.Method, e.Reason,
            e.StatusCode, e.BytesWritten, e.Duration)
    })

    mux := loomhttp.NewMuxer()
    mux.Use(transport.HTTPMiddleware(obs))
}
```

`transport.HTTPMiddleware` only injects the observer into the request
context; span/trace setup, propagation, and metric recording remain in
`observability/otel`. The two are composable — stack them in any order;
neither package depends on the other.

`Event.Reason` is a stable enumeration suitable for metric labels:
`ok`, `request_decode_failed`, `invalid_jsonrpc_envelope`,
`invalid_jsonrpc_batch`, `invalid_jsonrpc_method`,
`invalid_jsonrpc_params`, `unsupported_method`, `missing_credentials`,
`invalid_credentials`, `permission_rejected`, `principal_mismatch`,
`handler_error`, `panic`, `response_write_failed`, `stream_write_failed`,
`stream_flush_failed`, `stream_write_timeout`, `stream_flush_timeout`,
`stream_final_response_suppressed`, `mcp_session_missing`,
`mcp_session_not_found`, `mcp_session_principal_mismatch`, and
`mcp_events_stream_write_failed`.

The two timeout reasons distinguish a configured streaming deadline from an
ordinary writer or flusher failure. `stream_final_response_suppressed` is a
successful JSON-RPC protocol decision: an ID-less SSE notification or raw GET
listener called `SendAndClose`, so Loom closed the stream without emitting an
invalid final response. Do not count that reason as a server fault; use it to
find implementations that should send their final value with `Send` instead.

For non-HTTP entry points (e.g. a JSON-RPC consumer reading frames from a
queue) use `transport.WithObserver(ctx, obs)` to inject the observer into
the request context before invoking the generated handler.

### CORS

Model browser cross-origin policy in the HTTP or JSON-RPC design with `CORS`
or `RuntimeCORS()` rather than wrapping generated handlers in
application-local middleware.
Generated servers mount route-local `OPTIONS` preflight handlers and write
`Access-Control-Allow-*` headers from the shared `loom/http` runtime helper.
Service-level CORS overrides API-level CORS, and OpenAPI publishes the
effective HTTP route policy under `x-loom-cors`. JSON-RPC keeps Go's
`CrossOriginProtection` secure default when no CORS policy is designed;
`Origin("*")` without credentials is the explicit allow-all opt-out.

Use static `CORS` when the allowed origins are part of the design contract. Use
`RuntimeCORS()` when deployment configuration owns those values. At startup,
build a raw `loomhttp.CORSPolicy`, call `loomhttp.NewRuntimeCORSPolicy`, and pass
the validated immutable snapshot to the generated server constructor. Treat a
validation error as a startup configuration failure. Loom does not read
environment variables and does not live-reload the snapshot.

CORS does not authorize WebSocket origins. Configure the generated server's
WebSocket upgrader `CheckOrigin` policy separately; CORS headers on the HTTP
upgrade response are not a substitute for handshake origin enforcement.

### Health Checks

```go
import (
    "context"

    "github.com/CaliLuke/loom/clue/health"
    "github.com/CaliLuke/loom/clue/log"
)

func main() {
    ctx := log.Context(context.Background(), log.WithFormat(log.FormatJSON))
    checker := health.NewChecker(
        health.NewPinger("database", dbHealthAddr),
        health.NewPinger("cache", cacheHealthAddr),
    )

    http.Handle("/healthz", log.HTTP(ctx)(health.Handler(checker)))
}
```

### Complete Observable Service Skeleton

```go
func main() {
    ctx := log.Context(context.Background(), log.WithFormat(log.FormatJSON))

    svc := NewService()
    endpoints := genservice.NewEndpoints(svc)
    endpoints.Use(debug.LogPayloads())
    endpoints.Use(log.Endpoint)

    mux := loomhttp.NewMuxer()
    mux.Use(debug.HTTP())
    mux.Use(log.HTTP(ctx))

    // Mount debug endpoints
    debug.MountDebugLogEnabler(debug.Adapt(mux))
    debug.MountPprofHandlers(debug.Adapt(mux))

    // Mount health checks
    mux.Handle("/healthz", health.Handler(health.NewChecker(...)))

    // Start server
    server := &http.Server{Addr: ":8080", Handler: mux}
    server.ListenAndServe()
}
```

---

## Security

Loom provides robust security features through its DSL.

### Security Schemes

#### Basic Authentication

```go
var BasicAuth = BasicAuthSecurity("basic", func() {
    Description("Basic authentication using username and password")
})
```

#### API Key Authentication

```go
var APIKeyAuth = APIKeySecurity("api_key", func() {
    Description("Secures endpoint by requiring an API key")
})
```

#### JWT Authentication

```go
var JWTAuth = JWTSecurity("jwt", func() {
    Description("Secures endpoint by requiring a valid JWT token")
    Scope("api:read", "Read access to API")
    Scope("api:write", "Write access to API")
})
```

#### OAuth2 Authentication

```go
var OAuth2 = OAuth2Security("oauth2", func() {
    Description("OAuth2 authentication")
    AuthorizationCodeFlow("/authorize", "/token", "/refresh")
    Scope("api:write", "Write access")
    Scope("api:read", "Read access")
})
```

### Applying Security

Security can be applied at API, service, or method level:

```go
// API level - default for all endpoints
var _ = API("myapi", func() {
    Security(BasicAuth)
})

// Service level - override API default
var _ = Service("users", func() {
    Security(APIKeyAuth)

    Method("list", func() {
        // Uses service-level APIKeyAuth
        Payload(func() {
            APIKey("api_key", "key", String)
            Required("key")
        })
    })

    Method("admin", func() {
        // Override with JWT for this method
        Security(JWTAuth)
        Payload(func() {
            Token("token", String)
            Required("token")
        })
    })

    Method("public", func() {
        // No security for this method
        NoSecurity()
    })
})
```

### Security Best Practices

1. **Always use HTTPS in production**
2. **Define security at API level** for consistent defaults
3. **Use `NoSecurity()` explicitly** for public endpoints
4. **Implement rate limiting** for API key authentication
5. **Use appropriate token expiration** for JWT tokens
6. **Regularly rotate secrets and keys**
7. **Log and monitor authentication failures**
8. **Validate all input** even for authenticated requests

---

## Common Patterns

### Graceful Shutdown

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())

    // Create server
    server := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    // Start server in goroutine
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Errorf(ctx, err, "server error")
        }
    }()

    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutdownCancel()

    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Errorf(ctx, err, "shutdown error")
    }

    cancel()
    wg.Wait()
}
```

### Configuration Management

```go
type Config struct {
    HTTPAddr     string        `env:"HTTP_ADDR" default:":8080"`
    GRPCAddr     string        `env:"GRPC_ADDR" default:":8081"`
    DatabaseURL  string        `env:"DATABASE_URL" required:"true"`
    LogLevel     string        `env:"LOG_LEVEL" default:"info"`
    ReadTimeout  time.Duration `env:"READ_TIMEOUT" default:"10s"`
    WriteTimeout time.Duration `env:"WRITE_TIMEOUT" default:"30s"`
}

func main() {
    var cfg Config
    if err := envconfig.Process("", &cfg); err != nil {
        log.Fatal(err)
    }

    server := &http.Server{
        Addr:         cfg.HTTPAddr,
        Handler:      mux,
        ReadTimeout:  cfg.ReadTimeout,
        WriteTimeout: cfg.WriteTimeout,
    }
}
```

### Server Timeouts

```go
server := &http.Server{
    Addr:              ":8080",
    Handler:           mux,
    ReadHeaderTimeout: 10 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      60 * time.Second,
    IdleTimeout:       120 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1MB
}
```

---

## Summary

Production-ready Loom services should include:

1. **Observability**: Tracing, metrics, logging, and health checks
2. **Security**: Appropriate authentication and authorization
3. **Resilience**: Graceful shutdown, timeouts, and error handling
4. **Configuration**: Environment-based configuration management
5. **Monitoring**: Debug endpoints and profiling capabilities

These patterns ensure your services are reliable, secure, and maintainable in production environments.

---

## See Also

- [Loom log package](https://pkg.go.dev/github.com/CaliLuke/loom/clue/log) — structured logging and HTTP/gRPC middleware
- [Loom health package](https://pkg.go.dev/github.com/CaliLuke/loom/clue/health) — health check handler and dependency pingers
- [Loom debug package](https://pkg.go.dev/github.com/CaliLuke/loom/clue/debug) — debug log toggles, payload logging, and pprof mounting
- [DSL Reference: Security](dsl-reference.md#security) — Security scheme definitions
- [Error Handling Guide](error-handling.md) — Error handling patterns and best practices
- [Interceptors](interceptors.md) — Middleware and interceptor patterns
