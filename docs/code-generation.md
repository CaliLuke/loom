---
title: Code Generation
weight: 3
description: "Complete guide to Loom's code generation - commands, process, generated code structure, and customization options."
llm_optimized: true
aliases:
---

Loom's code generation transforms your design into production-ready service
interfaces, endpoints, transport adapters, clients, and API contracts. The
`loom example` command creates runnable starter files. Each service stub
returns a Loom fault until the application replaces it.



## Command Line Tools

### Installation

```bash
go install github.com/CaliLuke/loom/cmd/loom@v1.8.0-alpha.15
```

### Pinning the Generator in a Module

Applications that want reproducible generation can record the Loom command as
a Go tool dependency:

```bash
go get -tool github.com/CaliLuke/loom/cmd/loom@v1.8.0-alpha.15
go tool loom gen example.com/myservice/design
```

This records the command's complete dependency graph in the application's
`go.mod` and `go.sum`. Installing only the root Loom module does not necessarily
record command-only dependencies such as the code generator's source-emission
packages.

For an existing `//go:generate` or script that intentionally uses `go run`, add
the command package—not only the root module—before invoking it:

```bash
go get github.com/CaliLuke/loom/cmd/loom@v1.8.0-alpha.15
go run github.com/CaliLuke/loom/cmd/loom gen example.com/myservice/design
```

### Commands

Generation commands expect Go package import paths, not filesystem paths:

```bash
loom gen github.com/CaliLuke/loom-examples/calc/design
```

#### Import OpenAPI (`loom import openapi`)

```bash
loom import openapi <input.json-or-yaml> [-o <design.go-or-directory>] [--allow-lossy] [FILTERS]
```

Run `loom import openapi --help` for the importer-specific flag reference.

Use operation filters to import one service boundary from a large contract:

```bash
loom import openapi monolith.json --tag "Face capture" --tag Videoselfie -o design/face.go
loom import openapi monolith.json --path-prefix /omni/b2b/v1 -o design/b2b.go
loom import openapi monolith.json --path "/omni/*/device-*" -o design/device.go
loom import openapi monolith.json --list-tags
```

Repeat a filter to add selections. Different filter types also form a union.
Tag matches are exact. Path patterns use Go `path.Match` syntax, where `*`
does not match `/`.

The importer retains the transitive component closure for the selected
operations. An unrelated component does not appear in the generated design or
its refusal set.

When a tag filter is active, the command reports each unclaimed path on
standard error. `--list-tags` reports deterministic operation and path counts
without writing a design.

This command creates one gofmt-formatted Loom design from the strict supported
subset of an OpenAPI 3.0, 3.1, or 3.2 contract. OpenAPI 3.0 inputs are
translated to the equivalent OpenAPI 3.1 design metadata. The default output is
`design/design.go`. An existing directory, a path ending in a separator, or a
non-existing extensionless path is treated as a directory; a `.go` path names
the output file directly.

See [OpenAPI Import Coverage](openapi-import-coverage.md) for the field and
schema-keyword matrix. The matrix distinguishes preserved, conditional, lossy,
and rejected constructs.

Import is intentionally lossless-or-fail by default: unsupported constructs are
reported together and no partial design or TODO placeholders are written. The
command also refuses to overwrite an existing target. Review the imported
design, then run `loom gen <module-import-path>/design` normally.

Use `--report` to print grouped blocker counts and affected operations without
writing a design. Use `--skip-unrenderable` to write all renderable operations
and report each skipped operation. Partial import also omits unsupported
document-level members such as `servers`, info metadata, and tag metadata. It
reports these under `skipped (document level)` without making
otherwise-renderable operations fail. Operation-level servers still make that
operation unrenderable.

API-key security schemes in headers, query parameters, and cookies import
without loss. The importer preserves root and operation requirement
alternatives, including AND requirements, an anonymous `{}` alternative, and
an explicit operation `security: []` override. Generated designs use
`Security()` with no scheme for the anonymous alternative and `NoSecurity()`
for the explicit empty override. Unsupported security scheme kinds, references,
locations, fields, and OAuth-style scope values remain strict diagnostics; the
importer never silently weakens an authentication contract.

Both modes use these exit codes:

| Code | Result |
|---|---|
| `0` | All selected operations are importable. |
| `3` | Some selected operations are importable; partial output is available. |
| `2` | No selected operation is importable; no design is written. |
| `1` | Usage, input, output, or another command failure occurred. |

