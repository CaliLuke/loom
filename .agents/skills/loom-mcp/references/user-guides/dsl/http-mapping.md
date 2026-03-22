# DSL: HTTP Mapping

Use this for `HTTP(...)` mappings, params, body/header rules, and static files.

## Basic Mapping

```go
Method("show", func() {
    Payload(func() {
        Field(1, "id", String)
        Required("id")
    })
    HTTP(func() {
        GET("/{id}")
    })
})
```

## Common Tools

- `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, `TRACE`
- `Param(...)` for query or path binding
- `Header(...)` for header binding
- `Body(...)` for explicit body mapping
- `Path(...)` for API/service prefixes
- `Parent(...)` and `CanonicalMethod(...)` for nested resources

## Mapping Rules

- For primitive/array/map payloads, Goa maps from the first relevant HTTP element you declare.
- For object payloads, path params map from the route and remaining fields usually come from the body.
- Use `Body("field")` when one payload field should be the whole request body.
- Use `Header("field:X-Header-Name")` or `Attribute("field:wire_name")` to rename transport fields.

## Files

Static file serving is HTTP-only:

```go
var _ = Service("web", func() {
    Files("/assets/{*path}", "./public/assets")
    Files("/favicon.ico", "./public/favicon.ico")
    Files("/{*path}", "./public/index.html")
})
```
