---
title: Interceptors
weight: 7
description: "Complete guide to interceptors and middleware in Loom - type-safe Loom interceptors, HTTP middleware, and gRPC interceptors."
llm_optimized: true
aliases:
---

Loom provides a comprehensive solution for request processing that combines type-safe interceptors with traditional middleware patterns. This guide covers all three approaches.

## Overview

When processing requests in a Loom service, you have three complementary tools:

1. **Loom Interceptors**: Type-safe, compile-time checked access to your service's domain types
2. **HTTP Middleware**: Standard `http.Handler` pattern for HTTP-specific concerns
3. **gRPC Interceptors**: Standard gRPC patterns for RPC-specific needs

### When to Use Each

| Concern | Tool |
|---------|------|
| Business logic validation | Loom Interceptors |
| Data transformation | Loom Interceptors |
| Request/response enrichment | Loom Interceptors |
| Logging, tracing | HTTP/gRPC Middleware |
| Compression, CORS | HTTP Middleware |
| Metadata handling | gRPC Interceptors |
| Rate limiting | HTTP/gRPC Middleware |

---

## Loom Interceptors

Loom interceptors provide type-safe access to your service's domain types, with compile-time checks and generated helper methods.

### Runtime Model (Generated Code)

Interceptors are not runtime hooks hidden inside the transport. In Loom they are **generated endpoint wrappers**. The DSL tells Loom what fields an interceptor may read/write, and code generation produces:

- **Service-side contract** in `gen/<service>/service_interceptors.go`
  - `ServerInterceptors` interface: one method per interceptor
  - `*<Interceptor>Info` structs: service/method metadata + accessors
  - `*Payload` / `*Result` accessor interfaces: only the fields you declared as readable/writable
- **Client-side contract** in `gen/<service>/client_interceptors.go`
  - `ClientInterceptors` interface and its `*Info` + accessor types
- **Wrapper chain** in `gen/<service>/interceptor_wrappers.go`
  - Per-method `Wrap<Method>Endpoint` and `Wrap<Method>ClientEndpoint`
  - For streaming methods, stream wrappers that intercept `SendWithContext` / `RecvWithContext`
- **Wiring** in `gen/<service>/endpoints.go` and `gen/<service>/client.go`
  - `NewEndpoints` applies server wrappers around service endpoints
  - `NewClient` applies client wrappers around transport endpoints

The important consequence is: **server interceptors execute after transport decoding and before your service method**, and **client interceptors execute before transport encoding and after transport decoding** (because they wrap the same typed endpoint abstraction your client code calls).

### The Server Interceptor Contract

Loom generates a per-service interface. Each interceptor method must call `next` exactly once to proceed (or return an error/response early):

```go
type ServerInterceptors interface {
    RequestAudit(ctx context.Context, info *RequestAuditInfo, next loom.Endpoint) (any, error)
}
```

At runtime you’ll typically:

1. Use `info.Payload()` / `info.Result(res)` for **type-safe** access (preferred).
2. Use `info.Service()`, `info.Method()`, and `info.CallType()` to tag logs/metrics with stable identifiers.
3. Call `next(ctx, info.RawPayload())` to continue the chain.
4. Optionally mutate the payload before calling `next`, or mutate the result after it returns.

Example (result enrichment + timing):

```go
type Interceptors struct{}

func (i *Interceptors) RequestAudit(ctx context.Context, info *RequestAuditInfo, next loom.Endpoint) (any, error) {
    start := time.Now()

    res, err := next(ctx, info.RawPayload())
    if err != nil {
        return nil, err
    }

    r := info.Result(res)
    r.SetProcessedAt(time.Now().UTC().Format(time.RFC3339Nano))
    r.SetDuration(int(time.Since(start).Milliseconds()))

    return res, nil
}
```

Why the accessor interfaces matter:

- If you declare `ReadPayload(Attribute("recordID"))`, Loom generates `RecordID() <type>`.
- If you declare `WriteResult(Attribute("cachedAt"))`, Loom generates `SetCachedAt(<type>)`.
- Your interceptor can’t accidentally reach for fields you didn’t declare; that’s the compile-time contract.

### The Client Interceptor Contract

Client interceptors are the same idea on the client side: they wrap the transport endpoint you pass into `gen/<service>.NewClient(...)`.

In practice that means:

- `info.RawPayload()` is the **typed method payload** (e.g. `*GetPayload`), not an `*http.Request`.
- If you “write” to the payload in the interceptor, the transport endpoint will encode those changes (headers/body/etc.) according to your transport mappings.
- You can use `info.Result(res)` to read/write result fields after the transport decodes the response.

### Ordering (What Actually Runs First)

Interceptors are applied by generating a wrapper chain. The generated `Wrap<Method>Endpoint` is the source of truth for ordering.

Conceptually, codegen does this:

```go
func WrapGetEndpoint(endpoint loom.Endpoint, i ServerInterceptors) loom.Endpoint {
    endpoint = wrapGetCache(endpoint, i)
    endpoint = wrapGetJWTAuth(endpoint, i)
    endpoint = wrapGetRequestAudit(endpoint, i)
    endpoint = wrapGetSetDeadline(endpoint, i)
    endpoint = wrapGetTraceRequest(endpoint, i)
    return endpoint
}
```