`--report` and `--skip-unrenderable` work with operation filters and
`--allow-lossy`. Report mode never creates the output path.

The importer converts each `operationId` to an idiomatic Go method name. It
uses matching path words to split lowercase IDs and preserves initialisms such
as `B2B`.

Path parameter attributes keep their authored names so they remain identical
to route placeholders. For example, `{asset_id}` renders as an `asset_id`
payload attribute and `Param("asset_id")`; generated Go still uses the
idiomatic `AssetID` field name.

The importer maps `multipart/form-data` request bodies to
`MultipartRequest()`. It maps `application/x-www-form-urlencoded` request
bodies to `FormRequest()`. Both request body schemas must define an object.
The importer maps `type: string` schemas with `format: byte` or
`format: binary` to `Bytes`. Regeneration preserves the selected format.

If one request body lists multiple media types, each entry must use the same
schema and examples. Loom imports supported entries as one documented raw
stream. The generated service must inspect the content type and decode the
stream. A form or multipart entry requires a shared object schema. Different
schemas, examples, per-part encodings, and unsupported media types remain
strict import errors.

OpenAPI 3.0 `nullable: true` and OpenAPI 3.1/3.2 two-member type unions such as
`type: [string, "null"]` import as `Nullable()` and generate
`loom.Nullable[T]`. Authored `Nullable()` attributes use the same representation.
Its zero value means the property was absent; `loom.NullValue[T]()` means it
was explicitly null; and `loom.NullableValue(value)` carries a concrete value.
Use `Present`, `IsNull`, and `Value` when handling generated service types.

An unconstrained schema `{}` imports as Loom `Any`. This applies to component
schemas, object properties, array items, request bodies, and responses. The
regenerated OpenAPI preserves `{}`, including named component identity, and the
generated HTTP code accepts every JSON value: scalar, object, array, boolean,
or null. Generated payload and object fields for `{}` use `loom.Nullable[any]`
to distinguish an absent field from an explicit JSON `null`; use its presence
methods as described above. Direct named results retain their imported result
type and may return `nil`. A schema that omits `type` but declares constraints
is not treated as unconstrained and remains a strict import error.

A free-form object with `type: object` and `additionalProperties: true` imports
as `MapOf(String, Any)`. Generated Go uses `map[string]any`. Regenerated
OpenAPI preserves the object type and `additionalProperties: true`.

An object cannot combine declared members with `additionalProperties: true`.
The Loom DSL cannot preserve both parts of that contract, so import fails.

JSON-compatible `x-*` extensions are preserved at document, operation, schema,
parameter, request-body, and response scopes. Extensions at unsupported scopes
remain explicit import diagnostics and are never discarded silently.

A single-member `allOf` containing a local schema reference imports losslessly
as that reference. With `--allow-lossy`, the importer also supports the common
Spring inheritance shape `allOf: [$ref, inline object]`. It renders the parent
with `Extend(...)` and keeps the inline properties and required fields.
Regenerated OpenAPI flattens this inheritance relationship. When such an
inline object is used directly as array items, the importer promotes it to a
deterministically named component so Loom can render the array. The promotion
is reported as a lossy warning because it changes the schema's component
structure without changing its fields or validation. Other `allOf` shapes
remain blocked. The importer also blocks `oneOf`, `anyOf`, and `not`.
It rejects a Schema Object with `$ref` siblings because returning only the
reference would discard the sibling constraints. Wrap the reference in a
supported `allOf` shape instead.

OpenAPI 3.2 Media Type `description` requires `--allow-lossy` because the Loom
HTTP DSL has no media-level description. `prefixEncoding` remains a strict
error.

An operation must define exactly one primary successful response. The importer
uses its single 2xx response when present. If no 2xx response exists, it uses a
single 3xx response instead, so redirect-only operations import without
inventing a 2xx response. Other responses remain method errors.

For a single non-JSON success response, the importer keeps the media type and
schema as an OpenAPI-only body. It uses `FileResponse()` for a compatible
`GET` or `HEAD` response with status `200`. It uses
`SkipResponseBodyEncodeDecode()` for other methods and statuses. The service
implements the streamed response body.

The generated method keeps the original wire value in
`Meta("openapi:operationId", ...)`. Regenerated OpenAPI documents therefore
retain the source `operationId`.

