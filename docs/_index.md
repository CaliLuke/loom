---
title: "Loom Framework"
linkTitle: "Loom"
weight: 1
description: "Design-first API development with automatic code generation for Go services."
llm_optimized: true
aliases:
---

## Overview

Loom is a design-first framework for building services in Go. Define your API
once using Loom's DSL, then generate service interfaces, endpoint wrappers,
transport adapters, clients, and machine-consumable contracts from that design.
Loom emphasizes HTTP, JSON-RPC, streaming behavior, stronger OpenAPI 3.1
output, and continued gRPC support.

### Key Features

- **Design-First** — Define your API using a powerful DSL before writing implementation code
- **Code Generation** — Automatically generate server, client, and documentation code
- **Type Safety** — End-to-end type safety from design to implementation
- **Multi-Transport** — Support for HTTP, JSON-RPC, and gRPC from a single design
- **Validation** — Built-in request validation based on your design
- **Contracts** — Auto-generated OpenAPI 3.1 specifications for HTTP APIs

## How Loom Works

Loom follows a three-phase workflow that separates API design from implementation, ensuring consistency and reducing boilerplate.

Design files are the source of truth. Generated code belongs under `gen/`; your
service implementation, application wiring, and tests stay outside `gen/`.

### Phase 1: Design (You Write)

In the design phase, you define your API using Loom's DSL in Go files (typically in a `design/` directory):

- **Types**: Define data structures with validation rules
- **Services**: Group related methods together
- **Methods**: Define operations with payloads and results
- **Transports**: Map methods to HTTP endpoints, JSON-RPC methods, and/or gRPC procedures
- **Security**: Define authentication and authorization schemes

**What you create**: `design/*.go` files containing your API specification as Go code.

### Phase 2: Generate (Automated)

Run `loom gen` to generate transport, service, and contract code:

```bash
loom gen myservice/design
```

**What Loom creates** (in the `gen/` directory):
- Service interfaces and endpoint wrappers
- Server adapters with request routing and validation
- Type-safe client libraries
- OpenAPI 3.1 specifications for HTTP APIs
- JSON-RPC transport code when `JSONRPC` is used
- Protocol Buffer definitions (for gRPC)
- Transport encoders/decoders

**Important**: Never edit files in `gen/` — they are regenerated each time you run `loom gen`.

### Phase 3: Implement (You Write)

Write your business logic by implementing the generated service interfaces:

```go
// service.go - You write this
type helloService struct{}

func (s *helloService) SayHello(ctx context.Context, name string) (string, error) {
    return fmt.Sprintf("Hello, %s!", name), nil
}
```

**What you create**: Service implementation files that contain your actual business logic.

### What's Hand-Written vs Auto-Generated

| You Write | Loom Generates |
|-----------|---------------|
| `design/*.go` — API definitions | `gen/` — All transport code |
| `service.go` — Business logic | OpenAPI 3.1 specifications |
| `cmd/*/main.go` — Server startup | Protocol Buffer definitions |
| Tests and custom middleware | Request validation |

## Documentation Guides

| Guide | Description |
|-------|-------------|
| [Quickstart](quickstart.md) | Install Loom and build your first service |
| [DSL Reference](dsl-reference.md) | Complete reference for Loom's design language |
| [Code Generation](code-generation.md) | Understanding Loom's code generation process |
| [HTTP Guide](http-guide.md) | HTTP transport features, routing, and patterns |
| [gRPC Guide](grpc-guide.md) | gRPC transport features and streaming |
| [JSON-RPC Guide](../jsonrpc/README.md) | JSON-RPC transport, batching, WebSocket, and SSE patterns |
| [Error Handling](error-handling.md) | Defining and handling errors |
| [Interceptors](interceptors.md) | Interceptors and middleware patterns |
| [Production](production.md) | Observability, security, and deployment |
| [Pulse Lifecycle](pulse.md) | Redis-backed maps, streams, pools, and shutdown contracts |

## Quick Example

```go
package design

import . "github.com/CaliLuke/loom/dsl"

var _ = Service("hello", func() {
    Method("sayHello", func() {
        Payload(String, "Name to greet")
        Result(String, "Greeting message")
        HTTP(func() {
            GET("/hello/{name}")
        })
    })
})
```

Generate and run:

```bash
loom gen hello/design
loom example hello/design
go run ./cmd/hello
```

## Getting Started

Start with the [Quickstart](quickstart.md) guide to install Loom and build your first service in minutes.

For comprehensive DSL coverage, see the [DSL Reference](dsl-reference.md).