Each `wrap...` returns a new endpoint that calls the interceptor with `next` pointing at the previous endpoint. As a result:

- **On the way in (request path)**: the **last** wrapper runs first.
- **On the way out (response path)**: the **first** wrapper runs first.

If ordering matters, rely on this mental model rather than a generic “service then method” rule: **read the generated wrap function for the method you care about**.

### Streaming Interceptors (Send/Recv)

For bidirectional streaming, codegen wraps the stream implementation so that each message send/receive is intercepted. A single interceptor method may be invoked for multiple call types:

- `loom.InterceptorUnary`: one-time interception of the stream endpoint call
- `loom.InterceptorStreamingSend`: interception of each `SendWithContext`
- `loom.InterceptorStreamingRecv`: interception of each `RecvWithContext`

Use `info.CallType()` to branch when needed. For send interceptions, `info.RawPayload()` is the message being sent. For recv interceptions, the “payload” is produced by `next` (your interceptor sees it as the returned value).

### Defining Interceptors

```go
var RequestLogger = Interceptor("RequestLogger", func() {
    Description("Logs incoming requests and their timing")
    
    // Read status from result
    ReadResult(func() {
        Attribute("status")
    })
    
    // Add timing information to result
    WriteResult(func() {
        Attribute("processedAt")
        Attribute("duration")
    })
})
```

### Applying Interceptors

Apply at service or method level:

```go
var _ = Service("calculator", func() {
    // Apply to all methods
    ServerInterceptor(RequestLogger)
    
    Method("add", func() {
        // Method-specific interceptor
        ServerInterceptor(ValidateNumbers)
        
        Payload(func() {
            Attribute("a", Int)
            Attribute("b", Int)
        })
        Result(func() {
            Attribute("sum", Int)
            Attribute("status", Int)
            Attribute("processedAt", String)
            Attribute("duration", Int)
        })
    })
})
```

### Implementing Interceptors

```go
func (i *Interceptors) RequestLogger(ctx context.Context, info *RequestLoggerInfo, next loom.Endpoint) (any, error) {
    start := time.Now()
    
    // Call next interceptor or endpoint
    res, err := next(ctx, info.RawPayload())
    if err != nil {
        return nil, err
    }
    
    // Access result through type-safe interface
    r := info.Result(res)
    
    // Add timing information
    r.SetProcessedAt(time.Now().Format(time.RFC3339))
    r.SetDuration(int(time.Since(start).Milliseconds()))
    
    return res, nil
}
```

### Access Patterns

#### Read-Only Access

```go
var Monitor = Interceptor("Monitor", func() {
    Description("Collects metrics without modifying data")
    
    ReadPayload(func() {
        Attribute("size")
    })
    
    ReadResult(func() {
        Attribute("status")
    })
})
```

#### Write Access

```go
var Enricher = Interceptor("Enricher", func() {
    Description("Adds context information")
    
    WritePayload(func() {
        Attribute("requestID")
    })
    
    WriteResult(func() {
        Attribute("processedAt")
    })
})
```

#### Combined Access

```go
var DataProcessor = Interceptor("DataProcessor", func() {
    Description("Processes both requests and responses")
    
    ReadPayload(func() {
        Attribute("rawData")
    })
    WritePayload(func() {
        Attribute("processed")
    })
    
    ReadResult(func() {
        Attribute("status")
    })
    WriteResult(func() {
        Attribute("enriched")
    })
})
```

### Client-Side Interceptors

```go
var ClientContext = Interceptor("ClientContext", func() {
    Description("Enriches requests with client context")
    
    WritePayload(func() {
        Attribute("clientVersion")
        Attribute("clientID")
    })
    
    ReadResult(func() {
        Attribute("rateLimit")
        Attribute("rateLimitRemaining")
    })
})

var _ = Service("inventory", func() {
    ClientInterceptor(ClientContext)
    // ...
})
```

### Streaming Interceptors

For streaming methods, use streaming variants:

```go
var ServerProgressTracker = Interceptor("ServerProgressTracker", func() {
    Description("Adds progress to server stream responses")
    
    WriteStreamingResult(func() {
        Attribute("percentComplete")
        Attribute("itemsProcessed")
    })
})

var ClientMetadataEnricher = Interceptor("ClientMetadataEnricher", func() {
    Description("Enriches outgoing client stream messages")
    
    WriteStreamingPayload(func() {
        Attribute("clientTimestamp")
    })
})
```

### Execution Order

Loom applies interceptors by building a wrapper chain around each method endpoint. The easiest way to understand the exact ordering (especially once you mix service-level + method-level interceptors) is to look at the generated `Wrap<Method>Endpoint` and remember:

- **Last wrapper executes first** on the request path.
- **First wrapper executes first** on the response path.

If you need a stable contract in your own code, treat the generated wrap function as the canonical ordering specification for that method.

---