The importer preserves schema `title`, `example`, `examples`, `default`,
`deprecated`, `readOnly`, and `writeOnly` without a flag. It also preserves
unformatted integer and number schemas. The `int32`, `int64`, `float`, and
`double` formats are also supported.

A schema-less error response stays bodyless. The importer does not add response
headers that are absent from the source contract.

The importer maps these members to `Title(...)`, `Example(...)`, `Default(...)`,
metadata, `Int`, or `Float64`. It also maps request and response media examples
to `Example(...)`. It reports null, external, or incompatible examples at
their source locations.

Use `--allow-lossy` only when you explicitly accept omission of non-contract
metadata or of constructs the Loom HTTP DSL cannot express per-parameter or
per-header. It writes the design and reports deterministic warnings to stderr
for: info metadata, external documentation, tag and path metadata, response
summaries, media type descriptions, and unrenderable or parameter/header
examples. It also reports
unrecognized `format` values and renders them without a format validation.
OpenAPI 3.1 specifies that an unknown format must not stop processing. The flag
also omits parameter-level or header-level `deprecated`. The HTTP DSL has no
deprecated marker for these items. The importer always preserves a schema's
own `deprecated` keyword. It never
downgrades contract-affecting diagnostics such as servers, security,
extensions, callbacks, links, custom serialization, media encodings, or
unsupported schema composition and structural keywords. The documented
`allOf` flattening and inline array-item promotion are explicit lossy
exceptions. Without `--skip-unrenderable`, other diagnostics prevent output.
With that flag, the importer omits each affected
operation. The document-level omissions use the separate behavior above.

#### Generate Code (`loom gen`)

```bash
loom gen <design-package-import-path> [-o <output-dir>] [--debug]
```

The primary command for code generation:
- Processes your design package and generates implementation code
- Generates and finalizes a complete staging tree, validates its manifest and
  outputs, then replaces the entire `gen/` directory and any declared plugin
  outputs on success. A generation, finalization, validation, or installation
  failure restores the previous generated artifacts.
- Run after every design change

#### Create Example (`loom example`)

```bash
loom example <design-package-import-path> [-o <output-dir>] [--debug]
```

A scaffolding command:
- Creates a one-time example implementation
- Generates handler stubs that return `loom.Fault`
- Does not create a success body, file, or stream event from a stub
- Run once when starting a new project
- Will NOT overwrite existing custom implementation

The transport encodes the fault as its standard internal-error response. Loom
does not add an undeclared HTTP `501` response to the design contract.

#### Create Contract Tests (`loom test-scaffold`)

```bash
loom test-scaffold <design-package-import-path> [-o <output-dir>] [--debug]
```

This command creates consumer-owned HTTP and gRPC response contract tests under
`internal/contracttest/`. Existing scaffold files are never overwritten. The
generated test enumerates the current server manifest and fails once for every
declared response that lacks an application callback, so later `loom gen`
changes remain visible without rewriting the scaffold.

Unary, multipart, SSE, and WebSocket cases use separate callback maps. A multipart
callback receives an `loomhttp.MultipartRequestContract` with the request
content type, part names, part media types, and required parts. The application
still owns multipart codecs and request fixtures. An SSE success callback returns
an `loomhttp.SSEResponseContractObservation` containing the handshake response,
parsed events, and terminal read error. The validator checks the handshake,
event data encoding, required ID and event-type fields, projection event types,
and clean stream completion.

A WebSocket callback receives the stream contract. It returns the upgrade
response, outbound JSON messages, and terminal read error. The validator checks
the `101` response, upgrade headers, JSON messages, and terminal behavior.
Declared errors that occur before an SSE frame or WebSocket upgrade remain
ordinary HTTP response cases.

Loom supports non-streaming multipart contracts for flat object bodies with
primitive or bytes fields. Other multipart shapes produce a generation
diagnostic. Multipart SSE and WebSocket endpoints also produce a diagnostic.

The gRPC scaffold publishes one stable case for each success or declared error.
Cases include the status code, protobuf success message or typed status detail,
and required header and trailer metadata. A callback receives the case and
returns a `loomgrpc.ResponseContractObservation` from a real generated client
call. Unary and server-streaming endpoints are supported. Server streams must
finish with clean EOF. Client-streaming and bidirectional endpoints produce a
generation diagnostic until their completion lifecycle is supported.

