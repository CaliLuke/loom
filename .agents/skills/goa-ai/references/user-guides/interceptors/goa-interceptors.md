# Interceptors: Goa Interceptors

Use this for generated interceptor wrappers, accessor contracts, and execution ordering.

## Runtime Model

Goa interceptors are generated endpoint wrappers, not ambient hooks.

Key generated files:

- `gen/<service>/service_interceptors.go`
- `gen/<service>/client_interceptors.go`
- `gen/<service>/interceptor_wrappers.go`
- `gen/<service>/endpoints.go`
- `gen/<service>/client.go`

## Contracts

- Server interceptors wrap typed service endpoints after decoding and before the service method.
- Client interceptors wrap typed client endpoints before encoding and after decoding.
- Accessors only expose fields declared with `Read...` or `Write...` DSL.

## Ordering

The generated `Wrap<Method>Endpoint` is the source of truth.

- Last wrapper runs first on the request path.
- First wrapper runs first on the response path.

Read the generated wrapper if ordering matters.

## Implementation Pattern

```go
func (i *Interceptors) RequestAudit(ctx context.Context, info *RequestAuditInfo, next goa.Endpoint) (any, error) {
    res, err := next(ctx, info.RawPayload())
    if err != nil {
        return nil, err
    }
    r := info.Result(res)
    r.SetDuration(123)
    return res, nil
}
```
