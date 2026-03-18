# HTTP: Content Negotiation, CORS, Static Files

Use this for custom encoders, CORS rules, and file serving.

## Content Negotiation

Built-in encoders commonly cover:

- JSON
- XML
- Gob
- HTML
- plain text

Selection order:

1. `Accept`
2. request `Content-Type` when `Accept` is absent
3. default encoder, usually JSON

## Custom Encoders

Wire custom encoders/decoders in server setup, not the DSL.

## CORS

Use the CORS plugin:

```go
import (
    cors "goa.design/plugins/v3/cors/dsl"
    . "goa.design/goa/v3/dsl"
)
```

Prefer explicit origins over `*` for authenticated APIs.

## Static Files

```go
Files("/static/{*path}", "./public")
Files("/favicon.ico", "./public/favicon.ico")
Files("/{*path}", "./dist/index.html")
```

`Files(...)` is HTTP-only.
