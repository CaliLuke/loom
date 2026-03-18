# gRPC: Service Design

Use this for Goa gRPC service structure, type mapping, field numbering, and metadata/message mapping.

## Service Skeleton

```go
var _ = Service("calculator", func() {
    GRPC(func() {
        Meta("package", "calculator.v1")
        Meta("go.package", "calculatorpb")
    })
})
```

## Type Mapping

- `Int`/`Int32` -> `int32`
- `Int64` -> `int64`
- `UInt`/`UInt32` -> `uint32`
- `UInt64` -> `uint64`
- `Float32` -> `float`
- `Float64` -> `double`
- `String` -> `string`
- `Boolean` -> `bool`
- `Bytes` -> `bytes`

## Field Numbering

- `1-15` for frequent fields
- `16-2047` for less frequent fields

## Metadata And Message

Use `Metadata(...)` to send fields as gRPC metadata and `Message(...)` to control which fields land in the protobuf message body.
