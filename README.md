<p align="center">
  <a href="https://github.com/CaliLuke/loom/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/CaliLuke/loom?style=for-the-badge"></a>
  <a href="https://pkg.go.dev/github.com/CaliLuke/loom"><img alt="Go reference" src="https://pkg.go.dev/badge/github.com/CaliLuke/loom.svg"></a>
  <a href="https://go.dev/dl/"><img alt="Go version" src="https://img.shields.io/github/go-mod/go-version/CaliLuke/loom?style=for-the-badge"></a>
  <a href="https://github.com/CaliLuke/loom/actions/workflows/test.yml"><img alt="Build status" src="https://img.shields.io/github/actions/workflow/status/CaliLuke/loom/test.yml?branch=main&style=for-the-badge&label=build"></a>
  <a href="https://github.com/CaliLuke/loom/actions/workflows/codeql.yml"><img alt="CodeQL status" src="https://img.shields.io/github/actions/workflow/status/CaliLuke/loom/codeql.yml?branch=main&style=for-the-badge&label=CodeQL"></a>
  <a href="https://deepwiki.com/CaliLuke/loom"><img alt="Ask DeepWiki" src="https://deepwiki.com/badge.svg"></a>
  <a href="./LICENSE"><img alt="MIT license" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge"></a>
</p>

# Loom

Loom is a design-first Go framework that turns one API definition into service
interfaces, HTTP, gRPC, and JSON-RPC transports, type-safe clients, CLIs, and
OpenAPI 3.2 contracts with an optional OpenAPI 3.1 compatibility target.

[Quick start](#quick-start) · [Documentation](docs/_index.md) ·
[Go reference](https://pkg.go.dev/github.com/CaliLuke/loom) ·
[Ask DeepWiki](https://deepwiki.com/CaliLuke/loom)

## Why Loom?

- **Contracts you can build against.** Loom emits OpenAPI 3.2.0 by default from
  one shared contract model. Set `Meta("openapi:version", "3.1")` to make the
  same renderer omit 3.2-only members while keeping the canonical output paths.
  Representative contracts are parsed with `libopenapi`, linted
  with Redocly, and compiled through Hey API and `oapi-codegen` in
  [consumer smoke tests](http/codegen/openapi/v3/contract_smoke_test.go#L140).
- **One design, multiple transports.** The same service model drives HTTP,
  gRPC, and JSON-RPC servers and clients. SSE and WebSocket endpoints publish
  explicit message and handshake metadata under `x-loom-async`.
- **Framework-owned API behavior.** Validation, authentication, CORS,
  RFC 9457 problem responses, streaming, and transport observability are
  modeled once instead of being rebuilt around every handler.
- **Repeatable generation.** `loom gen` stages and validates the complete output
  before replacing generated files. Humans and coding agents follow the same
  design → generate → implement workflow.

## How it works

| You write | Loom generates |
| --- | --- |
| `design/*.go` API definitions | Service interfaces and endpoint wrappers |
| Business logic | HTTP, gRPC, and JSON-RPC servers and clients |
| Application wiring and tests | Request validation, CLIs, and transport code |
| Transport policy in the DSL | OpenAPI 3.2, optional 3.1 compatibility, and Protocol Buffer definitions |

Design files are the source of truth. Generated files live under `gen/`; your
business logic stays in ordinary, non-generated Go files.

## Loom and Goa

Loom was [derived from Goa](LICENSE#L5) and retains its design-first model:
describe the service in a Go DSL, generate the transport layer, and implement
the resulting service interface.

Loom is intended for teams that specifically need:

- OpenAPI 3.2 with a version-gated 3.1 compatibility target as a tested machine-facing contract;
- reusable contract components, request/response schema separation, and
  explicit async metadata;
- RFC 9457 HTTP errors and framework-owned session, CORS, streaming, and
  observability behavior;
- transactional code generation and compile-time generator extensions.

[Goa](https://goa.design/) remains the original project with the
larger established community. If Loom's contract and transport guarantees are
not requirements, Goa may be the better fit. Loom is actively diverging and
does not promise source compatibility with every Goa release.

## Quick start

Loom requires the Go version declared in [go.mod](go.mod#L3), currently Go
1.27 or later.

Install the CLI and create a module:

```bash
go install github.com/CaliLuke/loom/cmd/loom@v1.9.0-alpha.10

mkdir hello && cd hello
go mod init example.com/hello
go get github.com/CaliLuke/loom@v1.9.0-alpha.10
mkdir design
```

Create `design/design.go`:

```go
package design

import . "github.com/CaliLuke/loom/dsl"

var _ = Service("hello", func() {
	Method("greet", func() {
		Payload(func() {
			Field(1, "name", String, "Name to greet")
			Required("name")
		})
		Result(String)

		HTTP(func() {
			GET("/hello/{name}")
		})
	})
})
```

Generate the service and starter implementation:

```bash
loom gen example.com/hello/design
loom example example.com/hello/design
go mod tidy
```

`loom example` writes consumer-owned stubs. Each stub returns `loom.Fault`
until you replace it, so an unfinished route cannot report success.

In the generated starter file `hello.go`, replace the `Greet` method with:

```go
func (s *hellosrvc) Greet(ctx context.Context, p *hello.GreetPayload) (string, error) {
	log.Printf(ctx, "hello.greet")
	return "Hello, " + p.Name + "!", nil
}
```

Run the service:

```bash
go run ./cmd/hello --http-port=8000
```

Then call it from another terminal:

```bash
$ curl http://localhost:8000/hello/Ada
"Hello, Ada!"
```

The same generation run creates a type-safe client and
`gen/http/openapi.{json,yaml}`. Continue with the
[guided quickstart](docs/quickstart.md) or explore the
[generated-code workflow](docs/code-generation.md).

## Documentation

- **Design:** [DSL reference](docs/dsl-reference.md) and
  [code generation](docs/code-generation.md)
- **Transports:** [HTTP](docs/http-guide.md), [gRPC](docs/grpc-guide.md), and
  [JSON-RPC](jsonrpc/README.md)
- **Operations:** [error handling](docs/error-handling.md),
  [interceptors](docs/interceptors.md), and
  [production guidance](docs/production.md)
- **Agent-assisted development:**
  [Loom skill](.agents/skills/loom/SKILL.md)

## Project expectations

Loom is a code-generation framework: adopting it means keeping the design as
the source of truth, regenerating after design changes, and committing
generated code. If you prefer handwritten transport handlers or a schema-first
workflow based on existing OpenAPI or Protocol Buffer files, Loom is probably
not the right abstraction.

For releases and project activity, see [GitHub Releases](https://github.com/CaliLuke/loom/releases)
and [GitHub Issues](https://github.com/CaliLuke/loom/issues). Contributions are
welcome; start with [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Loom is available under the [MIT License](LICENSE).