The JSON-RPC scaffold publishes success, declared error, and notification
suppression cases. Cases include the designed result type, error code, and
typed error-data name. Each callback returns a
`jsonrpc.ResponseContractObservation` from a real generated handler request.
Unary and server-SSE endpoints are supported. A server-SSE request with an ID
must finish with a final response. An ID-less stream must suppress that final
response. WebSocket, client-streaming, and bidirectional completion contracts
produce a generation diagnostic.

#### Show Version

```bash
loom version
```

### Development Workflow

1. Create initial design
2. Run `loom gen` to generate base code
3. Run `loom example` to create implementation stubs
4. Run `loom test-scaffold` to create response-contract test providers
5. Implement your service logic and contract scenarios
6. Run `loom gen` after every design change

**Best Practice:** Commit generated code to version control rather than generating during CI/CD. This ensures reproducible builds and allows tracking changes in generated code.

For browser and TypeScript consumers, use the endorsed
[`@hey-api/openapi-ts` recipe](typescript-clients.md). It covers the OpenAPI 3.1
compatibility target, generated SDK and validation artifacts, session cookies,
and drift checks.

For published API compatibility, use the endorsed
[oasdiff workflow](openapi-evolution.md). It regenerates the contract, detects
stale committed output, reports a Markdown changelog, and blocks breaking
changes.

---

## Generation Process

When you run `loom gen`, Loom follows a systematic process:

### 1. Bootstrap Phase

Loom creates a temporary `main.go` that:
- Imports Loom packages and your design package
- Runs DSL evaluation
- Triggers code generation

### 2. Design Evaluation

- DSL functions execute to create expression objects
- Expressions combine into a complete API model
- Relationships between expressions are established
- Design rules and constraints are validated

### 3. Code Generation

- Validated expressions pass to code generators
- Templates render to produce code files
- Output writes to the `gen/` directory

---

## Generated Project Structure

A typical project after `loom gen` and `loom example` depends on the transports
used in the design. A service with HTTP and gRPC enabled looks like this:

```
myservice/
├── cmd/                    # Example commands you can customize
│   └── calc/
│       ├── grpc.go
│       └── http.go
├── design/                 # Your design files
│   └── design.go
├── gen/                    # Generated code (don't edit)
│   ├── calc/               # Service-specific code
│   │   ├── client.go
│   │   ├── endpoints.go
│   │   └── service.go
│   ├── http/               # HTTP transport layer
│   │   ├── calc/
│   │   │   ├── client/
│   │   │   └── server/
│   │   ├── openapi.json
│   │   └── openapi.yaml
│   └── grpc/               # gRPC transport layer
│       └── calc/
│           ├── client/
│           ├── server/
│           └── pb/
└── myservice.go            # Your service implementation
```

Small HTTP and JSON-RPC services keep compact `types.go` files. When a
generated transport package grows beyond the type-section split threshold,
Loom writes deterministic concern files such as `types_requests.go`,
`types_responses.go`, `types_unions.go`, `types_validation.go`, and
`types_helpers.go` in the same package. The exported Go API is unchanged; the
split only keeps large generated packages navigable and diff-friendly.

Large generated HTTP and JSON-RPC clients also expose deterministic operation
groups derived from the first route path segment, for example
`client.Items.List()` and `client.Items.BuildListRequest(...)`. The flat
methods remain available on `Client`, so existing consumers keep compiling
while larger services gain a narrower navigation surface.

### Service Interfaces

Generated in `gen/<service>/service.go`:

```go
// Service interface defines the API contract
type Service interface {
    Add(context.Context, *AddPayload) (res int, err error)
    Multiply(context.Context, *MultiplyPayload) (res int, err error)
}

// Payload types
type AddPayload struct {
    A int32
    B int32
}

// Constants for observability
const ServiceName = "calc"
var MethodNames = [2]string{"add", "multiply"}
```

### Endpoint Layer

Generated in `gen/<service>/endpoints.go`:

```go
// Endpoints wraps service methods in transport-agnostic endpoints
type Endpoints struct {
    Add      loom.Endpoint
    Multiply loom.Endpoint
}

// NewEndpoints creates endpoints from service implementation
func NewEndpoints(s Service) *Endpoints {
    return &Endpoints{
        Add:      NewAddEndpoint(s),
        Multiply: NewMultiplyEndpoint(s),
    }
}

// Use applies middleware to all endpoints
func (e *Endpoints) Use(m func(loom.Endpoint) loom.Endpoint) {
    e.Add = m(e.Add)
    e.Multiply = m(e.Multiply)
}
```

