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

Every generated method/path registration has one owner. Loom rejects conflicting
effective routes before generation, including HTTP endpoints, JSON-RPC dispatchers,
static files and automatic GET or CORS OPTIONS routes. Parameter names do not make
a route distinct: `GET /items/{id}` conflicts with `GET /items/{key}`. Literal
children and catch-all routes may coexist because routing precedence distinguishes
them. Logical methods within one JSON-RPC service intentionally share its dispatcher.

Services selected together by a `Server` must have distinct route ownership.
Disjoint server selections may reuse routes, but one service's own mounts must
always be unambiguous. Omitted service selection hosts all services. Two services
with different ordinary verbs can still conflict through their CORS preflight
routes; place them at different paths or on separate servers. A local explicit
OPTIONS endpoint continues to own its service's preflight wrapper. Static-file
mounts do not acquire CORS implicitly. OpenAPI's document-wide path-template rules
remain stricter than physical routing. Regenerate clients and servers after fixing
a rejected design.

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

By default, named query maps use one JSON document per parameter value. For example,
`Param("filter")` with `MapOf(String, String)` sends
`filter=%7B%22status%22%3A%22active%22%7D`, whose decoded value is
`{"status":"active"}`. Nested maps and arrays inside the map use the same JSON
representation. Clients in any language should JSON-encode the map, then pass
that string to their URL query encoder. Scalar parameters and repeated scalar
arrays keep their ordinary query representation.

A JSON query value applies its declared default only when the query key is
absent. A supplied document, including `{}`, a zero or an empty string, is
decoded and validated as sent. Servers reject a `null` document with a
`decode_payload` response. The JSON query maps and unions that Loom generates
also reject `null` members and validate every member before building the
payload. String elements with a custom Go type keep that type's text codec
inside the generated map. `struct:field:type` on a whole map does not replace
the generated map type, so such a map is decoded and validated like any other
map; only an emitted custom Go type owns its inner representation. The
generated CLI flag for a JSON query value applies the same rules before
building the payload: an explicit `null` document or member is an error rather
than an omitted value, and an omitted flag uses the declared default.
Regenerate clients and servers to adopt these rules; previously a defaulted
JSON query value could fail to compile, and the CLI accepted `null`.

`MapParams()` remains an unprefixed whole-query map, for example
`status=active&region=west`. Its contract differs from a named `Param("filter")`.
OpenAPI describes default named maps with parameter `content: application/json`
and a typed schema. Use explicit serialization metadata to select a different
wire representation.

### Parameter serialization

Set `openapi:style` and `openapi:explode` on the mapped parameter occurrence.
These settings control generated clients, servers and OpenAPI together:

```go
HTTP(func() {
    GET("/items")
    Param("filter", func() {
        Meta("openapi:style", "deepObject")
    })
    Param("tags", func() {
        Meta("openapi:style", "form")
        Meta("openapi:explode", "false")
    })
})
```

A scalar map `filter` now sends `filter[status]=active` (with URL-escaped
brackets), and the scalar array `tags` sends `tags=red,blue`. Fixed objects with
scalar fields are also supported. Metadata on an ordinary JSON body field is
rejected; parameter serialization belongs to a text binding or form encoding.

| Location | Supported styles | Default |
| --- | --- | --- |
| Path | `simple`, `label`, `matrix` | `simple`, not exploded |
| Query | `form`, `spaceDelimited`, `pipeDelimited`, `deepObject` | `form`, exploded; named maps use JSON unless a style or explode is authored |
| Request and response header | `simple` | `simple`, not exploded |
| Request cookie | `cookie`, `form` | `cookie`, exploded |
| Named URL-encoded form field | `form`, `spaceDelimited`, `pipeDelimited`, `deepObject` | `form`, exploded; named objects/maps use JSON unless encoding flags are authored |

Styles accept scalars, scalar arrays, fixed objects with scalar properties and
maps with scalar keys and values. `deepObject` requires an object or map;
its `explode` flag has no effect. Space- and pipe-delimited styles require a
collection and reject `explode: true`. Objects with `AdditionalAttributes`
require JSON bodies; text bindings and form bodies do not support their
undeclared members. Nested values need an admitted JSON
content binding. Text `Any` and nullable text values are rejected. Untagged text
unions require a supported codec for every branch and the same unique schema
and typed-decoder match on decode and encode.
JSON body `Any` retains its existing behavior. `openapi:allowReserved` remains
documentation metadata: it does not relax the selected codec's escaping or
data-alphabet rules. Its explicit presence still selects form-field style rules.

