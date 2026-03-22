# gRPC: Streaming And Errors

Use this for gRPC streaming modes and status-code mapping.

## Streaming Modes

- `StreamingResult` for server-side streaming
- `StreamingPayload` for client-side streaming
- both for bidirectional streaming

Implementation rules:

- Handle `ctx.Done()`
- Handle `io.EOF` correctly
- Keep message sizes reasonable
- Apply flow control consciously

## Error Mapping

```go
GRPC(func() {
    Response(CodeOK)
    Response("not_found", CodeNotFound)
    Response("invalid_input", CodeInvalidArgument)
})
```

Common mappings:

- `not_found` -> `CodeNotFound`
- `invalid_argument` -> `CodeInvalidArgument`
- `internal_error` -> `CodeInternal`
- `unauthenticated` -> `CodeUnauthenticated`
- `permission_denied` -> `CodePermissionDenied`
