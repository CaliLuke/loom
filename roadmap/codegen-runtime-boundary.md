# Generated Transport Runtime Boundary

Status: active. This design tracks issue #267 and its transport-specific work.

## Decision

Generated transport code contains contract data and typed adapters. Handwritten
transport packages own stable protocol execution.

Generated code can bind service types through functions and interfaces. Runtime
code must not use broad reflection to inspect application values.

The first prototype moves HTTP handler adaptation and route registration into
the `http` runtime package. Generated route functions now call
`loomhttp.MountHandler` with an `http.Handler`.

## Evidence

The measurements use the repository fixtures and the current Go toolchain.
Each count includes all generated Go files in the fixture.

| Fixture | All Go lines before | All Go lines after | Server Go lines before | Server Go lines after |
| --- | ---: | ---: | ---: | ---: |
| HTTP ticktock | 1,981 | 1,963 | 780 | 762 |
| HTTP quality | 1,900 | 1,888 | 634 | 622 |
| gRPC quality | 3,152 | 3,152 | 607 | 607 |
| JSON-RPC ticktock | 2,350 | 2,350 | 933 | 933 |
| JSON-RPC mixedtick | 1,982 | 1,982 | 873 | 873 |

The HTTP ticktock `server.go` file decreases from 342 lines to 324 lines.
The HTTP quality `server.go` file decreases from 218 lines to 206 lines.

The HTTP quality generated packages compile before and after the prototype.
A cold-cache compile took 5.78 seconds before and 5.88 seconds after.
This 0.10-second difference is measurement noise, not a compile-time change.

The `http` package statement coverage stays at 78.0 percent. The new runtime
path has a direct dispatch test and generated-output coverage.

### Unary HTTP phase

Issue #280 moves the ordinary unary request lifecycle into `NewUnaryHandler`.
Typed generated callbacks retain the exact payload and result types.

The HTTP quality total decreases from 1,888 lines to 1,865 lines after this
phase. Its server total decreases from 622 lines to 599 lines. Its `server.go`
file decreases from 206 lines to 183 lines.

The total decrease from the original HTTP quality baseline is 35 lines. The
generated-quality gate compiles and analyzes all five fixture groups.

The direct unary lifecycle tests increase `http` package coverage from 78.0
percent to 78.4 percent.

### HTTP stream and raw-body phase

Issue #281 moves shared HTTP handler state into `HandlerLifecycle`. This runtime
type owns request observation and error routing before and after response commit.

The runtime also owns raw-body writes, file serving, and content cleanup.
Generated code retains typed result and stream adapters.

The HTTP ticktock total decreases from 1,963 lines to 1,930 lines after this
phase. Its server total decreases from 762 lines to 729 lines. Its `server.go`
file decreases from 324 lines to 291 lines.

The HTTP quality fixture has no stream, file, or raw-body endpoint. Its line
counts remain 1,865 total, 599 server, and 183 in `server.go`.

Direct runtime tests cover pre-commit errors, post-commit errors, cleanup
failures, and late raw-body write failures. The SSE and WebSocket adversarial
integration tests also cover the generated adapters.

The `http` package statement coverage is 78.3 percent after this phase.

Use these commands to repeat the checks:

```bash
make generated-code-quality
go test ./http ./http/codegen
```

## Ownership Boundary

Handwritten runtime packages own these behaviors:

- route registration and handler adaptation
- request context and transport observation
- protocol-level error routing
- response commit and write-failure behavior
- stream open, close, and completion behavior
- transport setup that does not depend on service types

Generated packages own these values and adapters:

- service payload, result, error, and stream types
- typed service invocation
- request and response transformations
- route, status, header, and content-type declarations
- design-declared application policy
- response contract manifests

This boundary keeps generated Go readable. It also keeps compile-time checks at
the service boundary.

## Runtime API Shape

The prototype uses this runtime call:

```go
loomhttp.MountHandler(mux, http.MethodGet, "/items", handler)
```

`MountHandler` accepts `http.Handler`. The Go compiler checks the handler type,
and the runtime does not inspect the handler with reflection.

The unary HTTP phase uses typed callbacks with this general shape:

```go
type UnaryHandlerSpec[Payload, Result any] struct {
    Service       string
    Method        string
    Decode        func(*http.Request) (Payload, error)
    Invoke        func(context.Context, Payload) (Result, error)
    Encode        func(context.Context, http.ResponseWriter, Result) error
    EncodeError   func(context.Context, http.ResponseWriter, error) error
    HandleFailure func(context.Context, http.ResponseWriter, error)
}
```

Generated code supplies each callback. Runtime code controls the protocol
sequence and reports failures through the shared observation contract.

Other HTTP handlers create `HandlerLifecycle` and pass typed closures to its
raw-body and file helpers. SSE and WebSocket adapters report their commit state
to `HandlerFailed`.

The runtime helpers do not accept an untyped service registry.

## Compatibility Contract

Generated code and the runtime use normal Go symbol compatibility. New
generated code fails at compile time against a runtime that lacks its helper.

Runtime helpers stay compatible for the rest of the Loom v1 module. A breaking
helper change requires a new API name or a new major module version.

Generated files do not need a hidden runtime version check. The Go compiler
reports missing or incompatible helper symbols directly.

A runtime fix affects existing generated code only after that code delegates to
the runtime helper. Older generated code keeps its embedded behavior until the
application runs `loom gen` again.

## Plugin Boundary

Plugins can add contract data, typed adapters, middleware, or codecs. Plugins
must not copy the runtime request loop into generated templates.

Each extension point uses a narrow transport interface. The interface states
its error, context, and stream ownership.

Application policy stays outside the runtime. Examples include authorization
decisions, fixtures, error content, and retry policy.

## Delivery Sequence

1. HTTP route mounting uses the runtime helper in this prototype.
2. Issue #280 moves unary HTTP handler lifecycle into the runtime. The phase
   is implemented on its ticket branch.
3. Issue #281 moves HTTP stream and raw-body lifecycle into the runtime. The
   phase is implemented on its ticket branch.
4. Issue #282 moves JSON-RPC dispatch lifecycle into the runtime.
5. Issue #283 moves gRPC metadata and status lifecycle into the runtime.

Each phase records line counts for the same fixtures. Each phase also runs the
transport integration tests and `make generated-code-quality`.

## Exit Criteria

Issue #267 is complete when this prototype and design are merged. The four
transport tickets own the staged implementation after that merge.