`Bytes` use raw text in text bindings, subject to the location's escaping and
alphabet rules. For example, bytes containing `hello` send `hello`. JSON-valued
bindings and JSON bodies retain base64 encoding.

Text object properties use their DSL member names. JSON field tags affect JSON
bodies and JSON-valued form fields; they do not rename or hide text properties.

Exploded objects claim their property names. A dynamic exploded map claims the
remaining names in its query, cookie or form namespace. Only one such map may
claim that remainder; its keys cannot use names reserved by sibling bindings,
including credentials. A fixed `deepObject` claims its exact bracketed property
names; a dynamic map claims the bracketed prefix. Overlapping claims are rejected
during design validation.

Generated encoders reject values that cannot round-trip through the selected
grammar. This includes delimiter collisions in unescaped text and a required
collection that emits no wire key. An optional zero-key collection is absent on
the wire; defaults apply to absence. Label paths escape dots inside values before
joining members. Exploded-label values whose dots cannot survive the server's
path decoding are rejected. Newly styled path helpers return `(string, error)`;
ordinary simple-path helper signatures are unchanged.

Scalar header codecs reject leading or trailing spaces and tabs because HTTP
framing removes them. Interior spaces and tabs remain data; list and object
values also follow the simple-list delimiter rules.

Raw `cookie` style sends cookie data verbatim. It rejects commas, quotes,
semicolons, backslashes, non-ASCII data and leading/trailing whitespace; interior
spaces remain unquoted. Choose `form` for percent-encoded cookie values. Form
cookies support scalars and non-exploded collections; exploded form collections
have no equivalent cookie-pair grammar. OpenAPI 3.2 represents raw cookies
explicitly. Export to 3.1 requires proof that the complete raw-cookie value and
key domains are equivalent to form encoding; otherwise generation fails. An
unrestricted raw string does not meet that condition. Neither does `FormatUUID`
alone: validation accepts URN and brace spellings that form encoding changes.
A UUID enum containing only unreserved spellings can meet the condition.

Regenerate clients and servers together when changing a style. In particular,
old bracket-style requests require explicit `deepObject`, and text `Any`
bindings must become concrete types or move into a JSON body.

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

Form encoding follows the authored body type. An object with one union field
keeps that named field; it is not flattened. For a flat tagged form such as an
OAuth grant, declare the payload as a root union with
`Type("Grant", OneOf(func() { ... }))`. If an older design relied on implicit
flattening, change it to a root union and regenerate. Generated service calls
then use that union directly instead of a one-field wrapper.

A form field whose encoding declares `application/json` can contain a nullable
union. An absent optional field remains absent; a field containing `null` is
present. Required fields must be present. Nullable styled text fields and
nullable form body roots remain rejected. Regenerate adapters to capture these field policies.

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

Nested complex members of a flattened object branch still use named JSON form
fields. A map branch uses the value key with one JSON value; nullable maps retain
JSON `null`, including nullable complex members. Arrays of complex values emit
one JSON value per item; an inner array is one JSON item. A repeated-array branch
can emit only its discriminator when empty. Nil and nonnil empty arrays have identical form wire and decode to a
nonnil empty array; minimum-length validation still applies. Nullable repeated
arrays and nullable flattened objects have no distinct null spelling and are
rejected for native form occurrences. A payload field cannot claim the
same physical form name as the discriminator, including inherited or renamed
fields. These restrictions do not affect a type used only in JSON.

OpenAPI describes native tagged roots with flat occurrence-specific alternatives,
including their examples, while JSON uses retain the tagged envelope. Adding
OpenAPI hint metadata does not change encoding, branch selection or validation.
Regenerate transports to use captured native array layout and JSON nullability.


By default, a named object or map form field also holds one JSON document. For example,
`address={"street":"Main"}` and `labels={"region":"west"}` are the decoded
form values; the form encoder percent-encodes the JSON strings on the wire.
This applies at every nesting depth. JSON tags and custom JSON codecs determine
member names inside these values; form tags determine the outer field name.
For example, `Meta("struct:tag:form", "wire")` changes an outer field to `wire`
for both generated clients and servers. A full `struct:tag:*` override replaces
the generated tag set: if that set has no form tag, an empty form tag, or `-`,
the field is omitted from form encoding. A JSON-name-only override preserves
the default form name. As for JSON query values, `struct:field:type` on a whole
map field does not replace its generated JSON value, which keeps its `null`
member checks; custom element types keep their codec, and a custom string field
keeps its own text codec. Regenerate clients and servers to adopt this.
Encoding metadata still uses the HTTP element name;
OpenAPI exports the effective form name and rejects overlapping field claims.
OpenAPI marks these fields with `contentType: application/json`, without a
serialization style. Scalars and repeated scalar arrays retain their form
encoding. To select a text grammar, put `openapi:encoding:<field>:style` and
`openapi:encoding:<field>:explode` on the body occurrence. Any authored `style`,
`explode` or `allowReserved` flag selects style serialization instead of the
implicit JSON encoding, even alongside `contentType: application/json`.
For example, `Meta("openapi:encoding:labels:style", "deepObject")` sends
`labels[region]=west`. The same rules apply independently to URL-encoded branches
of a multi-media request or response.

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

