# Errors: Transport And Formatting

Use this for HTTP/gRPC mappings, formatter customization, and test expectations.

## Transport Mapping

```go
HTTP(func() {
    Response("DivByZero", StatusBadRequest)
})

GRPC(func() {
    Response("DivByZero", CodeInvalidArgument)
})
```

Keep mappings consistent across transports.

## Producing Errors

- Prefer generated `Make...` helpers for `ErrorResult`-backed errors.
- Return custom generated structs directly only when the design uses a custom error type.

## Custom HTTP Formatter

Use a custom `goahttp.Statuser` formatter when the wire error shape must differ from the default.

## Tests

Test the generated error name or generated custom type explicitly:

```go
if serr, ok := err.(*goa.ServiceError); !ok || serr.Name != "DivByZero" { ... }
```
