---
title: HTTP Guide
weight: 4
description: "Complete guide to HTTP transport in Loom - routing, content negotiation, WebSocket, SSE, and static content."
llm_optimized: true
aliases:
---

This guide covers HTTP-specific features in Loom, from basic routing to advanced topics like request body encoding, WebSocket streaming, SSE, and content negotiation.

## HTTP Routing

### Basic Routing

Routes are defined using the `HTTP` function within a Service:

```go
var _ = Service("calculator", func() {
    HTTP(func() {
        Path("/calculator")  // Base path for all endpoints
    })

    Method("add", func() {
        Payload(func() {
            Field(1, "a", Int, "First operand")
            Field(2, "b", Int, "Second operand")
        })
        Result(Int)
        HTTP(func() {
            POST("/add")  // POST /calculator/add
        })
    })
})
```

Loom supports all standard HTTP methods: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`, `TRACE`.

A single method can handle multiple HTTP methods or paths:

```go
Method("manage_user", func() {
    Payload(User)
    Result(User)
    HTTP(func() {
        POST("/users")          // Create
        PUT("/users/{user_id}") // Update
        Response(StatusOK)
        Response(StatusCreated)
    })
})
```

### Path Parameters

Capture dynamic values from the URL:

```go
Method("get_user", func() {
    Payload(func() {
        Field(1, "user_id", String, "User ID")
    })
    Result(User)
    HTTP(func() {
        GET("/users/{user_id}")  // Maps to payload.UserID
    })
})
```

Map URL parameter names to payload field names:

```go
Method("get_user", func() {
    Payload(func() {
        Field(1, "id", Int, "User ID")
    })
    HTTP(func() {
        GET("/users/{user_id}")  // URL uses user_id
        Param("id:user_id")      // maps user_id to payload.ID
    })
})
```

### Query Parameters

Define query parameters with the `Param` function:

```go
Method("list_users", func() {
    Payload(func() {
        Field(1, "page", Int, "Page number", func() {
            Default(1)
            Minimum(1)
        })
        Field(2, "per_page", Int, "Items per page", func() {
            Default(20)
            Minimum(1)
            Maximum(100)
        })
    })
    Result(CollectionOf(User))
    HTTP(func() {
        GET("/users")
        Param("page")
        Param("per_page")
    })
})
```

Optional string query parameters use pointers in generated service payloads.
The pointer states have these meanings:

| Generated value | HTTP query | Meaning |
|---|---|---|
| `nil` | key omitted | The parameter is absent. |
| `ptr("")` | `track_visit=` | The parameter is present and empty. |
| `ptr("yes")` | `track_visit=yes` | The parameter is present and nonempty. |

Omit `Required("track_visit")` to get this generated representation. Map the
attribute with the normal `Param` function. No additional transport DSL is
necessary:

```go
Payload(func() {
    Attribute("track_visit", String)
})
HTTP(func() {
    GET("/resource")
    Param("track_visit")
})
```

Generated clients omit a nil pointer. They emit the query key for every nonnil
pointer, including a pointer to an empty string. A default applies only when
the query key is absent.

Generated servers parse query strings strictly. A malformed query string returns
a `decode_payload` response with status 400. Percent-encode reserved delimiters,
such as `;`, in query values. Generated clients apply this encoding. Repeated
query keys retain their order.

Named query maps use one JSON document per parameter value. For example,
`Param("filter")` with `MapOf(String, String)` sends
`filter=%7B%22status%22%3A%22active%22%7D`, whose decoded value is
`{"status":"active"}`. Nested maps and arrays inside the map use the same JSON
representation. Clients in any language should JSON-encode the map, then pass
that string to their URL query encoder. Scalar parameters and repeated scalar
arrays keep their ordinary query representation.

Regenerate clients and servers together when migrating from bracketed map keys
such as `filter[status]=active`; those keys no longer populate the named map.
`MapParams()` remains an unprefixed whole-query map, for example
`status=active&region=west`. Its contract differs from a named `Param("filter")`.
OpenAPI describes named maps with parameter `content: application/json` and a
typed schema. Fixed object query parameters remain unsupported.

### Wildcards

Each path parameter must start a segment and use a name containing only ASCII
letters, digits, or underscores. Use separate segments for separate parameters,
such as `/items/{prefix}/{number}`. A static suffix such as `/{name}.json` is
supported. Composite segments such as `/{prefix}-{number}`, prefixed parameters
such as `/item-{name}`, and unmatched braces are rejected during DSL validation.
Move composite values into a single parameter, or use separate path segments,
then regenerate the service.

Catch-all parameters use `{*name}` and must terminate the path.
Capture all remaining path segments:

```go
Method("serve_files", func() {
    Payload(func() {
        Field(1, "path", String, "Path to file")
    })
    HTTP(func() {
        GET("/files/{*path}")  // Matches /files/docs/image.png
    })
})
```

### Path Value Escaping

The generated path builders, such as `ShowItemsPath`, return escaped URL paths.
They escape each path value as one segment, so a value such as `a/b` stays in
its segment as `a%2Fb`. They also escape the dots of a `.` or `..` value. A
catch-all value keeps its `/` separators, and each of its segments is escaped.
Numbers and booleans are not escaped. Generated clients send the escaped path
unchanged, and the router returned by `loomhttp.NewMuxer` decodes each value.
Services receive the original values. Do not add `url.PathEscape` or
`url.PathUnescape` calls in the application.

A path builder result is already escaped. Do not set it as `url.URL.Path`,
which escapes it again, so `a b` becomes `a%2520b`. Build the URL with
`loomhttp.RequestURL(scheme, host, path)`, or set the result as `RawPath` and
its unescaped form as `Path`.

The router matches route literals and request paths in the same escaped form.
Literals containing spaces or non-ASCII characters work with escaped parameter
values, including an encoded `/`. Route metadata (`r.Pattern` and
`ResolvePattern`) retains the authored pattern.

Array path parameters use literal commas between escaped elements. For example,
`a%2Cb,c` decodes to `["a,b", "c"]`. The generated decoder splits the raw capture
before unescaping each element. A literal `%2C` value stays `%2C`; it is not
decoded twice. A `+` in a path remains a plus sign.

Custom implementations of `loomhttp.Muxer` must provide both `Vars` and
`RawVars`. `Vars` returns decoded captures; `RawVars` preserves their original
percent escapes and literal separators. Implement `RawVars` from the router's
escaped captures, not by escaping values returned by `Vars`: decoding has
already lost the distinction between `%2C` and a comma separator. Adding this
method is required when upgrading a custom mux. `loomhttp.NewMuxer()` provides
both methods.

### Service Relationships

Use `Parent` to establish service hierarchies:

```go
var _ = Service("users", func() {
    HTTP(func() {
        Path("/users/{user_id}")
        CanonicalMethod("get")  // Override default "show"
    })
    
    Method("get", func() {
        Payload(func() {
            Field(1, "user_id", String)
        })
        HTTP(func() {
            GET("")  // GET /users/{user_id}
        })
    })
})

var _ = Service("posts", func() {
    Parent("users")  // Inherit parent's path
    
    Method("list", func() {
        // user_id inherited from parent
        HTTP(func() {
            GET("/posts")  // GET /users/{user_id}/posts
        })
    })
})
```

### Path Prefix Hierarchy

Combine prefixes at API and service levels:

```go
var _ = API("myapi", func() {
    HTTP(func() {
        Path("/api")  // Global prefix
    })
})