An explicitly selected empty object response is a JSON object (`{}`). For
example, `Body("record")` preserves a named record with no fields, and
`OpenAPIBody` retains its object schema. Use `Body(Empty)` for no response body.
Without an explicit body, a result with all fields mapped to headers or cookies
continues to have no body. Regenerate clients and servers to adopt this
correction if they previously omitted an explicitly selected empty object.

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

Loom generates both client encoding and server decoding for object bodies and
recursive untagged `OneOf` alternatives whose leaves are objects. Incoming
untagged alternatives must have exactly one matching schema and one successful
typed decoder. Outgoing values must decode back to the same selected branch,
including nested selections. Ambiguous or unrepresentable values fail before
request body publication. Tagged constructor unions remain unsupported as
multipart roots; they may appear inside JSON-valued fields.

`Bytes` fields carry raw part bytes. Arrays use one part per item: byte items
remain raw, text items remain literal, and object or nested-array items use JSON.
An explicit JSON Encoding Object on a typeless union encodes the whole selected
value. JSON-valued parts retain JSON tags, nullable values and recursive shapes.
Explicit style metadata selects the shared collection grammar and overrides
content-type metadata. Multipart styles never URI-escape names or contents;
plus signs and percent escapes stay literal. A delimiter inside a delimited
collection item cannot be represented and is rejected; repeated parts retain it.

Outer part names follow form tags, independently of nested JSON tags. A body with
one scalar byte field can derive string `filename` and `content_type` fields from
that uploaded file's headers. An authored target part, even an empty one, takes
precedence. Multiple byte fields disable derivation; duplicate scalar file parts
are rejected. Derived metadata does not create extra physical parts for closed
object or union matching.

`OptionalRequestBody()` permits an absent multipart body. A present empty MIME
envelope remains a supplied object and undergoes normal branch matching.
Malformed envelopes and source failures remain errors for optional bodies.

Regenerate clients and servers together. Automatically supported bodies no longer
accept multipart encoder or decoder callbacks in generated constructors. Remove
those callback arguments from application wiring. Explicit custom Go carrier
contracts and unsupported ordinary root shapes retain custom callbacks. A custom
decoder constructs the payload; the generated adapter then decodes and validates
mapped parameters and credentials before invoking the endpoint.

Direct `loomhttp.ReadMultipartForm[WithLimit]` callers now consume the ordered
`MultipartForm.Parts` slice. Each `MultipartPart` contains `Name`, `Filename`,
`ContentType` and raw `Data`; the separate `Values` and `Files` maps and
`MultipartFile` type have been removed. Preserve slice order when interpreting
repeated fields. The request's complete encoded source still counts toward the
body limit, and callers retain ownership of `Body.Close`.

### Optional Request Bodies

Use `OptionalRequestBody` when an endpoint accepts either no body or a typed
body. It applies to JSON bodies and to generated `FormRequest` and
`MultipartRequest` bodies:

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

The optional body must be an object, map or union that its media type supports.
JSON accepts objects, maps and unions; a JSON object or map payload must
declare its body with `Body`, while a union payload uses its root body. Form
bodies follow the form contract above: objects, maps, tagged unions, and
untagged unions whose alternatives are non-nullable objects or maps. Multipart
bodies must use the generated contract described under Multipart Requests;
explicit custom carrier contracts and other shapes that need multipart
callbacks cannot be optional. A body selected with `Body("name")` must come
from an optional payload attribute. For JSON and form, an object body built
from the payload attributes cannot contain required attributes; an optional
multipart object may retain required members, checked when the body is
supplied. `OptionalRequestBody` cannot be combined with raw request body
streaming.

The built-in JSON decoder reports an empty body, or one containing only JSON
whitespace, as end of input. An optional body treats that as absence; a
required body rejects it with `missing_payload`. A required form body decodes
zero bytes as an empty value set, then applies its field validations. For an
optional form body, zero bytes mean absence; nonempty input such as `&&` is a
present empty value set even though it has no decoded keys. An optional
multipart body is absent only when it has zero bytes; an empty MIME envelope is
supplied. These rules apply to ordinary objects, maps and unions. Malformed
input and validation errors still fail normally.

