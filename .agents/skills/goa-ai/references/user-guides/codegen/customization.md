# Codegen: Customization

Use this for Goa metadata that changes generated types, packages, protobuf, or OpenAPI output.

## Common Metadata Knobs

- `Meta("type:generate:force", ...)`
- `Meta("struct:pkg:path", "types")`
- `Meta("struct:field:name", "ID")`
- `Meta("struct:tag:json", "id,omitempty")`
- `Meta("struct:field:type", "...")`
- `Meta("struct:name:proto", "...")`
- `Meta("struct:field:proto", "...")`
- `Meta("protoc:include", "...")`
- `Meta("openapi:generate", "false")`
- `Meta("openapi:summary", "...")`
- `Meta("openapi:operationId", "{service}.{method}")`

## Example

```go
var CommonType = Type("CommonType", func() {
    Meta("struct:pkg:path", "types")
    Attribute("id", String, func() {
        Meta("struct:field:name", "ID")
    })
})
```

## When To Use

- Shared package layout requirements
- Protobuf customization
- OpenAPI naming or extension needs
- Forcing generation of otherwise unreferenced types
