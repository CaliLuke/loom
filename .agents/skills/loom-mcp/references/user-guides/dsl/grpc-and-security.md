# DSL: gRPC And Security

Use this for `GRPC(...)` mappings, metadata/message separation, and auth DSL.

## gRPC Mapping

```go
Method("create", func() {
    Payload(CreatePayload)
    Result(User)
    GRPC(func() {
        Response(CodeOK)
    })
})
```

Useful gRPC DSL:

- `Metadata(...)` for metadata/header mapping
- `Message(...)` for request/response message contents
- `Response(...)` for status-code mapping

## Field Numbering

- Reserve `1..15` for hot fields
- Use `16+` for less frequent fields

## Security Schemes

- `JWTSecurity(...)`
- `APIKeySecurity(...)`
- `BasicAuthSecurity(...)`
- `OAuth2Security(...)`

Apply security at API, service, or method scope:

```go
Method("secure_endpoint", func() {
    Security(JWTAuth, func() {
        Scope("api:read")
    })
})
```

## Implementation Reminder

Goa generates `Auther` methods for the configured schemes. Validate credentials and scopes there, not in transport glue.