For a whole native object body, absence leaves the service payload nil unless
separately mapped fields or credentials require an outer payload. Those inputs
are still decoded and validated, even when the body is absent. Body-field
defaults apply only when the body is supplied; separately mapped fields keep
their own defaults. Regenerate clients and servers to adopt this absence
preservation across JSON, form and multipart requests.

An object or constructor union selected with `Body("name")` from an optional
payload attribute is optional with or without `OptionalRequestBody`. When the
attribute is nil, the generated client sends no request body, without a
`Content-Type` header. The generated server decodes an empty or
whitespace-only JSON body, or a zero-byte form body, as a nil attribute. A present body is
decoded and validated normally, so the required fields of the object apply
only when the client sends a body. The built-in JSON decoder rejects a
`null` body with `decode_payload` unless the body allows null. Optionality
permits absence, not null. A JSON-RPC method that selects its params the same
way works alike: the client omits `params` for a nil attribute and the server
decodes absent `params` as a nil attribute. The server rejects non-nullable
`"params": null` with `-32602`.
An object value `{}` is decoded and validated normally; a union still needs
its discriminator.

For a generated union, an empty `Kind()` means absence even when the Go
pointer is nonnil. The client omits an optional body in that state and rejects
an absent required body before conversion or encoding. A selected null branch
remains present when the union permits null. These presence rules apply to
JSON, form and multipart bodies and to unions selected with `Body("name")`;
separately mapped query and header values are still sent when the body is
optional. Regenerate clients to obtain the corrected presence checks.

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

### Selected Response Presence

`Body("field")` inside `Response` requires a concrete result and a supplied
selected field. This applies even when that field is optional in the result
type. Missing fields fail before response publication; supplied empty strings,
empty collections and scalar zeros remain valid. Explicit null is valid only
when the selected member admits null. Parent nullability does not broaden the
selected member's contract.

A response with only mapped headers/cookies also requires a concrete result.
Clients reconstruct a concrete result even if all optional mappings are absent,
with their normal defaults and requiredness checks. Whole-body nullable responses
remain distinct: null cannot carry mapped metadata. Regenerate transports after
upgrading; absent optional selected fields no longer produce JSON null.

### Tagged Success Responses

Use `Tag("kind", "accepted")` inside a method success `Response` to select it
when the result field matches. The field must exist and have String kind,
including named String types. Pass the logical field name even if its Go field
name or wire mapping differs. Optional absent and nullable null values do not
match. Tags retain authored first-match priority, followed by the single untagged
fallback. Generated clients reconstruct the selected tag with its declared
carrier, including named, nullable and projected fields. Regenerate both clients
and servers after upgrading; missing and non-string Tag fields now fail design
admission instead of producing invalid Go.

### Status-selected Responses

Use an exact status integer, `Status2XX` through `Status5XX`, or
`StatusFallback` as a response key. Exact keys take precedence over a matching
class, followed by the fallback. A key belongs to success responses or named
errors, never both. Named errors can share a key and retain their error identity.

| Key | Final statuses it selects |
| --- | --- |
| Exact, such as `404` | that status |
| `Status4XX` (likewise `Status2XX`, `Status3XX`, `Status5XX`) | the class, minus declared exact keys |
| `StatusFallback` | 200 through 599, minus declared exact keys and classes |

Declaration order never changes selection. A body-bearing response also
excludes 204 and 304. The server checks a published status against its selected
key before writing headers. A status outside that domain becomes a
`*loomhttp.StatusViolationError` framework error, routed through the ordinary
error encoder without a designed error name. The client selects one key per
final status. A response that then fails decoding returns an error; Loom does
not try another key.

Multiple untagged success responses, or a non-exact success response, select
from a native `StatusField` shared by all success branches. The field must be a
required, non-nullable `Int` without a default. Its constraints must admit at
least one status owned by the response key. A custom error can bind its own
native status field; the transport Problem envelope does not supply native
storage. JSON-RPC error numbers remain a separate protocol contract.

```go
HTTP(func() {
    GET("/value")
    Response(StatusOK, func() {
        StatusField("http_status")
        Body("existing")
    })
    Response(Status2XX, func() {
        StatusField("http_status")
        Body("created")
    })
})
```

