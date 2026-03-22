# HTTP: Routing

Use this for routes, prefixes, params, wildcards, and parent-child service paths.

## Base Path

```go
var _ = Service("calculator", func() {
    HTTP(func() {
        Path("/calculator")
    })
})
```

## Parameters

- Path params: `GET("/users/{user_id}")`
- Renamed path params: `GET("/users/{user_id:id}")`
- Query params: `Param("page")`

## Wildcards

```go
GET("/files/*path")
```

## Parent Services

```go
var _ = Service("posts", func() {
    Parent("users")
    Method("list", func() {
        HTTP(func() { GET("/posts") })
    })
})
```

## Prefix Hierarchy

- API-level `HTTP(Path(...))` adds a global prefix
- Service-level `HTTP(Path(...))` adds a service prefix
- Method routes append to both