Endpoint middleware example:

```go
func LoggingMiddleware(next loom.Endpoint) loom.Endpoint {
    return func(ctx context.Context, req any) (res any, err error) {
        log.Printf("request: %v", req)
        res, err = next(ctx, req)
        log.Printf("response: %v", res)
        return
    }
}

endpoints.Use(LoggingMiddleware)
```

### Client Code

Generated in `gen/<service>/client.go`:

```go
// Client provides typed methods for service calls
type Client struct {
    AddEndpoint      loom.Endpoint
    MultiplyEndpoint loom.Endpoint
}

func NewClient(add, multiply loom.Endpoint) *Client {
    return &Client{
        AddEndpoint:      add,
        MultiplyEndpoint: multiply,
    }
}

func (c *Client) Add(ctx context.Context, p *AddPayload) (res int, err error) {
    ires, err := c.AddEndpoint(ctx, p)
    if err != nil {
        return
    }
    return ires.(int), nil
}
```

---

## HTTP Code Generation

### Server Implementation

Generated in `gen/http/<service>/server/server.go`:

```go
func New(
    e *calc.Endpoints,
    mux loomhttp.Muxer,
    decoder func(*http.Request) loomhttp.Decoder,
    encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
    errhandler func(context.Context, http.ResponseWriter, error),
    formatter func(ctx context.Context, err error) loomhttp.Statuser,
) *Server

// Server exposes handlers for modification
type Server struct {
    Mounts   []*MountPoint
    Add      http.Handler
    Multiply http.Handler
}

// Use applies HTTP middleware to all handlers
func (s *Server) Use(m func(http.Handler) http.Handler)
```

Complete server setup:

```go
func main() {
    svc := calc.New()
    endpoints := gencalc.NewEndpoints(svc)
    mux := loomhttp.NewMuxer()
    server := genhttp.New(
        endpoints,
        mux,
        loomhttp.RequestDecoder,
        loomhttp.ResponseEncoder,
        nil, nil)
    genhttp.Mount(mux, server)
    http.ListenAndServe(":8080", mux)
}
```

### Client Implementation

Generated in `gen/http/<service>/client/client.go`:

```go
func NewClient(
    scheme string,
    host string,
    doer loomhttp.Doer,
    enc func(*http.Request) loomhttp.Encoder,
    dec func(*http.Response) loomhttp.Decoder,
    restoreBody bool,
) *Client
```

Complete client setup:

```go
func main() {
    httpClient := genclient.NewClient(
        "http",
        "localhost:8080",
        http.DefaultClient,
        loomhttp.RequestEncoder,
        loomhttp.ResponseDecoder,
        false,
    )

    client := gencalc.NewClient(
        httpClient.Add(),
        httpClient.Multiply(),
    )

    result, err := client.Add(context.Background(), &gencalc.AddPayload{A: 1, B: 2})
}
```

### Generated CLI Client

HTTP generation also emits command-line client support under
`gen/http/cli/<server>/cli.go` and per-service payload builders under
`gen/http/<service>/client/cli.go`. The generated parser is Kong-backed, so
service and method descriptions from the design become command help, while
Loom-owned generated code still constructs the typed endpoint and payload.

`loom example` wires that support into `cmd/<server>-cli` so local testing can
call any generated endpoint without hand-writing a test client:

```bash
go run ./cmd/calc-cli --url=http://localhost:8080 calc add --a=1 --b=2
go run ./cmd/calc-cli calc add --help
```

---

## gRPC Code Generation

### Protobuf Definition

Generated in `gen/grpc/<service>/pb/`:

```protobuf
syntax = "proto3";
package calc;

service Calc {
    rpc Add (AddRequest) returns (AddResponse);
    rpc Multiply (MultiplyRequest) returns (MultiplyResponse);
}

message AddRequest {
    int64 a = 1;
    int64 b = 2;
}
```

### Server Implementation

```go
func main() {
    svc := calc.New()
    endpoints := gencalc.NewEndpoints(svc)
    svr := grpc.NewServer()
    gensvr := gengrpc.New(endpoints, nil)
    genpb.RegisterCalcServer(svr, gensvr)
    lis, _ := net.Listen("tcp", ":8080")
    svr.Serve(lis)
}
```