The status field is excluded from inferred bodies. Mapping it to an explicit
body, header or cookie is rejected. It must remain present in every admitted
result view. Members owned by only one success branch are optional in native
result storage, while the branch retains their original requiredness and any
wire-local `Required` declarations. Payloads and shared type declarations retain
their requirements. Tag selection remains available for exact keys; it cannot
be mixed with class or fallback success keys.

`Informational("hints", StatusEarlyHints, Hints, func() { Header("link:Link") })`
declares a separately typed interim value. It carries headers, not a body, and
its name and key are unique within the endpoint. Exact statuses 100 and 102
through 199 may omit `StatusField`; `Status1XX` requires it. An exact 100 declaration is
importable, exportable and observable, but has no permitted server emission:
its generated `Send` returns a status violation. net/http owns automatic
`100 Continue`; 101 belongs to protocol upgrades and cannot be declared.
See [Informational Responses](#informational-responses) for sending and observing them.

For an error-only non-streaming operation, declare `NoSuccessResponse()` with
an empty result and at least one named error response. This suppresses the
implicit success response. JSON-RPC does not support this declaration. A nil
service error is then a framework violation, and a client receiving a final
status outside the error keys returns an invalid-response error.

An HTTP SSE response can pass a separate handshake type:
`Response(StatusCreated, Handshake, func() { Header("subscription:X-Subscription") })`.
Headers and their requirements come from that type, independently of
`StreamingResult`. A singleton exact handshake needs no status field; multiple
success keys or a class/fallback handshake require the same `StatusField` on
all branches. All successful branches use the same native handshake type.
The handshake supplies metadata, never the event body, and cannot use Tags.
Call the generated stream's `SetResponse` before its first successful write when
status or required metadata must be supplied. Without `SetResponse`, optional
members use their authored defaults and other members are omitted; explicit
zero values are published as supplied. Cookie policy for this implicit metadata
receives the request context. A rejected replacement keeps the
previous valid metadata. Missing or invalid metadata fails before response
publication. Once the response opens or terminates, setters return an error.
SSE success domains exclude 204 and 304 even when no event is sent.

For a method with both `Result` and `StreamingResult`, the typed handshake owns
status, headers and cookies for either negotiated success representation. The
unary body is the whole `Result`; SSE events come from `StreamingResult`.
Matching member names do not remove body fields or fill them from metadata.
Their views are independent. On the non-streaming path, discarded stream sends
and closes do not publish a response or prevent a later `SetResponse`.

Regenerate service and transport packages together. A typed mixed service client
returns `(result, metadata, error)` on its unary method; the streaming method
returns a stream whose `Response()` exposes the same metadata type. Both result
and metadata are validated before a successful unary return. Errors return zero
values for both. Default, untyped methods retain their existing client shape.
Typed mixed methods follow the existing HTTP-only streaming admission rules;
separate HTTP and JSON-RPC methods can still share one service.

Generated client interceptors on the unary method receive the response carrier
`*<service>.<Method>ResponseData` from `next`, with separate `Result` and
`Metadata` fields. They may modify either field, but must return that carrier.
Any other result fails the call with an invalid typed response metadata carrier
fault. Server service methods keep their `(result, error)` signature and set
metadata through the stream's `SetResponse`.

### Informational Responses

An `Informational` declaration generates a typed server emitter and a typed
client observer. Both are named after the method and declaration. For example,
`read` with `hints` produces `SendReadHints` in the HTTP server package and
`WithReadHints` in the HTTP client package:

```go
var Hints = Type("Hints", func() {
    Attribute("trace", String)
    Required("trace")
})

Method("read", func() {
    Result(SelectionResult)
    HTTP(func() {
        GET("/read")
        Informational("hints", StatusEarlyHints, Hints, func() {
            Header("trace:X-Trace")
        })
        Response(StatusOK)
    })
})
```

```go
// HTTP server package: func SendReadHints(ctx context.Context, v *selection.Hints) error
if err := server.SendReadHints(ctx, &selection.Hints{Trace: "early"}); err != nil {
    return nil, err
}

// HTTP client package: func WithReadHints(ctx context.Context, fn func(context.Context, *selection.Hints, error) error) context.Context
ctx = client.WithReadHints(ctx, func(ctx context.Context, hints *selection.Hints, decodeErr error) error {
    return decodeErr // a nonnil return aborts the request
})
res, err := c.Read(ctx)
```

The emitter must receive the request context or a context derived from it. It
sends exactly the declared headers and never leaks or alters final response
headers. It returns:

- `loomhttp.ErrInformationalClosed` once final work has started (result or error
  encoding, SSE open, WebSocket upgrade, file, raw or redirect), or when `ctx`
  does not carry the request exchange;
- `loomhttp.ErrInformationalUnsupported` for HTTP/1.0 requests;
- `loomhttp.ErrInformationalLimit` after `loomhttp.MaxInformationalResponses`
  (8) authored interim responses in the exchange;
- `*loomhttp.StatusViolationError` for a value outside the declaration's status
  domain or header constraints. Nothing is sent.

Authored emission never sends 100 or 101. For `Expect: 100-continue`, net/http
sends at most one automatic `100 Continue`, outside the authored budget.

On the client, a callback runs synchronously, in arrival order, before the final
response. It receives the decoded value and any metadata decoding error; only
its returned error aborts the request. Treat headers as read-only. Exact keys
take precedence over `Status1XX`, even when the exact observer is nil. A
declared exact 100 observes automatic `100 Continue`; otherwise `Status1XX`
selects it. If the selected channel's status field constraints exclude 100, the callback receives a decoding error.
Undeclared statuses are ignored. A nil callback disables
observation for that declaration. If no callback is registered, the request is
unchanged: no trace hook and no budget are installed.

With observation enabled, each transport exchange accepts at most eight non-100
interim responses and one 100, including undeclared and intermediary responses.
Redirects and retries start new exchanges. Exceeding a budget aborts with
`loomhttp.ErrInformationalLimit`. After the first observer or budget error, no
further Loom callbacks run. The call returns that error joined with any error
from the transport and from closing the response body; use `errors.Is`. A
`Got1xxResponse` hook that you installed with `httptrace` still runs once per
interim response. Regenerate clients and servers to obtain emitters and
observers.

### Mapped Wire Names

Mappings split at the first colon: `Param("value:filter:mode")` names the query
parameter `filter:mode`. The entire suffix is retained for query and body field
names. Header and cookie names must be nonempty HTTP tokens; spaces, colons,
semicolons, control characters and non-ASCII characters are rejected during
design evaluation. This also applies to inherited, security and response bindings.
Regenerate clients and servers after correcting a rejected mapping, or when
a query/body mapping previously lost text after a second colon.

### Scalar Mapping Presence

Query parameters, request headers/cookies, and response headers/cookies distinguish
an omitted value from a supplied empty string. A supplied empty String or Bytes
value is retained, including optional fields. Required mappings reject omission,
even when they declare a default. Defaults fill optional absence only; they never
replace supplied empty strings, zero numbers, or false booleans.

Supplied values always reach their codec and DSL validation. For example, an empty
integer fails conversion, and an empty string with `MinLength(1)` fails validation.
A custom text codec receives supplied empty input and decides whether it is valid.
Scalar headers continue to use the first field value, without comma-list splitting.
These request rules also apply to JSON-RPC. Regenerate clients and servers after
upgrading; peers that previously sent empty values to request omission/defaults
must omit the mapping instead. Authored body tags and codecs retain their contracts.

A parameter default is authored by the service, so a default that a custom
text codec cannot parse is a server fault: the server responds with status 500
and code `fault` without calling the endpoint. The fault takes precedence over
client errors still being collected, such as a missing required query
parameter. It happens only when the default is needed, so requests that
supply the value are unaffected. This applies to string-backed query, header
and cookie parameters, styled parameters and their object properties, and
JSON query values, including member defaults applied while converting a JSON
query value. A generated client that decodes a response header or cookie with
such a default reports a `validation_error` whose cause is the fault. Untagged
text unions cannot declare custom Go types, so their generated branches never
meet such a default. Every branch of an untagged text union is still decoded,
and a hand-written branch error, including a fault, fails the whole match even
when another branch matches.
Regenerate servers and clients to adopt this classification; earlier servers
reported some of these failures as `decode_payload` (400) or
`invalid_field_type` (400).

### Collection Headers

Map an array of primitives with `Header("tags:X-Tags")` to use OpenAPI's
`simple` comma-list representation. `["red", "blue"]` becomes `X-Tags: red, blue`.
Named primitive elements and named arrays use the same wire representation.
Repeated lines and a combined comma list decode to the same ordered collection;
receivers ignore empty list members and trim separator spaces and tabs.

Nil means the header is absent. A nonnil empty collection sends a present empty
header and decodes back to an empty collection. Defaults apply only to absence.
A required empty collection is valid unless a length constraint rejects it.

Items cannot contain commas, be empty, or start or end with a space or tab.
Loom rejects such defaults, enum values and examples during design evaluation;
encoders reject unrepresentable runtime values before sending the request or
committing the response. Errors identify the header without echoing its values.
Use a JSON body for arbitrary text lists. Regenerate clients and servers to adopt
these rules; old clients interpret combined lists as single values.

Scalar headers retain their complete wire text. Quoting does not escape commas
in a schema-based collection: use a scalar for a field with its own quoted grammar,
such as an Accept value containing quoted parameters. `Set-Cookie` is not a
comma-list header; array mappings to it are rejected. Use the cookie DSL for
cookies. These rules also apply to JSON-RPC request metadata. JSON-RPC response values
belong in `result` or `error.data`; response Header/Cookie mappings are rejected.
gRPC metadata and trailers retain their separate transport contract.

### Raw Request and Response Bodies

Use `SkipRequestBodyEncodeDecode` when the service should receive the request
body as an `io.ReadCloser`, for example large uploads or pass-through
endpoints. Generated clients accept the caller-supplied `io.ReadCloser`. All
non-body payload attributes must be mapped to path parameters, query
parameters, or headers.

Map a payload field with `Header("length:Content-Length")` to declare the outgoing
body length. Regenerated clients apply that value after the body encoder runs,
including raw, custom and form bodies. Positive lengths keep streaming; the HTTP
transport detects a stream that ends at a different length. An omitted field
keeps the encoder's normal framing. Invalid lengths, known body mismatches and
conflicting transfer encoding fail before sending.

An explicit zero requires an empty body. Loom probes an opaque reader for EOF
without buffering the stream; a nonempty reader is rejected. Cancellation closes
the reader to unblock the probe, following Go's `http.Request.Body` contract.
After confirmed EOF and successful close, the request uses `http.NoBody` and an
empty replay function. Explicit zero is supported for POST, PUT and PATCH. Other
methods, including GET and HEAD, return an encoding error before sending because
Go HTTP/1 and HTTP/2 omit that header. Make the length mapping optional and omit
its value for empty bodies on those methods. Positive lengths remain supported.
Failed request encoding closes an attached unpublished
raw body; read, close and cancellation errors remain available through
`errors.Is`/`errors.As`. Regenerate clients to adopt this ownership and framing.

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

Documentation bodies can describe untagged scalar/object unions and nested
unions, for example `OpenAPIBody(OneOf(String, Manifest), Untagged)`. The same
schema works with `OpenAPIRequestBody`. These modifiers are local to the body;
sharing a named type with a typed method does not change that method's codec.
The application remains responsible for producing or consuming the documented
raw bytes. Runtime untagged union restrictions still apply to typed methods.

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
When `Content-Type` is absent, previews use JSON masking, matching the built-in
HTTP decoders. Untyped bodies that are not valid JSON are omitted; declare an
explicit content type such as `text/plain` for ordinary opaque previews.

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

JSON credential names also match Unicode case variants and names differing only
in dashes or underscores, covering aliases accepted by Go's
[`case:ignore` JSON tags](https://pkg.go.dev/encoding/json/v2#hdr-JSON_Representation_of_Go_structs).
This conservative masking also applies to strict fields; it does not change which
names the request decoder accepts. Existing generated metadata works unchanged.

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

Opaque URL previews are omitted in both Debug and access logs: an opaque URL can
carry userinfo and query credentials outside the parsed fields. Ordinary
hierarchical URLs keep their masked query previews. The original request URL is
unchanged, including its opaque representation.

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
The default negotiator adds `Vary: Accept`, including when `Accept` is absent or
falls back to JSON, so caches separate responses selected by different request
headers. Existing `Vary` fields are preserved. A fixed response `ContentType`
does not add this header because its encoding does not depend on `Accept`.

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
Valid structured `+json` media types (for example `application/merge-patch+json`)
use the JSON decoder everywhere. This parses JSON; merge-patch or other
media-specific business semantics remain the service's responsibility.

Inside a method's `HTTP` block, `ContentType("application/vnd.example+json")`
declares the exact media for a typed JSON request. MIME parameters are preserved.
Generated clients set it only when supplying the body, before calling a custom
encoder factory; OpenAPI advertises the same media. Absent optional bodies have
no static media header, while explicit null and empty collections remain supplied
bodies. Regenerate clients after importing or declaring a request media type.

Request `ContentType` cannot be combined with a mapped Content-Type header,
form/multipart framing, raw requests, JSON-RPC or WebSocket transport. It does not
add strict server negotiation: existing supported request codecs remain accepted,
and custom decoders still receive the original request and header.


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

### Request Cookie Values

Generated HTTP and JSON-RPC HTTP clients validate cookie names and values before
sending a request. Invalid values, including semicolons, quotes, backslashes,
control bytes and non-ASCII characters, return an `encoding_error`; Loom does
not silently remove characters or escape the value. Valid empty values, spaces
and commas retain their standard cookie representation. Absent optional cookies
remain omitted. Regenerate clients to adopt this validation.

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
`ResponseContractCases` manifest. Each case records its kind, the declared
status `Selector`, the status `Domain` the response may use, allowed base media
types, error name, and required response headers and cookies declared by the
design. An exact status such as `202` has
`Selector: loomhttp.StatusSelector{Code: 202}` and `Domain: []int{202}`, and a
WebSocket success case has the `101` handshake as its domain.
`ValidateResponseContract` checks the response status against `Domain` only.
The former `StatusCode` field is removed: regenerate with `loom gen`, and read
`Domain` or `Selector` in application code that inspected it. SSE success cases also record the stream direction,
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

`Open` is lazy and idempotent: it flushes the handshake status, its headers and
`Content-Type: text/event-stream` once, while a typed send or comment also opens
the stream when needed. The status is 200 unless the design selects another one
(see [SSE Handshake Status and Completion](#sse-handshake-status-and-completion)).
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

When an HTTP SSE endpoint returns an error, the generated handler closes writer
admission before routing that error. A later send, comment or `Open` cannot
append SSE bytes to an ordinary error response. If an admitted operation already
started the successful stream, the endpoint error goes to the configured late
error handler instead. Cancellation and invalid-comment errors retain their
usual precedence. Regenerate HTTP servers to adopt this failure-admission fix.

Regenerate servers to adopt stream lifecycle observation. With a transport
observer installed, SSE header commitment emits one opening event and handler
completion closes a committed writer once. An unopened stream emits neither.
Observer callbacks can reenter stream controls; they run outside writer locks.
A recoverable encoding failure does not imply that the stream has closed.

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

### SSE Handshake Status and Completion

The success responses described in
[Status-selected Responses](#status-selected-responses) determine the status:

- A single exact success, such as `Response(StatusCreated)`, publishes that
  status. The stream has `SetResponse` only when the response passes a
  handshake type.
- Several success keys, or a `Status2XX`/`StatusFallback` key, require a
  `SetResponse` value whose status field selects one of them. Loom never
  guesses a status.

The first of these operations publishes the handshake: `Open`, `SendComment`,
an event send, `Close`, or a nil return from the service method. A nil return
from an unopened stream publishes the handshake with zero events. Missing
status-mode metadata, a missing required handshake header or an invalid value
fails before anything is written. That failure takes the ordinary error path.
An error returned before the stream opens uses the declared error responses.
After the stream opens, a returned error terminates the stream; it cannot
replace the committed status.

The HTTP server stream's `Close` completes the stream successfully. On an
unopened stream it publishes the handshake under the same rules, using the
request context, and returns any publication error. Return that error from the
service method. Calling `Close` after the stream has already finished, failed or
closed does nothing and returns nil. To report a failure, return the error
instead of calling `Close`. The generated handler's cleanup runs on every return
and panic. It never publishes a response. Regenerate servers to adopt `Close` as
successful completion; previously, it ended an unopened stream without a
response.

Generated clients select the final response key before constructing a stream.
Error keys return errors. For a success key, `Content-Type` must parse as
`text/event-stream`, with optional valid parameters such as `charset=utf-8`. A
missing, malformed or different media type, including `text/event-stream-extra`,
is rejected before the stream is created. A stream with a handshake type
exposes the decoded metadata, including the actual status, through `Response()`,
even when no event arrives. Each JSON event is decoded as `application/json` data, independent of
the handshake status and of the request's media type. Raw field-level data
keeps the codec described in [Event Data Encoding](#event-data-encoding).
Regenerate clients to adopt this media check and status selection.

Browser `EventSource` fails the connection unless the status is 200. Generated
Go clients accept every declared success status, but an API consumed by
browsers should declare exact `Response(StatusOK)` for its SSE success.

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

For trusted peers, host and scheme use the final value across all
`X-Forwarded-Host` or `X-Forwarded-Proto` lines, whether the proxy appends a
separate line or a comma-separated entry. An empty final host keeps the request
host; an empty or unsupported final scheme keeps the request scheme. Earlier
values are never selected as a fallback. This prevents a client-supplied first
line from overriding the trusted proxy's appended value. The application must
still configure trusted CIDRs and ensure its proxy supplies authoritative values.

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

A file target is served directly, including an exact `/index.html` route or an
alias with a different name. Each mapping retains its own target, even when
request paths share a basename. File responses preserve MIME types, byte ranges
and conditional requests. Regenerate existing servers to adopt this behavior.

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
