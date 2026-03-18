# DSL: Data Modeling

Use this for Goa type modeling, validations, formats, and examples.

## Core Types

```go
Boolean Int Int32 Int64 UInt UInt32 UInt64
Float32 Float64 String Bytes Any
```

## Structured Types

```go
var Person = Type("Person", func() {
    Attribute("name", String)
    Attribute("age", Int32, func() {
        Minimum(0)
        Maximum(120)
    })
    Required("name", "age")
})
```

## Collections And Composition

- `ArrayOf(T)` for ordered collections
- `MapOf(K, V)` for typed maps
- `Reference(Type)` to reuse attribute definitions selectively
- `Extend(Type)` to inherit all attributes

```go
var Employee = Type("Employee", func() {
    Reference(Person)
    Attribute("name")
    Attribute("age")
    Attribute("employeeID", String, func() {
        Format(FormatUUID)
    })
})
```

## Validation

- Strings: `Pattern`, `MinLength`, `MaxLength`, `Format`
- Numbers: `Minimum`, `Maximum`, `ExclusiveMinimum`, `ExclusiveMaximum`
- Objects: `Required`
- Generic: `Enum`

## Formats

- `FormatDate`, `FormatDateTime`, `FormatUUID`, `FormatEmail`
- `FormatHostname`, `FormatIPv4`, `FormatIPv6`, `FormatIP`
- `FormatURI`, `FormatMAC`, `FormatCIDR`, `FormatRegexp`, `FormatJSON`, `FormatRFC1123`

## `Attribute` vs `Field`

- Use `Attribute` for HTTP-only types.
- Use `Field(number, ...)` when gRPC support or stable protobuf numbering matters.

## Examples

```go
Attribute("email", String, func() {
    Example("work", "john@work.com")
    Format(FormatEmail)
})
```

Use `Randomizer(...)` at API level if deterministic examples matter.
