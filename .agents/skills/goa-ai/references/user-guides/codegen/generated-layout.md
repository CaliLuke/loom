# Codegen: Generated Layout

Use this for understanding what Goa generates and where to implement custom logic.

## Typical Layout

```text
myservice/
├── cmd/
├── design/
├── gen/
│   ├── <service>/
│   ├── http/
│   └── grpc/
└── <service>.go
```

## Important Areas

- `gen/<service>/service.go`: service interface, payload/result types, constants
- `gen/<service>/endpoints.go`: transport-agnostic endpoint wrappers
- `gen/<service>/client.go`: typed client over endpoints
- `gen/http/<service>/...`: HTTP server/client adapters
- `gen/grpc/<service>/...`: gRPC server/client/protobuf adapters

## Ownership Rule

- Do not edit `gen/`.
- Put business logic in non-generated service files.
- Treat `cmd/` example code as yours after scaffolding.