### Client Implementation

```go
func main() {
    conn, _ := grpc.Dial("localhost:8080",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    defer conn.Close()

    grpcClient := genclient.NewClient(conn)
    client := gencalc.NewClient(
        grpcClient.Add(),
        grpcClient.Multiply(),
    )

    result, _ := client.Add(context.Background(), &gencalc.AddPayload{A: 1, B: 2})
}
```

---

## Customization

### HTTP Artifact Selection

Loom generates HTTP servers, per-service clients, and the aggregate client CLI
by default. A service that only hosts an HTTP API can generate server packages
without unused client artifacts:

```go
var _ = API("MyAPI", func() {
    Meta("http:generate", "server")
})
```

The `server` mode keeps generated service packages, HTTP server packages, and
OpenAPI output. It omits `gen/http/<service>/client/` and `gen/http/cli/` and
removes those directories when a project switches from the default `all` mode.

### Type Generation Control

Force generation of types not directly referenced by methods:

```go
var MyType = Type("MyType", func() {
    // Force generation in specific services
    Meta("type:generate:force", "service1", "service2")
    
    // Or force generation in all services
    Meta("type:generate:force")
    
    Attribute("name", String)
})
```

### Package Organization

Generate types in a shared package:

```go
var CommonType = Type("CommonType", func() {
    Meta("struct:pkg:path", "types")
    Attribute("id", String)
})
```

Creates:
```
gen/
└── types/
    └── common_type.go
```

### Field Customization

```go
var Message = Type("Message", func() {
    Attribute("id", String, func() {
        // Override field name
        Meta("struct:field:name", "ID")
        
        // Add custom struct tags
        Meta("struct:tag:json", "id,omitempty")
        Meta("struct:tag:msgpack", "id,omitempty")
        
        // Override type
        Meta("struct:field:type", "bson.ObjectId", "github.com/globalsign/mgo/bson", "bson")
    })
})
```

### Protocol Buffer Customization

```go
var MyType = Type("MyType", func() {
    // Override protobuf message name
    Meta("struct:name:proto", "CustomProtoType")
    
    Field(1, "status", Int32, func() {
        // Override protobuf field type
        Meta("struct:field:proto", "int32")
    })

    // Use Google's timestamp type
    Field(2, "created_at", String, func() {
        Meta("struct:field:proto", 
            "google.protobuf.Timestamp",
            "google/protobuf/timestamp.proto",
            "Timestamp",
            "google.golang.org/protobuf/types/known/timestamppb")
    })
})

// Specify protoc include paths
var _ = API("calc", func() {
    Meta("protoc:include", "/usr/include", "/usr/local/include")
})

// Override the protoc command for one gRPC service
var _ = Service("calc", func() {
    Meta("protoc:cmd", "/usr/bin/protoc", "--fatal_warnings")
})
```

### OpenAPI Customization

```go
var _ = API("MyAPI", func() {
    // Control generation
    Meta("openapi:generate", "false")

    // Emit JSON only. Valid values are "json", "yaml", and "both" (default).
    Meta("openapi:output", "json")
    
    // Override the default two-space JSON formatting
    Meta("openapi:json:prefix", "  ")
    Meta("openapi:json:indent", "  ")
    
    // Omit synthesized examples while retaining authored Example(...) values
    Meta("openapi:example", "false")

    // OpenAPI 3.2 is the default. Set 3.1 only for compatibility consumers;
    // the same renderer skips 3.2-only sections.
    Meta("openapi:version", "3.1")

    // OpenAPI 3.2 document identity (omitted by the 3.1 target)
    Meta("openapi:self", "https://example.com/openapi.json")
})

var _ = Service("UserService", func() {
    // Add tags
    HTTP(func() {
        Meta("openapi:tag:Users")
        Meta("openapi:tag:Backend:desc", "Backend API Operations")
    })
    
    Method("CreateUser", func() {
        // Custom operation ID
        Meta("openapi:operationId", "{service}.{method}")
        
        // Custom summary
        Meta("openapi:summary", "Create a new user")
        
        HTTP(func() {
            // Add extensions
            Meta("openapi:extension:x-rate-limit", `{"rate": 100}`)
            POST("/users")
        })
    })
})

var User = Type("User", func() {
    // Override type name in OpenAPI spec
    Meta("openapi:typename", "CustomUser")
})
```

