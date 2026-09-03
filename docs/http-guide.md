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
        GET("/users/{user_id:id}")  // URL uses user_id, maps to payload.ID
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

### Wildcards

Capture all remaining path segments:

```go
Method("serve_files", func() {
    Payload(func() {
        Field(1, "path", String, "Path to file")
    })
    HTTP(func() {
        GET("/files/*path")  // Matches /files/docs/image.png
    })
})
```

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

Loom's JSON runtime and generated transports use Go 1.27's
`encoding/json/v2`. Object member names match Go fields case-sensitively, and
decoding rejects duplicate names and invalid UTF-8. Custom JSON integrations
should use the same package. Use `encoding/json/jsontext.Value` for unparsed
JSON values instead of the legacy raw-message representation.

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

The optional body must be an object body, or a map when `FormRequest` is set,
and cannot contain required body attributes. It cannot be combined with
`MultipartRequest` or raw request body streaming. JSON decoders tolerate
`io.EOF` only for endpoints that opt into `OptionalRequestBody`; form decoders
accept an empty form. Malformed input and validation errors still fail normally.

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
reader. These raw body modes are HTTP-only and cannot be combined with gRPC or
streaming payload/result methods.

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

Debug capture redacts authorization and cookie headers, headers and URL query
parameters named for tokens, secrets, credentials, sessions, or API keys, and
sensitive fields nested in JSON or form bodies. Malformed structured bodies are
omitted instead of printed without redaction. Ordinary non-structured bodies
remain visible, so debug mode should not be enabled for production traffic or
payloads containing unmodeled secrets.

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
most 32 MiB. Generated multipart decoding uses the same 32 MiB aggregate limit
across all parts, including unnamed parts; it is not a per-file allowance.
Requests that exceed either limit receive an RFC 9457
`application/problem+json` response with status `413` and code
`request_too_large`, rather than the `decode_payload` response used for
malformed bodies.

JSON-RPC endpoints keep the protocol's HTTP `200` response and map the same
condition to an `InvalidRequest` error: code `-32600`, null `id`, message
`request body too large`, and error data name `request_too_large`. A
POST-initiated SSE request receives that envelope as a `message` event.

Generated clients also cap buffered response-body restoration at 32 MiB.

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
context-cancel unblocking, close-control frames, and JSON frame read/write
coordination so generated endpoint wrappers stay thin.

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
context error.

SSE uses the same optional `StreamWritePolicy` shown above. A positive timeout
bounds every frame write and flush using `http.ResponseController`; zero keeps
the existing unbounded behavior. Transports that cannot install the required
deadline return `loomhttp.ErrStreamWriteDeadlineUnsupported`.

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
untrusted address. `PeerAddr` always remains the direct peer.

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
