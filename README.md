<p align="center">
  <a href="https://github.com/CaliLuke/loom/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/CaliLuke/loom.svg?style=for-the-badge"></a>
  <a href="https://pkg.go.dev/github.com/CaliLuke/loom"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge"></a>
  <a href="https://github.com/CaliLuke/loom/actions/workflows/ci.yml"><img alt="GitHub Action: Test" src="https://img.shields.io/github/actions/workflow/status/CaliLuke/loom/ci.yml?branch=main&style=for-the-badge"></a>
  <a href="https://goreportcard.com/report/github.com/CaliLuke/loom"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/CaliLuke/loom?style=for-the-badge"></a>
  <a href="/LICENSE"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge"></a>
  <a href="https://github.com/CaliLuke/loom/discussions"><img alt="Discussions" src="https://img.shields.io/badge/Community-Discussions-0285FF?style=for-the-badge"></a>
</p>

# Loom - The AI-First API Framework

## Overview

Loom is a design-first API framework pushed in an AI-first direction: stronger machine-consumable contracts, less app-local glue, and transport generation that holds up under automation. It was shaped to support the creation of Auto-K and the style of development that came with it: agent-assisted iteration, contract-first generation, and downstream tooling that depends on trustworthy specs.

Loom is design-first. You describe the service once in a Go DSL, then generate transports, clients, docs, and scaffolding from that contract. The difference is where Loom puts pressure on the framework: OpenAPI 3.1 quality, reusable public contract components, JSON-RPC and streaming behavior, session/auth ergonomics, and repo conventions that make AI-assisted development materially easier.

## Why Loom?

Traditional API development breaks down in two places: teams end up hand-maintaining transport glue, and the published spec is too weak to safely drive codegen, SDKs, and agents. Loom exists to fix both.

Loom is designed for teams that want:
- one design source of truth for service behavior, transport shape, and published contract
- OpenAPI output strong enough to feed downstream client generators directly
- framework-owned solutions for recurring glue instead of per-app patches
- a repo and workflow that cooperate with AI-assisted implementation instead of fighting it

## Key Features

- **Expressive Design Language**: Define your API with a clear, type-safe DSL that captures your intent
- **Comprehensive Code Generation**:
  - Type-safe server interfaces that enforce your design
  - Client packages with full error handling
  - Transport layer adapters (HTTP/gRPC/JSON-RPC) with routing and encoding
  - OpenAPI 3.1 documentation that's always in sync
  - CLI tools for testing your services
- **Multi-Protocol Support**: Generate HTTP REST, gRPC, and JSON-RPC endpoints from a single design
- **Clean Architecture**: Business logic remains separate from transport concerns
- **Enterprise Ready**: Supports authentication, authorization, CORS, logging, and more
- **Comprehensive Testing**: Includes extensive unit and integration test suites ensuring quality and reliability

## Why Loom?

Loom currently emphasizes:
- **AI-first development**: the repo ships with its own Loom skill for AI-assisted service work, plus repo conventions and generated-contract rules meant to keep agents on the rails instead of patching around framework gaps
- **Auto-K-driven framework design**: many capabilities were added because Auto-K needed them in the framework, not as one-off application glue
- **Stronger OpenAPI 3.1 as a product surface**: Loom treats the OpenAPI document as a machine contract, not a byproduct. The generator validates with `libopenapi`, lints with Redocly, and smoke-tests downstream generation with `openapi-typescript` and `oapi-codegen`
- **Better reusable public contracts**: repeated parameters, request bodies, responses, examples, and schemas are hoisted and named more deliberately so generated specs are easier to consume and diff
- **More truthful request/response schema publication**: `readOnly` and `writeOnly` metadata split public request and response schemas when they should not share one shape
- **Standards-oriented error contracts**: Loom HTTP defaults now use RFC 9457-style `application/problem+json` problem documents with stable machine-readable codes, and the DSL can still override public problem `type`/`title` metadata when needed
- **First-class async contract publication**: SSE and WebSocket endpoints publish framework-owned async metadata in OpenAPI so downstream tooling can reason about stream payloads and handshake behavior
- **JSON-RPC as a real transport, not an afterthought**: Loom treats JSON-RPC and JSON-RPC SSE as framework-owned behavior with dedicated generation and integration coverage
- **Less application glue**: session auth, auth error reuse, response links, form and multipart request support, observability hooks, and transport-specific contract controls live in the framework instead of being repeatedly rebuilt in services

The short version: Loom is optimized for AI-assisted service development and machine-grade contracts.

## Built For Auto-K

Auto-K was one of the forcing functions behind Loom. The framework was pushed to absorb repeated infrastructure and contract concerns that would otherwise have remained application-local:
- cleaner auth/session modeling
- better generated OpenAPI for downstream automation
- stronger streaming and JSON-RPC behavior
- more reusable public contract components
- better direct seam tests for generator behavior