Loom synthesizes deterministic OpenAPI examples by default. Regenerating the
same design produces the same examples, and changing one schema does not shift
examples for unrelated operations or components. Set
`Meta("openapi:example", "false")` at API scope when committed specifications
should omit synthesized examples. Explicit `Example(...)` values remain in both
the JSON and YAML outputs. Loom emits both formats by default. Set
`Meta("openapi:output", "json")` or `Meta("openapi:output", "yaml")` to emit
only one; generation removes a stale unselected sibling file. JSON is
deterministically ordered, formatted with two-space indentation, and terminated
with a newline for readable contract diffs.

---

## Types and Validation

### Validation Enforcement

Loom validates data at system boundaries:
- **Server-side**: Validates incoming requests
- **Client-side**: Validates incoming responses
- **Internal code**: Trusted to maintain invariants

### Pointer Rules for Struct Fields

| Properties | Payload/Result | Request Body (Server) | Response Body (Server) |
|------------|---------------|----------------------|---------------------|
| Required OR Default | Direct (-) | Pointer (*) | Direct (-) |
| Not Required, No Default | Pointer (*) | Pointer (*) | Pointer (*) |

Special types:
- **Objects (structs)**: Always use pointers
- **Arrays and Maps**: Never use pointers (already reference types)

Example:
```go
type Person struct {
    Name     string             // required, direct value
    Age      *int               // optional, pointer
    Hobbies  []string           // array, no pointer
    Metadata map[string]string  // map, no pointer
}
```

### Default Value Handling

- **Marshaling**: Default values initialize nil arrays/maps
- **Unmarshaling**: Default values apply to missing optional fields (not missing required fields)

---

## Views and Result Types

Views control how result types are rendered in responses.

### How Views Work

1. Service method includes a view parameter
2. A views package is generated at the service level
3. View-specific validation is automatically generated

### Server-Side Response

1. Viewed result type is marshalled
2. Nil attributes are omitted
3. View name is passed in "loom-view" header

### Client-Side Response

1. Response is unmarshalled
2. Transformed into viewed result type
3. View name extracted from "loom-view" header
4. View-specific validation performed
5. Converted back to service result type

### Projection Helpers

For `ResultType` and `View` designs, Loom generates exported projection helpers
in the service package. These helpers convert canonical result values to view
types and view values back to canonical result types. They are used by generated
wrappers such as `NewViewed...` and by transport encoders/decoders, which keeps
HTTP, gRPC, and JSON-RPC projections aligned with the canonical result model.
Projection generation is recursive across nested structs, slices, maps, unions,
collections, and optional fields.

View fields inherit canonical requiredness unless the view uses
`ViewRequired(...)` or `ViewOptional(...)`. Those overrides drive projected
validation, HTTP field pointers and JSON tags, and OpenAPI `required` arrays;
nested named views retain their own requiredness contract.

### Default View

If no views are defined, Loom adds a "default" view that includes all basic fields.

---

## Extension Points

Loom exposes a compile-time generator plugin registry for framework extensions
such as Loom-MCP. A plugin registers from a package imported by the design
dependency graph using `codegen.RegisterPluginFirst`, `RegisterPlugin`, or
`RegisterPluginLast`. Each registration can provide a prepare callback that
updates evaluated roots and a generation callback that adds or transforms
files.

Generation snapshots the registry before evaluation phases run, so concurrent
registrations cannot alter an in-progress run. Plugins execute in deterministic
groups (`First`, normal, then `Last`) and by name within each group. Treat this
as a compile-time extension contract: applications should use the design DSL
and runtime middleware for application behavior, while external framework
packages use plugins only for reproducible generated artifacts.

External plugins may populate `codegen.File.SectionTemplates` when their
sections are template-backed. Loom-owned generators use the generic
`codegen.File.Sections` API, but `SectionTemplates` remains a supported public
extension surface and is adapted by `File.AllSections` during rendering.

Framework-owned HTTP, gRPC, and JSON-RPC behavior still belongs directly in the
relevant generator package. Generated section hooks are an implementation
mechanism inside generators, not a second plugin lifecycle.

---

## See Also

- [DSL Reference](dsl-reference.md) — Complete DSL reference for design files
- [HTTP Guide](http-guide.md) — HTTP transport features and customization
- [gRPC Guide](grpc-guide.md) — gRPC transport features and Protocol Buffers
- [Quickstart](quickstart.md) — Getting started with code generation
