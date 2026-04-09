# How are default values evaluated in protocol buffers?

Non-nil default values are not supported in protocol buffers
(see https://developers.google.com/protocol-buffers/docs/proto3#default).
Hence, there is no way to figure out whether a field was explicitly set to
the default value or just not set at all. So Loom does not initialize such
fields with their default values.

# How does Loom deal with nested maps and arrays in protocol buffers?

proto3 syntax for protocol buffer does not support nested maps and arrays
(see https://github.com/protocolbuffers/protobuf/issues/4596). In such cases,
Loom wraps the inner map/array into a user type having a single attribute named
"field" with RPC tag number 1.

Example:

Type definition
```
Type("MyType", func() {
  Field(3, "nested", MapOf(Int, MapOf(String, ArrayOf(Bool))))
})
```
is transformed into protocol buffer message below
```
message MyType {
  map<int32, MapOfStringArrayOfBool> nested = 3;
}

message MapOfStringArrayOfBool {
  map<string, ArrayOfBool> field = 1;
}

message ArrayOfBool {
  repeated bool field = 1;
}
```
for which protoc generates the following types
```
type MyType struct {
  Nested map[int32]*MapOfStringArrayOfBool
}

type MapOfStringArrayOfBool struct {
  Field map[string]*ArrayOfBool
}

type ArrayOfBool struct {
  Field []bool
}
```

# How does Loom handle the Any type in gRPC?

Loom supports the `Any` type in gRPC by mapping it to `google.protobuf.Value`, which is specifically designed to represent dynamic JSON-like values. This is simpler and more efficient than using `google.protobuf.Any`.

## Conversion Process

- **Go to Protobuf**: When converting from Go `any` to `*structpb.Value`, Loom uses `structpb.NewValue()` which directly converts Go types to protobuf Value.
- **Protobuf to Go**: When converting from `*structpb.Value` to Go `any`, Loom uses the `AsInterface()` method which returns the corresponding Go value.

## Example Usage

In your Loom design:
```go
Method("echo", func() {
    Payload(func() {
        Field(1, "data", Any, "Any type of data")
    })
    Result(func() {
        Field(1, "data", Any, "Echoed data")
    })
    GRPC(func() {
        Response(CodeOK)
    })
})
```

This generates the following protobuf:
```proto
import "google/protobuf/struct.proto";

message EchoRequest {
    optional google.protobuf.Value data = 1;
}

message EchoResponse {
    optional google.protobuf.Value data = 1;
}
```

## Supported Patterns

- Direct Any fields: `Field(1, "data", Any)`
- Maps with Any values: `MapOf(String, Any)`
- Arrays of Any: `ArrayOf(Any)`
- Nested structures containing Any

## Supported Value Types

The `google.protobuf.Value` type natively supports:
- Null values
- Numbers (integer and floating point)
- Strings
- Booleans
- Structs (maps)
- Lists (arrays)

## Limitations

- Complex Go types (channels, functions, custom structs) need to be JSON-serializable
- Type information is abstracted to basic JSON types
- Precision may be lost for very large integers (uses float64 internally)