## HTTP Middleware

HTTP middleware handles protocol-level concerns using the standard `http.Handler` pattern.

### Common Middleware Stack

```go
import (
    "github.com/CaliLuke/loom/clue/debug"
    "github.com/CaliLuke/loom/clue/log"
    loomhttp "github.com/CaliLuke/loom/http"
    loomhttpmiddleware "github.com/CaliLuke/loom/http/middleware"
    loomhttpotel "github.com/CaliLuke/loom/http/middleware/otel"
)

mux := loomhttp.NewMuxer()

// Add middleware (outermost to innermost)
mux.Use(debug.HTTP())                               // Debug logging
mux.Use(loomhttpotel.Middleware("service"))         // OpenTelemetry
mux.Use(log.HTTP(ctx))                              // Request logging
mux.Use(loomhttpmiddleware.PopulateRequestContext()) // Loom context
```

`loom/http/middleware.RequestID` still exists for compatibility, but it is
deprecated. Prefer the OpenTelemetry middleware and application-owned request ID
handling for new services.

### Creating Custom Middleware

```go
func ExampleMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing
        start := time.Now()
        
        next.ServeHTTP(w, r)
        
        // Post-processing
        log.Printf("Request took %v", time.Since(start))
    })
}
```

### Security Headers Middleware

```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        next.ServeHTTP(w, r)
    })
}
```

### Context Enrichment Middleware

```go
func ContextEnrichmentMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        ctx = context.WithValue(ctx, "request.start", time.Now())
        ctx = context.WithValue(ctx, "request.id", r.Header.Get("X-Request-ID"))
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Error Handling Middleware

```go
func ErrorHandlingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("panic recovered: %v", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        
        next.ServeHTTP(w, r)
    })
}
```

---

## gRPC Interceptors

gRPC interceptors handle protocol-level concerns for RPC calls.

### Unary Interceptors

```go
func LoggingInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        start := time.Now()
        
        resp, err := handler(ctx, req)
        
        log.Printf("Method: %s, Duration: %v, Error: %v",
            info.FullMethod, time.Since(start), err)
        
        return resp, err
    }
}
```

### Stream Interceptors

```go
func StreamLoggingInterceptor() grpc.StreamServerInterceptor {
    return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
        start := time.Now()
        
        err := handler(srv, ss)
        
        log.Printf("Stream: %s, Duration: %v, Error: %v",
            info.FullMethod, time.Since(start), err)
        
        return err
    }
}
```

### Integration with Loom

```go
func main() {
    srv := grpc.NewServer(
        grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
            MetadataInterceptor(),
            LoggingInterceptor(),
            MonitoringInterceptor(),
        )),
        grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
            StreamMetadataInterceptor(),
            StreamLoggingInterceptor(),
        )),
    )

    pb.RegisterServiceServer(srv, server)
}
```

---

## Combining All Three

Here's how all three approaches work together:

```go
func main() {
    // 1. Create service with Loom interceptors
    svc := NewService()
    interceptors := NewInterceptors(log.Default())
    endpoints := NewEndpoints(svc, interceptors)
    
    // 2. Set up HTTP with middleware
    mux := loomhttp.NewMuxer()
    mux.Use(loomhttpotel.Middleware("payment-svc"))
    mux.Use(debug.HTTP())
    mux.Use(log.HTTP(ctx))
    
    httpServer := genhttp.New(endpoints, mux, dec, enc, eh, eh)
    genhttp.Mount(mux, httpServer)
    
    // 3. Set up gRPC with interceptors
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
            grpc_recovery.UnaryServerInterceptor(),
            grpc_prometheus.UnaryServerInterceptor,
        )),
    )
    
    grpcSvr := gengrpc.New(endpoints, nil)
    genpb.RegisterPaymentServer(grpcServer, grpcSvr)
}
```

### Execution Flow

```
Request Processing:
─────────────────────────────────────────────────────────────────>
HTTP/gRPC Middleware → Loom Interceptors → Service Method

Response Processing:
<─────────────────────────────────────────────────────────────────
Service Method → Loom Interceptors → HTTP/gRPC Middleware
```

---

## Best Practices

### Loom Interceptors
- Use for business logic validation and data transformation
- Keep interceptors focused on single responsibilities
- Use type-safe access patterns

### HTTP Middleware
- Order middleware carefully (panic recovery first, then logging, etc.)
- Pre-compile expensive objects (regex, etc.)
- Use sync.Pool for frequently allocated objects

### gRPC Interceptors
- Focus on protocol-level concerns
- Handle context cancellation properly
- Use appropriate status codes

### General
- Test interceptors/middleware in isolation
- Consider performance impact
- Document the purpose of each interceptor

---

## See Also

- [DSL Reference](dsl-reference.md) — Interceptor DSL definitions
- [HTTP Guide](http-guide.md) — HTTP-specific middleware patterns
- [gRPC Guide](grpc-guide.md) — gRPC interceptor patterns
- [Loom log package](https://pkg.go.dev/github.com/CaliLuke/loom/clue/log) — endpoint and transport logging middleware