var _ = Service("users", func() {
    HTTP(func() {
        Path("/v1/users")  // Service prefix
    })
    
    Method("show", func() {
        HTTP(func() {
            GET("/{id}")  // Final: /api/v1/users/{id}
        })
    })
})
```

---

## Request Bodies

By default, Loom generates JSON request encoding and decoding for payload
attributes that are not mapped to path parameters, query parameters, headers,
or cookies. HTTP endpoints can opt into other request body contracts when the
wire format is part of the API design.

Loom and generated transports use Go 1.27's `encoding/json/v2`. Decoding is
case-sensitive and rejects duplicate object names and invalid UTF-8. Generated
`Any` fields use `loom.JSONValue`, which aliases `jsontext.Value`. This
type retains unparsed JSON and preserves each number spelling. Use the same
type for custom JSON integrations. Do not use the legacy raw-message type.

### Form Requests

Use `FormRequest` for `application/x-www-form-urlencoded` endpoints:

```go
Method("token", func() {
    Payload(func() {
        Attribute("client_id", String)
        Attribute("grant_type", String)
        Attribute("code", String)
        Required("client_id", "grant_type", "code")
    })
    Result(Token)
    HTTP(func() {
        POST("/token")
        FormRequest()
        Response(StatusOK)
    })
})
```

Loom generates form encoding and decoding for object, map, and constructor
union payloads. A root `MapOf(String, T)` uses its keys as dynamic form field
names. Form fields use the payload attribute names unless the design uses HTTP
element name mapping. For constructor unions, scalar branches use the canonical
discriminator/value fields, while object branches are flattened onto standard
top-level form fields. This supports OAuth-style token requests such as
`grant_type=refresh_token` without app-local parser hooks.

A named object or map form field also holds one JSON document. For example,
`address={"street":"Main"}` and `labels={"region":"west"}` are the decoded
form values; the form encoder percent-encodes the JSON strings on the wire.
This applies at every nesting depth. JSON tags and custom JSON codecs determine
member names inside these values; form tags determine the outer field name.
OpenAPI marks these fields with `contentType: application/json`, without a
serialization style. Scalars and repeated scalar arrays retain their form
encoding. Update external clients that previously sent bracketed nested keys;
there is no alternate bracket decoder for named objects or maps.

When a dynamic form map must coexist with path, query, header, cookie, or
security attributes, put the map in an object payload and select it with
`Body(...)`:

```go
Payload(func() {
    Attribute("config", MapOf(String, String))
    Attribute("trace", String)
})
HTTP(func() {
    PATCH("/config")
    Header("trace:X-Trace")
    Body("config")
    FormRequest()
})
```

### Multipart Requests

Use `MultipartRequest` for `multipart/form-data` uploads:

```go
Method("upload", func() {
    Payload(func() {
        Attribute("project_id", String)
        Attribute("file", Bytes)
        Attribute("filename", String)
        Attribute("content_type", String)
        Attribute("label", String)
        Required("project_id", "file", "label")
    })
    HTTP(func() {
        POST("/projects/{project_id}/uploads")
        Param("project_id")
        MultipartRequest()
    })
})
```

For supported object payloads, Loom generates server-side multipart decoding.
`Bytes` attributes receive file part content. When a payload has one file part,
string attributes named `filename` and `content_type` are populated from the
uploaded part metadata. Non-file scalar fields flow through the generated
request-body constructor and normal validation path. Multipart request client
encoding still accepts a generated hook so applications can control how files
are read and written.

Unsupported body shapes, such as arrays of file bytes, use the generated custom
decoder callback. That callback owns body decoding and must construct the payload
on success. The generated adapter then decodes and validates mapped path, query,
header and cookie fields, followed by ordinary Basic credential checks. A failed
callback or field validation prevents the endpoint from running. Regenerate after
upgrading to pick up corrected custom adapters, including their shared credential
framing and custom text-scalar decoding.

`MultipartRequest` does not support constructor union payloads. Use form
requests for union-shaped OAuth-style payloads.

### Optional JSON and Form Bodies

Use `OptionalRequestBody` when an endpoint accepts either no body or a typed
JSON object body. Form requests may also use it with object or map bodies:

```go
Method("search", func() {
    Payload(func() {
        Attribute("query", String)
        Attribute("filters", SearchFilters)
    })
    Result(SearchResult)
    HTTP(func() {
        POST("/search")
        Body("filters")
        OptionalRequestBody()
    })
})
```

The optional body must be an object body, or a map when `FormRequest` is set.
A body built from the payload attributes cannot contain required attributes; a
body selected with `Body("name")` must come from an optional payload
attribute. It cannot be combined with `MultipartRequest` or raw request body
streaming. JSON decoders tolerate `io.EOF` only for endpoints that opt into
`OptionalRequestBody`; form decoders accept an empty form. Malformed input and
validation errors still fail normally.

An object or constructor union selected with `Body("name")` from an optional
payload attribute is optional with or without `OptionalRequestBody`. When the
attribute is nil, the generated client sends no request body, without a
`Content-Type` header. The generated server decodes an empty or
whitespace-only body, or an empty form, as a nil attribute. A present body is
decoded and validated normally, so the required fields of the object apply
only when the client sends a body. The built-in JSON decoder rejects a
`null` body with `decode_payload` unless the body allows null. Optionality
permits absence, not null. A JSON-RPC method that selects its params the same
way works alike: the client omits `params` for a nil attribute and the server
decodes absent `params` as a nil attribute. The server rejects non-nullable
`"params": null` with `-32602`.
An object value `{}` is decoded and validated normally; a union still needs
its discriminator.

A nullable attribute selected with `Body("name")`, such as a nullable object,
array, map, primitive, or union, keeps the absent, null, and concrete states
of its `loom.Nullable` payload field. When the attribute is optional, the
generated client sends no body for an absent value, `null` for a null value,
and the value otherwise. The generated server decodes an empty or
whitespace-only body as an absent value and `null` as a null value, and it
validates a concrete value: the fields of an object, the non-null elements of
an array, the values of a map, the selected branch of a union, and the
validations of a primitive. When the attribute is required, an empty body,
or absent JSON-RPC `params`, fails with `missing_payload`; a JSON-RPC server
answers `-32602`. An `Any` attribute works the same way, but its
`loom.JSONValue` payload field is nil when absent and holds `null` when null.
A required `Any` attribute has no absent state: the client sends a nil value
as `null`. A JSON-RPC client omits `params` only for an absent value; it
sends `"params": null` for a null value, which the server decodes as a null
value.

An optional primitive, array, map, or `Bytes` attribute selected with
`Body("name")` is optional the same way. When the attribute is nil, the
generated client sends no request body, and a JSON-RPC client omits
`params`. The generated server decodes an empty or whitespace-only body, or
absent JSON-RPC `params`, as a nil attribute, and validates a present body.
A JSON `null` body is rejected unless the body allows null, regardless of
whether it is required. An empty array or map is present: it is
sent and decoded as an empty value. A primitive with a default value is not a
pointer in the payload, so the client always sends it. If that attribute is
optional and non-nullable, an empty or whitespace-only body, or absent JSON-RPC `params`,
receives its declared default. An explicit zero, false, or empty string stays
unchanged. A required body must still be present.

The generated client CLI makes the body flag of an optional object, union,
nullable, `Any`, primitive, array, map, or `Bytes` attribute selected with
`Body("name")` optional. Without a declared default, an empty flag leaves the
attribute nil or absent.
For a defaulted primitive or collection, an omitted flag uses the declared
default; an explicit flag overrides it, including an empty array or map.
Collection flags accept JSON. Boolean map keys use the member names `"true"`
and `"false"`, including in nested collections. The flag example shows the JSON that the flag accepts, including the `type`
and `value` fields of a union.

HTTP and JSON-RPC CLI descriptions and displayed defaults are literal text.
Sequences such as `${name}` and `$$` remain visible as written; no template
escaping is needed in the design. Regenerate clients to apply this correction.
Payload defaults and explicit flag values retain their authored text.

HTTP and JSON-RPC CLI JSON body flags validate member presence before constructing
the service payload. An omitted required member is rejected even when it declares
a default; an explicitly supplied `false`, `0`, or empty string remains supplied.
Defaults fill omitted optional members, including members of a supplied nested
object. An omitted outer object stays absent. Non-nullable bodies and members
reject JSON `null`; nullable values retain it. Custom JSON codecs keep their
representation. Regenerate clients to adopt this correction; flag syntax and
ordinary client wire types do not change.

HTTP JSON bodies use the same boolean map-key representation, including maps
inside tagged or untagged unions and optional or nullable fields. Only the
member names `"true"` and `"false"` are accepted for boolean keys; boolean values
remain JSON booleans. Custom JSON and text codecs keep their authored
representation. For custom JSON integration, pass `loom.JSONOptions()` to
`json.Marshal` or `json.Unmarshal` to use the same map-key rules. Add
`json.Deterministic(true)` when stable member ordering is required.

### Bytes in JSON Bodies

`Bytes` values in JSON bodies are base64 strings. OpenAPI schema examples,
enums, and defaults use the same JSON representation. Raw binary body media
examples contain the literal bytes when they form valid UTF-8 text; Loom omits
inline media examples for other byte sequences. Explicit `dataValue` and
`serializedValue` example metadata keep their respective data and serialized
representations.

For built-in JSON encoding, a `Bytes` schema describes a padded base64 string.
`MinLength` and `MaxLength` count decoded bytes, not base64 characters. Loom
intersects length bounds inherited through named aliases and emits standard
JSON Schema constraints for the base64 grammar and effective decoded length.
`contentEncoding: base64` also identifies the encoding; that annotation alone
does not validate the value. Requiredness, nullability and enum constraints
remain separate.

This schema projection applies when the representation uses built-in JSON.
Raw and text bodies, form and multipart fields, custom codecs, and explicit
schema overrides retain their own contracts.

### Raw Request and Response Bodies

Use `SkipRequestBodyEncodeDecode` when the service should receive the request
body as an `io.ReadCloser`, for example large uploads or pass-through
endpoints. Generated clients accept the caller-supplied `io.ReadCloser`. All
non-body payload attributes must be mapped to path parameters, query
parameters, or headers.

Add `OpenAPIRequestBody` when contract consumers also need to know the raw body
shape. The schema, media type, description, and requiredness affect OpenAPI
only; Loom still passes the original request stream directly to the service and
the generated client still sends the caller-supplied stream without encoding
or buffering it.

```go
var _ = Service("uploads", func() {
    Method("upload", func() {
        Payload(func() {
            Field(1, "id", String, "Upload identifier")
            Field(2, "checksum", String, "Expected SHA-256 digest")
            Required("id", "checksum")
        })

        HTTP(func() {
            POST("/uploads/{id}")
            Header("checksum:X-Checksum")
            SkipRequestBodyEncodeDecode()
            OpenAPIRequestBody(Bytes, "application/octet-stream", true, func() {
                Description("Archive bytes streamed directly to the service.")
            })
        })
    })
})
```

Use `OpenAPIRequestBodyTypes` when one raw schema accepts multiple media types.
The service must inspect the request content type and decode the stream.

```go
OpenAPIRequestBodyTypes(Upload, []string{
    "application/json",
    "application/x-www-form-urlencoded",
    "multipart/form-data",
}, true)
```

Use `String` with a text media type for raw text, or a named Loom type when the
stream has a structured schema that should appear as an OpenAPI component. The
optional DSL function accepts normal schema documentation such as
`Description`, `Example`, `Format`, and `Meta`. Both documentation expressions
require `SkipRequestBodyEncodeDecode` and cannot be combined with `Body`,
`FormRequest`, `MultipartRequest`, or `OptionalRequestBody`.

Use `SkipResponseBodyEncodeDecode` when the service returns a raw response body
reader. Set `ContentType` in the response to declare the media type. The
generated handler sets this static header before it streams the body.
`OpenAPIBody` describes the response schema but does not enable response
encoding. Raw body modes are HTTP-only. You cannot combine them with gRPC or
streaming payload or result methods.

Use `FileResponse` for seekable downloads that need standard range and
conditional request behavior. It is HTTP-only and accepts explicit `GET` and
`HEAD` routes; Loom does not synthesize a `HEAD` route.

```go
Method("download", func() {
    Payload(func() {
        Attribute("id", String)
        Required("id")
    })
    Result(func() {
        Attribute("etag", String)
    })
    HTTP(func() {
        GET("/downloads/{id}")
        HEAD("/downloads/{id}")
        FileResponse()
        Response(func() {
            ContentType("application/pdf")
            Header("etag:ETag")
        })
    })
})
```

The generated service method returns the modeled result metadata followed by
`*loomhttp.FileResponse` and `error`. Set `Name`, `ModTime`, and seekable
`Content`; the generated handler encodes result headers first, then delegates
status, content length, ranges, cache validators, and GET/HEAD body behavior to
`http.ServeContent`. It closes `Content` after the request when it also
implements `io.Closer`. Generated clients return the modeled result plus an
`io.ReadCloser`; callers own closing that response body.

OpenAPI documents the normal 200 response and the ServeContent-owned 206,
304, 412, and 416 outcomes, together with Range and conditional request
headers. Application responses must keep one untagged 200 success and must not
claim those protocol statuses. Without an explicit response `ContentType`, the
spec uses binary `*/*` because `FileResponse.Name` controls runtime MIME
inference. Result metadata may map ETag and application headers, but not the
transport-owned Content-Type, Content-Length, Content-Range, Accept-Ranges, or
Last-Modified headers. Every result metadata attribute must map to a response
header or cookie; FileResponse exclusively owns the response body.
Generated response-contract manifests cover declared application response
branches, not FileResponse's transport-owned 206, 304, 412, and 416 outcomes
or decoder and mux failures such as 400 and 413.

For an ordinary unary GET endpoint, mount an opt-in derived companion after
mounting the generated server:

```go
catalogsvr.Mount(mux, server)
loomhttp.MountDerivedHead(mux, "/catalog/{id}", server.Show)
```

The companion executes the GET handler so authentication, cookies, headers,
and application effects stay aligned, counts the encoded representation to set
`Content-Length`, and suppresses the body. Use an explicitly designed `HEAD`
route with `FileResponse` for downloads. Do not use derived HEAD for streaming
handlers.

---

## Debug Client Capture

`loomhttp.NewDebugDoer` is intended for local CLI diagnostics. Generated HTTP
and JSON-RPC example clients use it when debug mode is enabled. Each request and
response body capture is limited to 64 KiB; the full body still flows to the
server or caller, while an oversized captured body is omitted from debug
output.

Response capture is passive: `Do` returns without reading response bytes, and
`Fprint` prints only the latest exchange snapshot. The automatic request/response
summary is written once when the caller observes EOF or calls `Close`; nil and
`http.NoBody` responses print immediately. Keep closing response bodies normally.
An unread, unclosed response produces no automatic summary. A snapshot taken
before EOF is marked incomplete; incomplete JSON/form previews are omitted.
Concurrent Close may print an incomplete snapshot before an in-flight Read returns.
It does not wait for or drain that read. Original read bytes/errors and every
Close call/error reach the caller unchanged. Sequential reuse retains each open
response's own diagnostic identity; concurrent `Do` calls are not supported.
Client request capture retains its bounded prefix/GetBody behavior.

Client and server Debug share credential redaction. Conventional names include
`apiKey`, `accessToken`, `clientSecret`, `key`, and their separated spellings;
authorization and cookie headers are wholly masked. Structured JSON fields are
masked at any nesting depth, and URL/form parameters include repeated values.
Malformed or incomplete structured previews are omitted.

Regenerate after changing security or wire mappings. Credential-bearing services
expose `Fields()` in `gen/http/<service>/diagnostics.go` (or `gen/jsonrpc/<service>`).
Generated example clients supply this metadata automatically. Handwritten clients
and servers pass it when enabling Debug:

```go
import serviceDiagnostics "example.com/app/gen/http/widgets"

debugClient := loomhttp.NewDebugDoer(client, serviceDiagnostics.Fields())
debugServer := middleware.Debug(mux, writer, serviceDiagnostics.Fields())
```

For handwritten transports, supply `diagnostics.Fields` from
`github.com/CaliLuke/loom/observability/diagnostics`. Constructors copy these fields;
subsequent caller changes cannot alter a delayed response preview. Fields describe
actual header, query, JSON, and form wire names. Policies conservatively combine
selected services: the same name is masked even where another method uses it for
ordinary data. A scalar credential body is omitted completely. A credential path
omits the whole URL path and separately logged route variables. Known credential
bodies with unsupported content types are also omitted.

Application headers, URLs, and wire bodies remain unchanged. Ordinary opaque
bodies remain visible when no body credential policy applies, so use Debug only
for development and avoid payloads containing unmodeled secrets. Access logging uses the same URL masking policy: `log.HTTP` and `log.Client`
mask conventional query credentials and URL userinfo automatically. Ordinary
query values remain visible. Pass generated metadata for custom names or credential
paths:

```go
handler := log.HTTP(ctx, log.WithURLCredentials(serviceDiagnostics.Fields()))(mux)
transport := log.Client(http.DefaultTransport,
    log.WithClientURLCredentials(serviceDiagnostics.Fields()))
```

These options snapshot fields when constructed, can be reused, and combine when
supplied more than once. Only URL query and path locations affect access logs;
this does not configure response-body logging. Generated server examples supply
metadata for the HTTP and JSON-RPC services hosted by that server. `loom example`
leaves existing application files unchanged, so update existing access-log wiring
as shown above after regenerating metadata.

---

## Debug Server Capture

`middleware.Debug` preserves the response writer's streaming, flushing and
connection-control capabilities. It logs the committed response status and bytes
accepted by the writer, including partial writes that return an error.
Response previews use headers captured around the first observed commit. Later
handler changes cannot select a different redaction parser. If forwarding changes
those headers (including implicit content-type detection), or an opaque ReadFrom
commits first, Debug prints `[body omitted: response metadata unavailable]`.
Explicit, stable response headers preserve ordinary previews. Application bytes
and writer operations are unchanged.

Each request or response body preview is limited to 64 KiB. Larger previews are
omitted entirely with `[body omitted after 65536 bytes]`; application I/O is not
limited. SSE events flush normally before the handler returns. The debug summary
is written when the handler returns, with the request before the response.

Request capture observes downstream reads without reading ahead, draining unread
input, replacing read errors or taking ownership of Close. Partial bytes and the
original errors reach the reader unchanged. When EOF has not been observed, the
preview includes `[request body capture stopped before EOF]`, even if the number
of bytes matches Content-Length. Nil bodies and `http.NoBody` are complete empty
bodies. If Debug wraps a request-size policy, the preview also observes bytes
consumed by that policy, including its one-byte overflow probe. The policy's
rejection and limit remain unchanged in either middleware order.

---

## Content Negotiation

### Built-in Encoders

Loom's default encoders support:
- JSON (`application/json`, `*+json`)
- XML (`application/xml`, `*+xml`)
- Gob (`application/gob`, `*+gob`)
- HTML (`text/html`)
- Plain text (`text/plain`)

An explicit response `ContentType` in the design takes precedence over the
request. Otherwise the default response encoder selects a single supported
media type from `Accept`, ignoring media-type parameters such as `charset`.
Missing or unsupported values fall back to JSON. The built-in negotiator does
not rank comma-separated alternatives, quality values, or wildcards; install a
custom encoder when the API needs to select among several representations.

A mapped response `Content-Type` header is written after encoder selection;
its value alone does not select the encoder. Use the response `ContentType`
declaration to select a fixed built-in encoding. A default JSON response schema
describes the default negotiation path, not every possible `Accept` value.

When an endpoint produces one known set of media types and must reject an
incompatible `Accept` header, apply a strict policy to its generated handler:

```go
negotiation, err := loomhttp.NewResponseNegotiationPolicy("application/json")
if err != nil {
    return err
}
server.Show = negotiation.Handler(server.Show)
```

The policy understands comma-separated alternatives, quality values, and
wildcards. It adds `Vary: Accept` and returns Loom's standard RFC 9457 `406`
problem with code `not_acceptable` when every supported representation is
excluded. The default encoder remains permissive when no policy is installed.

Request decoding uses the request `Content-Type`, not `Accept`. A missing
request content type defaults to JSON. Unsupported request media types produce
an unsupported-media-type error rather than silently decoding as JSON.

Set a default response content type:

```go
Method("create", func() {
    HTTP(func() {
        POST("/media")
        Response(StatusCreated, func() {
            ContentType("application/json")
        })
    })
})
```

### Body Limits

Loom's built-in JSON, XML, Gob, HTML, and plain-text request decoders accept at
most 32 MiB. Generated multipart decoding, including custom decoder callbacks,
limits the complete encoded request to 32 MiB: part headers, boundaries, unnamed
parts and epilogue all count. Before a successful callback can invoke the
service, Loom consumes any remaining request bytes within that same limit.
Callback failures do not trigger extra reads; the server retains body-close
ownership. A larger configured policy replaces the default.
Requests that exceed either limit receive an RFC 9457
`application/problem+json` response with status `413` and code
`request_too_large`, rather than the `decode_payload` response used for
malformed bodies.

Regenerate servers after upgrading. Direct runtime callers now pass the original
`*http.Request` to `ReadMultipartForm` or `ReadMultipartFormWithLimit`, rather than
calling `MultipartReader` first. `ReadMultipartForm` honors `RequestBodyPolicy`.
Generated custom callback signatures are unchanged; callbacks must finish their
payload decoding before returning and must not retain the parser or a live part.

JSON-RPC endpoints keep the protocol's HTTP `200` response and map the same
condition to an `InvalidRequest` error: code `-32600`, null `id`, message
`request body too large`, and error data name `request_too_large`. A
POST-initiated SSE request receives that envelope as a `message` event.

Generated HTTP and JSON-RPC clients cap buffered response-body restoration at
32 MiB. The decoder owns consumed bodies: it closes the original once, including
on decode/read failure, and preserves close failures alongside other errors for
`errors.Is`. Restoration retains the captured bytes after decoding, including a
partial prefix on read failure; oversized bodies are rejected without restoration.
A cleanup failure returns no successful result.

Successful raw-body and file-response methods transfer the original open body to
the caller, who must close it. `RestoreResponseBody` does not buffer or replace
those streams. Failed responses remain decoder-owned. JSON-RPC notification
responses use the same bounded consumption and cleanup operation. Regenerate
clients to adopt this ownership correction.

Unexpected response bodies included in generated client errors are capped at
64 KiB so an invalid upstream response cannot create an unbounded diagnostic.
The debug capture limit described above is a separate 64 KiB cap and does not
truncate the body delivered to the transport caller.

Use a validated policy when an endpoint needs a deployment-specific limit:

```go
bodyPolicy, err := loomhttp.NewRequestBodyPolicy(2 << 20)
if err != nil {
    return err
}
server.Upload = bodyPolicy.Handler(server.Upload)
```

The policy applies before JSON, text, form, multipart, custom, or raw-body code
reads the request. Generated decoders retain the standard `413` response. A raw
body service receives the same `request_too_large` service error from `Body`
and should return it normally. A larger configured limit also replaces the
default for generated buffered decoders.

`SkipResponseBodyEncodeDecode` deliberately bypasses response buffering because
the application owns the raw response stream.

### Runtime Response Cookie Policy

Keep cookie values and portable security defaults in the design. When the
deployment supplies domain, Secure mode, or expiry, wrap
the generated method with a response-cookie policy:

```go
cookiePolicy := loomhttp.ResponseCookiePolicy(func(_ context.Context, cookie *http.Cookie) error {
    if cookie.Name != "session" {
        return nil
    }
    cookie.Domain = config.CookieDomain
    cookie.Secure = config.CookieSecure
    cookie.Expires = time.Now().Add(config.CookieLifetime)
    return nil
})
server.SignIn = cookiePolicy.Handler(server.SignIn)
```

Generated response encoders still own the cookie name, value, path, `Max-Age`,
`HttpOnly`, `SameSite`, and partitioning policy. Runtime policy cannot change
them. Loom validates the resulting cookie and preserves secure prefix and
`SameSite=None` requirements. A policy failure occurs before the response is
committed and follows the generated error formatter.

Generated server fields are intentionally public. Assign middleware to one
field, as above, when policy applies to one method; use `server.Use` only when
the policy applies to every method in the service.

### Custom Encoders

Create custom encoders for specialized formats:

```go
type MessagePackEncoder struct {
    w http.ResponseWriter
}

func (enc *MessagePackEncoder) Encode(v interface{}) error {
    enc.w.Header().Set("Content-Type", "application/msgpack")
    return msgpack.NewEncoder(enc.w).Encode(v)
}

func NewMessagePackEncoder(ctx context.Context, w http.ResponseWriter) loomhttp.Encoder {
    return &MessagePackEncoder{w: w}
}
```

Register custom encoders when creating the server:

```go
func main() {
    decoder := func(r *http.Request) loomhttp.Decoder {
        switch r.Header.Get("Content-Type") {
        case "application/msgpack":
            return NewMessagePackDecoder(r)
        default:
            return loomhttp.RequestDecoder(r)
        }
    }
    
    encoder := func(ctx context.Context, w http.ResponseWriter) loomhttp.Encoder {
        if accept := ctx.Value(loomhttp.AcceptTypeKey).(string); accept == "application/msgpack" {
            return NewMessagePackEncoder(ctx, w)
        }
        return loomhttp.ResponseEncoder(ctx, w)
    }
    
    server := myapi.NewServer(endpoints, mux, decoder, encoder, nil, nil)
}
```

### Response Contract Checks

For each supported unary, multipart, SSE, or WebSocket endpoint, the generated server package exposes a
`<Method>ResponseContractCases` function and collects them in the service-wide
`ResponseContractCases` manifest. Each case records the exact status, allowed
base media types, error name, and required response headers and cookies
declared by the design. SSE success cases also record the stream direction,
message type and encoding, SSE field mappings, required ID and event-type
fields, allowed projection event types, and terminal completion contract.
Declared errors that fail before the stream starts remain ordinary HTTP cases.
Supported multipart cases also record request content type, part names, part
media types, and required parts. WebSocket success cases record the `101`
handshake, upgrade headers, stream direction, message types, and terminal
behavior. Declared pre-upgrade errors remain ordinary HTTP cases.

After `loom gen`, create a consumer-owned provider scaffold:

```bash
loom test-scaffold example.com/myservice/design
```

Loom writes missing files under `internal/contracttest/` and never overwrites
them. Fill the unary, multipart, SSE, and WebSocket scenario maps with callbacks that send
real requests through the generated transport. Unary callbacks return the
response. Multipart callbacks receive an `loomhttp.MultipartRequestContract`
and return the response. SSE callbacks return an
`loomhttp.SSEResponseContractObservation` with the handshake response, parsed
events, and terminal read error. The test reports every missing case and calls
the matching validator, including for nil responses. WebSocket callbacks receive
an `loomhttp.WebSocketResponseContract`. They return an
`loomhttp.WebSocketResponseContractObservation` with the handshake, outbound
JSON messages, and terminal read error.

Loom validates transport-owned wire behavior, including `Loom-Error` for
declared errors. Application tests must still arrange the service state,
payload, and fake or fixture that reaches each case; Loom does not synthesize
domain behavior. Applications also own multipart codecs and request fixtures.
Because the scaffold reads the service-wide generated
manifest at runtime, a later design change fails the existing test until its
new response scenarios are supplied.

The current response-contract scaffold support matrix is:

| Transport shape | Scaffold status |
|---|---|
| Unary HTTP, including declared file-response success and error cases | Supported |
| Non-streaming multipart requests with flat primitive or bytes object fields | Supported |
| Multipart requests combined with SSE | Unsupported; generation emits a diagnostic |
| Multipart requests combined with WebSocket | Unsupported. Generation emits a diagnostic. |
| Other multipart request shapes | Unsupported; generation emits a diagnostic |
| Server SSE success and pre-stream declared errors | Supported |
| Mixed unary/SSE results | Unsupported; generation emits a diagnostic |
| Client or bidirectional SSE | Unsupported; generation emits a diagnostic |
| Plain HTTP WebSocket success and pre-upgrade declared errors | Supported |
| Mixed unary/WebSocket results or incomplete stream message shapes | Unsupported. Generation emits a diagnostic. |
| Raw request/response and redirects | Unsupported; generation emits a diagnostic |
| JSON-RPC and gRPC | No response-contract scaffold yet |

For file responses, the manifest covers the response declared by the design.
It does not add transport-owned conditional or range branches such as 206, 304,
412, or 416.

---

## WebSocket Integration

> **Design Recap**: Streaming is defined at the design level using `StreamingPayload` and `StreamingResult`. The DSL is transport-agnostic — the same design works for HTTP (WebSocket/SSE) and gRPC. See [DSL Reference: Streaming](dsl-reference.md#streaming) for design patterns. This section covers HTTP-specific WebSocket implementation.

WebSocket enables real-time, bidirectional communication. Loom implements WebSocket through its streaming DSL.

### Streaming Patterns

**Client-to-Server Streaming:**

```go
Method("listener", func() {
    StreamingPayload(func() {
        Field(1, "message", String, "Message content")
        Required("message")
    })
    HTTP(func() {
        GET("/listen")  // WebSocket endpoints must use GET
    })
})
```

**Server-to-Client Streaming:**

```go
Method("subscribe", func() {
    StreamingResult(func() {
        Field(1, "message", String, "Update content")
        Field(2, "action", String, "Action type")
        Field(3, "timestamp", String, "When it happened")
        Required("message", "action", "timestamp")
    })
    HTTP(func() {
        GET("/subscribe")
    })
})
```

**Bidirectional Streaming:**

```go
Method("echo", func() {
    StreamingPayload(func() {
        Field(1, "message", String, "Message to echo")
        Required("message")
    })
    StreamingResult(func() {
        Field(1, "message", String, "Echoed message")
        Required("message")
    })
    HTTP(func() {
        GET("/echo")
    })
})
```

### WebSocket Implementation

Generated WebSocket streams keep the public typed `Send`, `Recv`, `Close`,
and `*WithContext` methods, while the socket lifecycle is owned by
`loomhttp.WebSocketStream`. That runtime wrapper handles idempotent close,
context cancellation, one physical reader, bounded inbound admission, and
terminal close. It keeps detecting disconnects while a service is running or
waiting without calling `Recv`. A handler that returns releases its upgraded
connection: success sends close 1000, and error or panic sends 1011 with a generic
reason. A panic still propagates. Returning before upgrade leaves normal HTTP
error handling intact. Applications do not need a deferred close to make handler
completion safe.

`Close` is terminal. After a stream is closed, including a server stream
closed before its lazy upgrade, later `Send` and `Recv` calls return
`loomhttp.ErrWebSocketStreamClosed`, and a connection upgraded after the
close is closed immediately. Only one goroutine may call `Recv` at a time.

`Recv` separates connection failures from decode failures:

- A failed connection read (a close frame, an abrupt disconnect, or a read
  deadline expiry) is terminal. Later `Recv` calls return the same error
  without reading the socket again. The socket is released and the service
  context is canceled immediately. Already admitted messages remain FIFO for
  readers whose operation context is still live, followed by that terminal
  error. Return from your receive loop on a connection error.
- A context canceled during a read returns the context error and closes the
  stream, so later calls return `loomhttp.ErrWebSocketStreamClosed`. A
  context that is already done returns its error without reading, and a
  later call with a live context reads normally.
- A message that fails to decode returns the decode error, but the stream
  stays readable and the next `Recv` reads the next message.
- An empty or whitespace-only message returns `io.ErrUnexpectedEOF`.
- A client `Recv` returns `io.EOF` when the server closes with a normal
  closure (`1000`). A server `Recv` returns `io.EOF` when the client sends a
  JSON `null` message.

Messages are decoded with `encoding/json/v2`, as HTTP request bodies are.
Decoding is strict: data after the JSON value, duplicate object keys, and
invalid UTF-8 are rejected, and field names are matched case-sensitively.

Code that owns a raw `loomhttp.WebSocketStream` can call
`ReadMessage(ctx)` to read one message payload without decoding it. Every
error it returns is a connection, close, or context failure, and it follows
the same terminal-read rule.

Inbound HTTP and JSON-RPC WebSocket messages have a default 32 MiB limit,
matching `loomhttp.DefaultMaxRequestBodyBytes`. The limit includes expanded
bytes after decompression. A violation rejects the complete message, sends close
code 1009, releases the connection, and returns an error wrapping
`websocket.ErrReadLimit`. This applies to both clients and servers.

Set the policy once for each physical connection:

```go
config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{
    MaxMessageBytes:     8 << 20,
    MaxBufferedBytes:   16 << 20,
    MaxBufferedMessages: 128,
}, configure)
if err != nil {
    return err
}
```

The optional `configure` hook has type `loomhttp.ConnConfigureFunc`; pass `nil`
when no hook is needed. A zero `loomhttp.WebSocketConfig{}` selects 32 MiB per
message, 64 MiB total raw admission and 256 queued/currently decoding messages.
Zero fields in `WebSocketLimits` select defaults: 32 MiB per message, twice the
resolved message size for the total byte budget, and 256 messages. Negative
values reject. The message size cannot exceed MaxInt64/2 and the byte budget
must hold at least two maximum-size messages.

The runtime reserves one maximum-size message for its physical reader. Queued
messages and the raw message being decoded share the remaining byte budget and
the message-count limit. Empty messages count too. If either limit is exceeded,
Loom attempts close 1013, releases the socket, cancels the service context and
returns an error matching `loomhttp.ErrWebSocketBufferFull`. It never drops input
and continues. This protects connections whose service stops consuming; clients
that send too far ahead of a slow service must reconnect after rejection.

These are raw payload and queue-record bounds, not a total Go heap bound.
Decoded objects, compression/allocator overhead, and values retained by caller
code are outside the budget. `ReadJSON` holds its reservation through decoding;
`ReadMessage` transfers the bytes to its caller on return. One extra byte probes
an individual message for overflow. HTTP null input half-close remains distinct
from disconnection and permits `CloseAndRecv` to receive the final result.
Generated HTTP `NewConnConfigurer(config)` applies it to every streaming
endpoint; individual `<Method>Config` fields can select different policies.
JSON-RPC client and server constructors accept the config in the former raw
configure-function argument position. A shared JSON-RPC client keeps this
policy for its physical connection across calls. Regenerate transport code and
update constructor calls from `NewWebSocketConfig(maxBytes, hook)` to
`NewWebSocketConfig(WebSocketLimits{MaxMessageBytes: maxBytes}, hook)` when
adopting this API.

The runtime installs its framing limit before invoking the hook. A hook may
lower that limit; increasing it does not bypass the expanded-byte limit. If the
hook returns a different connection, it must release the input without
invalidating the returned connection and configure the replacement's framing
limit. Returning `nil` fails attachment and closes the input.

For direct runtime use, allocate `NewWebSocketStream(config)` and then call
`SetConn(ctx, conn, cancel)`, checking its error. The connection lifetime context
controls idle cancellation; it is not a per-message context. Attachment is the
sole configuration path and starts the reader after the hook. Configure reader
handlers and deadlines inside that hook, before the physical reader starts. Reattaching the same connection does nothing; attaching another while a
connection is owned rejects and closes the incoming connection. A connection
attached after terminal close is closed immediately without invoking the hook.
Raw runtime `Close` releases immediately. `CloseWithStatus(code, reason)` owns a
bounded close-control attempt and unconditional release; replace the old
`WriteClose` then `Close` sequence with it. `CloseWithFinalJSON(value)` uses one
second for writer acquisition, the final write and a best-effort wait for peer
termination, then always releases the socket. This lets a cooperating server
consume queued input and its end marker before the client disconnects. It does
not acknowledge service processing; a stalled peer or two simultaneous final
closers exhaust that same deadline. Successful writing followed by wait
expiration adds no close error. Generated HTTP client `Close` uses this terminal
operation with JSON null; use `CloseAndRecv` for input half-close with a final
response.

Generated HTTP and JSON-RPC server constructors accept an optional final
`loomhttp.StreamWritePolicy`. Constructing the policy validates the timeout;
each WebSocket write installs a fresh deadline and clears it afterward.

```go
writePolicy, err := loomhttp.NewStreamWritePolicy(5 * time.Second)
if err != nil {
    return err
}
srv := server.New(/* normal generated arguments */, writePolicy)
```

Server-side implementation:

```go
func (s *service) handleStream(ctx context.Context, stream Stream) error {
    connID := generateConnectionID()
    s.registerConnection(connID, stream)
    defer s.cleanupConnection(connID)

    errChan := make(chan error, 1)
    go func() {
        errChan <- s.handleIncoming(stream)
    }()

    select {
    case <-ctx.Done():
        return ctx.Err()
    case err := <-errChan:
        return err
    }
}
```

Connection management:

```go
type ConnectionManager struct {
    connections map[string]*ManagedConnection
    mu          sync.RWMutex
}

