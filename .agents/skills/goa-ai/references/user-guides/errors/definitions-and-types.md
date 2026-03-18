# Errors: Definitions And Types

Use this for API/service/method error scope, `ErrorResult`, and custom error types.

## Scopes

- API-level errors for cross-service reuse
- Service-level errors for all methods in one service
- Method-level errors for operation-specific cases

## Default `ErrorResult`

Includes:

- `Name`
- `ID`
- `Message`
- `Temporary`
- `Timeout`
- `Fault`

Generated helpers typically look like:

```go
func MakeDivByZero(err error) *goa.ServiceError
```

## Custom Error Types

Use a custom `Type(...)` only when clients need more structured context.

```go
Field(3, "name", String, func() {
    Meta("struct:error:name")
})
```

That metadata is required when multiple custom errors share a method and Goa needs to know which field carries the error name.