That matters beyond Auto-K. The same work makes Loom better for any codebase that wants to generate clients directly from the spec, keep transport behavior honest, and let AI tools operate on a clearer contract surface.

## AI Assistance

Loom comes with its own repository skill for AI-assisted development at [.agents/skills/loom/SKILL.md](./.agents/skills/loom/SKILL.md). It documents the framework contract rules, generation workflow, OpenAPI behavior, and the repo-specific guardrails an agent needs to make useful changes without thrashing generated code.

This is part of the product, not an afterthought. Loom is intentionally shaped so both humans and agents can work from the same design, the same generated surfaces, and the same published contract.

## How It Works

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────────┐
│ Design API  │────>│ Generate Code│────>│ Implement Business  │
│ using DSL   │     │ & Docs       │     │ Logic               │
└─────────────┘     └──────────────┘     └─────────────────────┘
```

1. **Design**: Express your API's intent in Loom's DSL
2. **Generate**: Run `loom gen` to create server interfaces, client code, and documentation
3. **Implement**: Focus solely on writing your business logic in the generated interfaces
4. **Evolve**: Update your design and regenerate code as your API evolves

## Quick Start

```bash
# Install Loom
go install github.com/CaliLuke/loom/cmd/loom@latest

# Install the current stable release explicitly
go install github.com/CaliLuke/loom/cmd/loom@v1.0.9

# Create a new module
mkdir hello && cd hello
go mod init hello

# Define a service in design/design.go
mkdir design
cat > design/design.go << EOF
package design

import . "github.com/CaliLuke/loom/dsl"

var _ = Service("hello", func() {
    Method("say_hello", func() {
        Payload(func() {
            Field(1, "name", String)
            Required("name")
        })
        Result(String)

        HTTP(func() {
            GET("/hello/{name}")
        })
    })
})
EOF

# Generate the code
loom gen hello/design
loom example hello/design

# Build and run
go mod tidy
go run cmd/hello/*.go --http-port 8000

# In another terminal
curl http://localhost:8000/hello/world
```

The example above:
1. Defines a simple "hello" service with one method
2. Generates server and client code
3. Starts a server that logs requests server-side (without displaying any client output)

### JSON-RPC Alternative

For a JSON-RPC service, simply add a `JSONRPC` expression to the service and
method:

```go
var _ = Service("hello" , func() {
    JSONRPC(func() {
        Path("/jsonrpc")
    })
    Method("say_hello", func() {
        Payload(func() {
            Field(1, "name", String)
            Required("name")
        })
        Result(String)

        JSONRPC(func() {})
    })
}
```

Then test with:
```bash
curl -X POST http://localhost:8000/jsonrpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"hello.say_hello","params":{"name":"world"},"id":"1"}'
```

## Documentation

The repository is the current source of truth for framework code, migration history, and transport behavior:

- **[Repository](https://github.com/CaliLuke/loom)**: Framework source, issues, releases, and discussions
- **[Roadmap](https://github.com/CaliLuke/loom/tree/main/roadmap)**: Active framework direction and remaining work
- **[JSON-RPC Architecture](https://github.com/CaliLuke/loom/blob/main/jsonrpc/ARCHITECTURE.md)**: Transport architecture notes
- **[Integration Tests](https://github.com/CaliLuke/loom/tree/main/http/integration_tests)**: End-to-end fixture and smoke coverage

## Development

Install the tracked git hooks with:

```bash
make install-hooks
```

The repo-managed pre-push hook runs `make lint`, and `make lint` now includes the `dupl` linter so duplicated Go code is blocked before push.

## Real-World Examples

The examples package is being rebuilt under Loom branding. Until that lands, the checked-in fixtures and integration tests in this repo are the most accurate working references.

- **Basic**: Simple service showcasing core Loom concepts
- **Cellar**: A more complete REST API example
- **Cookies**: HTTP cookie management
- **Encodings**: Working with different content types
- **Error**: Comprehensive error handling strategies
- **Files & Upload/Download**: File handling capabilities
- **HTTP Status**: Custom status code handling
- **Interceptors**: Request/response processing middleware
- **Multipart**: Handling multipart form submissions
- **Security**: Authentication and authorization examples
- **Streaming**: Implementing streaming endpoints (HTTP, WebSocket, JSON-RPC SSE)
- **Tracing**: Integrating with observability tools
- **TUS**: Resumable file uploads implementation

## Community & Support

- Ask questions on [GitHub Discussions](https://github.com/CaliLuke/loom/discussions)
- Report issues on [GitHub](https://github.com/CaliLuke/loom/issues)
- Follow releases on [GitHub Releases](https://github.com/CaliLuke/loom/releases)
- Watch the repo and release feed if you want rename and migration updates as Loom continues to diverge.

## License

MIT License - see [LICENSE](LICENSE) for details.