func (cm *ConnectionManager) AddConnection(id string, stream Stream) {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    cm.connections[id] = &ManagedConnection{
        ID:       id,
        Stream:   stream,
        LastPing: time.Now(),
    }
}
```

---

## Server-Sent Events

> **Design Recap**: SSE uses `StreamingResult` at the design level with `ServerSentEvents()` in the HTTP mapping. See [DSL Reference: Streaming](dsl-reference.md#streaming) for design patterns.

SSE provides one-way server-to-client streaming over HTTP. It's ideal for:
- Real-time notifications
- Live data feeds
- Progress updates
- Event streaming

Generated HTTP and JSON-RPC SSE clients return only events terminated by a
blank line. If the connection ends inside an event, they discard that unfinished
event and return `io.EOF` after delivering any preceding complete events. LF,
CR, and CRLF line endings have the same framing behavior.

### SSE Design

```go
var Event = Type("Event", func() {
    Attribute("message", String, "Message body")
    Attribute("timestamp", Int, "Unix timestamp")
    Required("message", "timestamp")
})

var _ = Service("sse", func() {
    Method("stream", func() {
        StreamingResult(Event)
        HTTP(func() {
            GET("/events/stream")
            ServerSentEvents()  // Use SSE instead of WebSocket
        })
    })
})
```

Customize SSE events:

```go
var Event = Type("Event", func() {
    Attribute("message", String, "Message body")
    Attribute("type", String, "Event type")
    Attribute("id", String, "Event ID")
    Attribute("retry", Int, "Reconnection delay in ms")
    Required("message", "type", "id")
})

