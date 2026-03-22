<p align="center">
  <a href="https://github.com/CaliLuke/loom/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/CaliLuke/loom.svg?style=for-the-badge"></a>
  <a href="https://pkg.go.dev/github.com/CaliLuke/loom/v3"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge"></a>
  <a href="https://github.com/CaliLuke/loom/actions/workflows/ci.yml"><img alt="GitHub Action: Test" src="https://img.shields.io/github/actions/workflow/status/CaliLuke/loom/ci.yml?branch=main&style=for-the-badge"></a>
  <a href="https://goreportcard.com/report/github.com/CaliLuke/loom"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/CaliLuke/loom?style=for-the-badge"></a>
  <a href="/LICENSE"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge"></a>
  <a href="https://github.com/CaliLuke/loom/discussions"><img alt="Discussions" src="https://img.shields.io/badge/Community-Discussions-0285FF?style=for-the-badge"></a>
</p>

# Loom - Design First, Code With Confidence

## Overview

Loom transforms how you build APIs and microservices in Go with its powerful design-first approach. Instead of writing boilerplate code, you express your API's intent through a clear, expressive DSL. Loom then automatically generates production-ready code, comprehensive documentation, and client libraries—all perfectly aligned with your design.

The result? Dramatically reduced development time, consistent APIs, and the elimination of the documentation-code drift that plagues traditional development.

## Why Loom?

Traditional API development suffers from:
- **Inconsistency**: Manually maintained docs that quickly fall out of sync with code
- **Wasted effort**: Writing repetitive boilerplate and transport-layer code
- **Painful integrations**: Client packages that need constant updates
- **Design afterthoughts**: Documentation added after implementation, missing key details

Loom solves these problems by:
- Generating 30-50% of your codebase directly from your design
- Ensuring perfect alignment between design, code, and documentation
- Supporting multiple transports (HTTP, gRPC, and JSON-RPC) from a single design
- Maintaining a clean separation between business logic and transport details

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
go install github.com/CaliLuke/loom/v3/cmd/loom@latest

# Create a new module
mkdir hello && cd hello
go mod init hello

# Define a service in design/design.go
mkdir design
cat > design/design.go << EOF
package design

import . "github.com/CaliLuke/loom/v3/dsl"

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
