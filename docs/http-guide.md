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

Loom generates form encoding and decoding for object payloads and constructor
union payloads. Form fields use the payload attribute names unless the design
uses HTTP element name mapping. For constructor unions, scalar branches use the
canonical discriminator/value fields, while object branches are flattened onto
standard top-level form fields. This supports OAuth-style token requests such
as `grant_type=refresh_token` without app-local parser hooks.

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

### Optional JSON Bodies

Use `OptionalRequestBody` when an endpoint accepts either no JSON body or a
typed JSON object body:

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

The optional body must be an object body and cannot contain required body
attributes. It cannot be combined with `FormRequest`, `MultipartRequest`, or
raw request body streaming. Generated decoders tolerate `io.EOF` only for
endpoints that opt into `OptionalRequestBody`; malformed JSON and validation
errors still fail normally.

### Raw Request and Response Bodies

Use `SkipRequestBodyEncodeDecode` when the service should receive the request
body as an `io.Reader`, for example large uploads or pass-through endpoints.
All non-body payload attributes must be mapped to path parameters, query
parameters, or headers.

Use `SkipResponseBodyEncodeDecode` when the service returns a raw response body
reader. These raw body modes are HTTP-only and cannot be combined with gRPC or
streaming payload/result methods.

---

## Content Negotiation

### Built-in Encoders

Loom's default encoders support:
- JSON (`application/json`, `*+json`)
- XML (`application/xml`, `*+xml`)
- Gob (`application/gob`, `*+gob`)
- HTML (`text/html`)
- Plain text (`text/plain`)

Response content type is determined by:
1. `Accept` header
2. `Content-Type` header (if no Accept)
3. Default (JSON)

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
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            event := &sse.Event{
                Message:   "Hello from server!",
                Timestamp: time.Now().Unix(),
            }
            if err := stream.Send(event); err != nil {
                return err
            }
        case <-ctx.Done():
            return nil
        }
    }
}
```

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

## Cross-Origin Requests

Use `CORS` in API or service HTTP scope to make browser access part of the
design. Service-level CORS overrides the API-level policy for that service.
Generated servers mount `OPTIONS` preflight handlers for designed routes and
write actual-request CORS headers through the shared `loomhttp` runtime helper.

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