Method("stream", func() {
    StreamingResult(Event)
    HTTP(func() {
        GET("/events/stream")
        ServerSentEvents(func() {
            SSEEventData("message")
            SSEEventType("type")
            SSEEventID("id")
            SSEEventRetry("retry")
        })
    })
})
```

### Event ID, Type, and Retry Fields

The attributes mapped by `SSEEventID`, `SSEEventType`, and `SSEEventRetry` may
be required, optional, or optional with a default value:

- The server writes `id:` and `event:` only for a non-empty value, and
  `retry:` only for a positive value. An optional attribute that is unset (nil)
  is omitted, and so is a pointer to an empty string or a non-positive retry.
  An empty `id:` field would reset the client's last event ID, but the server
  never writes one, so an empty id cannot reset it. An empty `event:` field
  means the default `message` type, the same as an absent field, and the
  generated client decodes it as unset.
- The generated client maps `id` to SSE's persistent last-event-ID buffer.
  After `id: c1`, subsequent events that omit `id:` still receive `c1`.
  An explicit empty `id:` from a peer resets the buffer. When the buffer is
  empty, an optional ID is nil, a defaulted ID receives its default, and a
  required ID receives the empty string. Use a payload attribute for an
  identifier that must describe only the current event.
- Event types are per-event: an optional `event` attribute is nil when no type
  is supplied, a defaulted attribute receives its default, and a required
  attribute receives the empty string.
- The generated client does not decode `retry:`. It is a reconnection hint for
  the SSE connection, not a value of the streamed result.

The persistent ID behavior follows the SSE library and replaces the old
per-event ID mapping. Keeping per-event presence would require a separate
library API; Loom uses the standard resume-cursor semantics instead.

### Event Data Encoding

The `data:` field of each event is encoded according to how the design maps it:

- **Whole-event results** (no `SSEEventData`): the data is always the JSON
  encoding of the streamed value, for every result type. Objects are JSON
  objects, `String` is a JSON string, `Bytes` is a base64 JSON string, and
  `Int`, `Boolean`, `Any`, arrays, and maps are their JSON values. Generated
  clients decode the data through the configured JSON decoder, so every value
  round-trips exactly, including empty strings, carriage returns, trailing
  newlines, and binary bytes. Browser clients should `JSON.parse(event.data)`.
- **Field-level data** (`SSEEventData("field")`): native string fields and
  `Bytes` fields emitted as native `[]byte` are written as raw text. Named byte
  types and nullable wrappers use JSON, as do objects and collections. Numbers
  and booleans appear as their JSON literal text. Raw text is subject to SSE framing: CR
  and CRLF become LF, a trailing line break is lost, an empty value can produce
  an event that clients do not dispatch, and bytes must be valid UTF-8 text. Map
  the data to a structured field, or use a whole-event result, when values can
  contain these characters.
- **Optional field-level data** (an optional attribute without a default):
  a `String` field is written as raw text when it is set. When it is nil, the
  event has no data. Generated clients set the field only when the data is not
  empty, so an empty string is received as nil. An optional number or boolean
  is written as its literal text, or `null` when it is nil, and generated
  clients decode `null` to nil.

For a viewed streaming result, the selected view determines the body of each
event, including collection elements. Dynamic streams use `SetView(name)`;
a design that fixes the view always uses that view. If the selected view omits
the field mapped to SSE data, Loom emits the codec's existing empty value: no
raw text for native strings/bytes, or JSON `null` for JSON-encoded fields.
Canonical requiredness or a default does not restore an omitted field.
Regenerate servers and clients after upgrading to apply this correction.

### Per-Event JSON Projections

Typed SSE streams may opt into multiple result-view projections. Map each SSE
`event:` discriminator to one view with `SSEProjection`; the server selects the
wire shape from the field named by `SSEEventType`, and generated clients decode
every declared view back into the canonical `StreamingResult` type.

```go
StreamingResult(EventResultType)
HTTP(func() {
    GET("/events/stream")
    ServerSentEvents(func() {
        SSEEventType("event_type")
        SSEProjection("legacy", "legacy")
        SSEProjection("updated", "updated")
    })
})
```

Declare at least two mappings. Event values and views must be unique, and each
view must belong to the streaming result type. A compatibility view can select
flat canonical fields while omitting a union field; another view can select the
canonical union wrapper. OpenAPI and `x-loom-async` render these alternatives as
`oneOf`. Without `SSEProjection`, SSE encoding remains unchanged.

Handle Last-Event-Id for resumable streams:

```go
Method("stream", func() {
    Payload(func() {
        Attribute("startID", String, "Last event ID received")
    })
    StreamingResult(Event)
    HTTP(func() {
        GET("/events/stream")
        ServerSentEvents(func() {
            SSERequestID("startID")  // Maps Last-Event-Id header
        })
    })
})
```

`SSERequestID` takes the DSL payload attribute name, such as `last_event_id`.
Generated HTTP handlers resolve its Go field name, including a
`Meta("struct:field:name", ...)` override, and preserve whether the field is
optional or required. A nonempty `Last-Event-ID` header replaces the decoded
payload field; without that header, the decoded value is retained.

### SSE Implementation

```go
func (s *Service) Stream(ctx context.Context, stream sse.StreamServerStream) error {
	control, ok := any(stream).(loomhttp.SSEControl)
	if !ok {
		return errors.New("SSE controls unavailable")
	}
	if err := control.Open(ctx); err != nil {
		return err
	}

	events := time.NewTicker(time.Second)
	heartbeats := time.NewTicker(15 * time.Second)
	defer events.Stop()
	defer heartbeats.Stop()

	for {
		select {
		case <-events.C:
			event := &sse.Event{
                Message:   "Hello from server!",
                Timestamp: time.Now().Unix(),
            }
			if err := stream.Send(event); err != nil {
				return err
			}
		case <-heartbeats.C:
			if err := control.SendComment(ctx, "heartbeat"); err != nil {
				return err
			}
        case <-ctx.Done():
            return nil
        }
    }
}
```

`Open` is lazy and idempotent: it flushes the normal `200 text/event-stream`
headers once, while a typed send or comment also opens the stream when needed.
Comments reject carriage returns and line feeds, share the typed-event write
lock, and are ignored as domain events by generated clients. Operations after
closure return `loomhttp.ErrSSEStreamClosed`; canceled requests return their
context error. A send also closes the stream when it fails after writing part
of a frame, when the response write itself fails, or when a flush fails,
including the flush in `Open`. Later `Open`, `SendComment`, and event sends then
return `loomhttp.ErrSSEStreamClosed` wrapping the original failure, and both
match with `errors.Is`. A send that fails before writing any bytes, such as an
event encoding error, returns that error and leaves the stream open, so the
handler can still send an error event. A send whose context is canceled after
the event was serialized returns the context error, but the buffered event may
still reach the client, so do not re-send it.

SSE uses the same optional `StreamWritePolicy` shown above. A positive timeout
bounds every frame write and flush using `http.ResponseController`; zero keeps
the existing unbounded behavior. Transports that cannot install the required
deadline return `loomhttp.ErrStreamWriteDeadlineUnsupported`.

Generated Go clients use the service's typed client stream interface for both
HTTP and JSON-RPC SSE. `Recv()` reads using the context passed when the stream
was opened; `RecvWithContext(ctx)` uses the supplied operation context. Client
streaming interceptors receive that same operation context. `Close()` releases
the response body, including through an interceptor wrapper. Close a stream
when stopping before EOF; canceling an active SSE read also terminates its
underlying response stream.

Regenerate clients and servers when adopting this contract. Replace direct SSE
`stream.Recv(ctx)` calls with `stream.RecvWithContext(ctx)`. Direct HTTP stream
constructor calls now supply the opening context:
`NewWatchStream(ctx, response, decoder)`. The generated transport endpoint
supplies it automatically. No-context `Recv()` is useful when the opening
context already has the intended lifetime bound.

A service method exposed over SSE and gRPC shares its client interface, including
`Close()`. On gRPC, this operation retains `CloseSend` semantics; cancel the
original RPC context to terminate reception. Per-operation gRPC cancellation
is a separate transport concern.

Browser client:

```javascript
const eventSource = new EventSource('/events/stream');

eventSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Received:', data);
};

eventSource.onerror = (error) => {
    console.error('EventSource failed:', error);
eventSource.close();
};
```

---

## Inbound Request Metadata

Use `loomhttp.RequestMetadataMiddleware` when endpoint or security code needs a
single immutable HTTP request snapshot. Apply it with the generated server's
`Use` method so metadata is present before generated decoding and security
functions execute.

```go
metadataPolicy, err := loomhttp.NewRequestMetadataPolicy(
    []string{"X-Tenant-ID"},
    []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
)
if err != nil {
    return err
}
srv.Use(loomhttp.RequestMetadataMiddleware(metadataPolicy))
```

The direct peer must match a trusted CIDR before Loom accepts
`X-Forwarded-For`, `X-Forwarded-Host`, or `X-Forwarded-Proto`. Client selection
walks `X-Forwarded-For` from the nearest hop on the right and selects the first
untrusted address. Repeated header lines are read in the same order as one
comma-separated list. If an empty or invalid IP entry is reached first, Loom
stops and keeps the direct peer as `ClientAddr`; it never skips that boundary
to select an address farther left. Entries must be bare IP addresses: values
such as `unknown`, `_hidden`, or `198.51.100.7:1234` are invalid. An absent header
or a chain containing only trusted addresses also retains the direct peer.
`PeerAddr` always remains the direct peer.

For example, with trusted peer `10.0.0.2` and trusted CIDR `10.0.0.0/8`,
`6.6.6.6, unknown, 10.0.0.1` keeps `ClientAddr` at `10.0.0.2`. In contrast,
`unknown, 198.51.100.7, 10.0.0.1` selects `198.51.100.7`: the untrusted address
establishes the boundary before the invalid entry is reached. Upgrade the
framework runtime to adopt this correction; generated code does not change.

Endpoint and security implementations use the typed accessor:

```go
metadata, ok := loomhttp.RequestMetadataFromContext(ctx)
if !ok || metadata.Origin != "https://app.example.com" ||
    metadata.SecFetchSite != "same-origin" {
    return nil, loom.MakeError("forbidden", "same-origin request required")
}
tenantIDs := metadata.HeaderValues("X-Tenant-ID")
```

`Origin`, `Sec-Fetch-Site`, request ID, method, path, effective host/scheme,
client and peer addresses, and user agent have typed fields. Additional headers
are retained only through the allowlist. `Authorization` and `Cookie` are absent
unless explicitly selected; cookie-based security can opt in when necessary:

```go
policy, err := loomhttp.NewRequestMetadataPolicy([]string{"Cookie"}, trusted)
// Never log metadata.Headers(), Authorization, or Cookie values.
```

`HeaderValues` and `Headers` always return copies, so consumers cannot mutate
the request or another consumer's snapshot.

The `clue/log` HTTP middleware uses `loomhttp.EffectiveClientAddress`. It honors
`ClientAddr` from this metadata snapshot and otherwise logs only the direct
network peer; forwarding headers are never trusted without an installed policy.

---

## Cross-Origin Requests

Use `CORS` in API or service HTTP scope to make browser access part of the
design. Service-level CORS overrides the API-level policy for that service.
Generated servers mount `OPTIONS` preflight handlers for designed routes and
write actual-request CORS headers through the shared `loomhttp` runtime helper.
An authored `OPTIONS` endpoint can share the same path: requests with an
`Access-Control-Request-Method` header use preflight handling, while ordinary
`OPTIONS` requests invoke the authored endpoint.

```go
var _ = API("tasks", func() {
    HTTP(func() {
        CORS(func() {
            Origin("https://app.example.com", func() {
                Methods("GET", "POST")
                Headers("Authorization", "Content-Type")
                ExposeHeaders("X-Request-Id")
                MaxAge(600)
                Credentials()
            })
            OriginRegex(`^https://preview-[^.]+\.example\.com$`)
        })
    })
})
```

Use `Origin("*")` only for non-credentialed APIs. Designs that combine
`Origin("*")` with `Credentials()` are rejected because browsers do not accept
that combination. OpenAPI output records the effective route policy under the
`x-loom-cors` extension.

For deployment-configured origins, select runtime mode in the design:

```go
HTTP(func() {
    RuntimeCORS()
})
```

Load the raw `loomhttp.CORSPolicy` in application startup, validate and snapshot
it with `loomhttp.NewRuntimeCORSPolicy`, then pass the resulting immutable value
to the generated server constructor. Invalid empty origins, regular
expressions, wildcard credentials, and negative max ages return configuration
errors. Generated actual-request and preflight handling remains identical for
ordinary HTTP and SSE routes. Runtime OpenAPI metadata marks the policy as
runtime-provided without publishing configured origins.

---

## Static Content

> **Design Recap**: Static file serving uses the `Files` DSL function. This is an HTTP-only feature. See [DSL Reference: Static Files](dsl-reference.md#static-files) for design patterns.

Serve static files using the `Files` function:

```go
var _ = Service("web", func() {
    // Serve files from a directory
    Files("/static/{*path}", "./public")
    
    // Serve a specific file
    Files("/favicon.ico", "./public/favicon.ico")
})
```

For Single Page Applications, serve the index.html for all routes:

```go
var _ = Service("spa", func() {
    // API endpoints
    Method("api", func() {
        HTTP(func() {
            GET("/api/data")
        })
    })
    
    // Serve SPA - catch-all for client-side routing
    Files("/{*path}", "./dist/index.html")
})
```

---

## See Also

- [DSL Reference: Streaming](dsl-reference.md#streaming) — Design-level streaming patterns
- [DSL Reference: Static Files](dsl-reference.md#static-files) — Files DSL for static content
- [DSL Reference: Error Handling](dsl-reference.md#error-handling-design-level) — Design-level error definitions
- [gRPC Guide](grpc-guide.md) — gRPC transport features
- [Error Handling Guide](error-handling.md) — Complete error handling patterns
- [Loom log package](https://pkg.go.dev/github.com/CaliLuke/loom/clue/log) — HTTP middleware for observability

---

## Best Practices

### URL Design
- Use nouns for resources: `/articles`, not `/list-articles`
- Use plural nouns consistently
- Let HTTP methods define actions
- Keep URLs hierarchical and predictable

### Error Handling
- Map errors to appropriate HTTP status codes
- Use consistent error response formats
- Include meaningful error messages

### Performance
- Use appropriate buffer sizes for WebSocket
- Implement connection pooling for high-traffic services
- Consider message batching for streaming endpoints

### Security
- Always use HTTPS in production
- Model browser CORS policy in HTTP design with `CORS` or `RuntimeCORS()`
- Validate all input parameters
- Set appropriate timeouts for long-lived connections
