# Interceptors: HTTP And gRPC Middleware

Use this for protocol-level middleware outside Goa's typed interceptor system.

## HTTP Middleware

Typical stack:

```go
mux.Use(debug.HTTP())
mux.Use(otelhttp.NewMiddleware("service"))
mux.Use(log.HTTP(ctx))
mux.Use(goahttpmiddleware.RequestID())
```

Use HTTP middleware for:

- logging and tracing
- compression and CORS
- panic recovery
- request-context enrichment

## gRPC Interceptors

Use gRPC interceptors for protocol-level unary/stream concerns:

```go
grpc.NewServer(
    grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
        LoggingInterceptor(),
    )),
)
```

Use them for:

- metadata handling
- RPC-level logging
- protocol metrics
- stream-level cross-cutting behavior

## Rule Of Thumb

- Business/domain transforms: Goa interceptors
- Protocol concerns: HTTP middleware or gRPC interceptors
