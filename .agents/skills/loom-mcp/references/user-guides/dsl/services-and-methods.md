# DSL: Services And Methods

Use this for `API`, `Service`, `Method`, payload/result, and streaming basics.

## API

```go
var _ = API("calculator", func() {
    Title("Calculator API")
    Description("A simple calculator service")
    Version("1.0.0")
})
```

Common API-level DSL:

- `Title`, `Description`, `Version`
- `TermsOfService`, `Contact`, `License`, `Docs`
- `Server(...)` and `Host(...)`
- shared `Error(...)`, `HTTP(...)`, and `GRPC(...)`

## Service

```go
var _ = Service("users", func() {
    Description("User management service")
    Error("unauthorized")
    Security(OAuth2, func() {
        Scope("read:users")
    })
})
```

## Method

```go
Method("add", func() {
    Payload(func() {
        Field(1, "a", Int32)
        Field(2, "b", Int32)
        Required("a", "b")
    })
    Result(Int32)
    Error("overflow")
})
```

## Payload And Result Shapes

- Primitive payload/result for simple methods
- Inline object payload/result for one-off shapes
- Predeclared `Type(...)` for reuse

## Streaming

- `StreamingPayload` for client streaming
- `StreamingResult` for server streaming
- Use both for bidirectional streaming

```go
Method("chat", func() {
    StreamingPayload(MessageIn)
    StreamingResult(MessageOut)
})
```
