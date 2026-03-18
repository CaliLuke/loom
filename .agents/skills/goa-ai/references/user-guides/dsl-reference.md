# DSL REFERENCE

Complete reference for Goa-AI's DSL functions - agents, toolsets, policies, and MCP integration.

## DSL Reference

Complete reference for Goa-AI’s DSL functions. Use it alongside the Runtime guide to understand how designs translate into runtime behavior.

## DSL Quick Reference

Function | Context | Description
---|---|---
Agent Functions | |
`Agent` | `Service` | Defines an LLM-based agent
`Use` | `Agent` | Declares toolset consumption
`Export` | `Agent`, `Service` | Exposes toolsets to other agents
`AgentToolset` | `Use` argument | References toolset from another agent
`UseAgentToolset` | `Agent` | Alias for `AgentToolset` + `Use`
`Passthrough` | `Tool (in Export)` | Deterministic forwarding to service method
`DisableAgentDocs` | `API` | Disables `AGENTS_QUICKSTART.md` generation
Toolset Functions | |
`Toolset` | Top-level | Declares a provider-owned toolset
`FromMCP` | Toolset argument | Configures MCP-backed toolset
`FromRegistry` | Toolset argument | Configures registry-backed toolset
`Description` | Toolset | Sets toolset description
Tool Functions | |
`Tool` | Toolset, Method | Defines a callable tool
`Args` | Tool | Defines input parameter schema
`Return` | Tool | Defines output result schema
`ServerData` | Tool | Defines server-only data schema (never sent to model providers)
`ServerDataDefault` | Tool | Default emission for optional server-data when server_data is omitted or `auto`
`BoundedResult` | Tool | Marks result as bounded view; enforces canonical bounds fields; optional sub-DSL can declare paging cursor fields
`Cursor` | BoundedResult | Declares which payload field carries the paging cursor (optional)
`NextCursor` | BoundedResult | Declares which result field carries the next-page cursor (optional)
`Idempotent` | Tool | Marks tool as idempotent within a run transcript; enables safe cross-transcript de-duplication for identical calls
`Tags` | Tool, Toolset | Attaches metadata labels
`BindTo` | Tool | Binds tool to service method
`Inject` | Tool | Marks fields as runtime-injected
`CallHintTemplate` | Tool | Display template for invocations
`ResultHintTemplate` | Tool | Display template for results
`ResultReminder` | Tool | Static system reminder after tool result
`Confirmation` | Tool | Requires explicit out-of-band confirmation before execution
Policy Functions | |
`RunPolicy` | Agent | Configures execution constraints
`DefaultCaps` | RunPolicy | Sets resource limits
`MaxToolCalls` | DefaultCaps | Maximum tool invocations
`MaxConsecutiveFailedToolCalls` | DefaultCaps | Maximum consecutive failures
`TimeBudget` | RunPolicy | Simple wall-clock limit
`Timing` | RunPolicy | Fine-grained timeout configuration
`Budget` | Timing | Overall run budget
`Plan` | Timing | Planner activity timeout
`Tools` | Timing | Tool activity timeout
`History` | RunPolicy | Conversation history management
`KeepRecentTurns` | History | Sliding window policy
`Compress` | History | Model-assisted summarization
`Cache` | RunPolicy | Prompt caching configuration
`AfterSystem` | Cache | Checkpoint after system messages
`AfterTools` | Cache | Checkpoint after tool definitions
`InterruptsAllowed` | RunPolicy | Enable pause/resume
`OnMissingFields` | RunPolicy | Validation behavior
MCP Functions | |
`MCPServer` | Service | Enables MCP support
`MCP` | Service | Alias for `MCPServer`
`ProtocolVersion` | MCP option | Sets MCP protocol version
`MCPTool` | Method | Marks method as MCP tool
`MCPToolset` | Top-level | Declares MCP-derived toolset
`Resource` | Method | Marks method as MCP resource
`WatchableResource` | Method | Marks method as subscribable resource
Registry Functions | |
`Registry` | Top-level | Declares a registry source
`URL` | Registry | Sets registry endpoint
`APIVersion` | Registry | Sets API version
`Timeout` | Registry | Sets HTTP timeout
`Retry` | Registry | Configures retry policy
`SyncInterval` | Registry | Sets catalog refresh interval
`CacheTTL` | Registry | Sets local cache duration
`Federation` | Registry | Configures external registry imports
`Include` | Federation | Glob patterns to import
`Exclude` | Federation | Glob patterns to skip
`PublishTo` | Export | Configures registry publication
`Version` | Toolset | Pins registry toolset version
Schema Functions | |
`Attribute` | Args, Return, ServerData | Defines schema field (general use)
`Field` | Args, Return, ServerData | Defines numbered proto field (gRPC)
`Required` | Schema | Marks fields as required

## Prompt Management (v1 Integration Path)

Goa-AI v1 does not require a dedicated prompt DSL (`Prompt(...)`, `Prompts(...)`).
Prompt management is currently runtime-driven:

- Register baseline prompt specs in `runtime.PromptRegistry`.
- Configure scoped overrides with `runtime.WithPromptStore(...)`.
- Render prompts from planners using `PlannerContext.RenderPrompt(...)`.
- Attach rendered prompt provenance to model calls with `model.Request.PromptRefs`.

For agent-as-tool flows, map tool IDs to prompt IDs using options like `runtime.WithPromptSpec(...)` on agent-tool registrations.
This is optional: when no consumer-side prompt content is configured, the runtime
uses the canonical JSON tool payload as the nested user message.

### Field vs Attribute

Both `Field` and `Attribute` define schema fields:

- `Attribute(name, type, description, dsl)` for JSON-only schemas.
- `Field(number, name, type, description, dsl)` for gRPC/protobuf with stable field numbers.

## Overview

Goa-AI extends Goa’s DSL with functions for declaring agents, toolsets, and runtime policies. The DSL is evaluated by Goa’s eval engine, so normal Goa rules apply.

## Import Path

```go
import (
    . "goa.design/goa/v3/dsl"
    . "goa.design/goa-ai/dsl"
)
```

## Entry Point

Declare agents inside a regular Goa `Service` definition.

## Outcome

Running `goa gen` produces:

- Agent packages under `gen/<service>/agents/<agent>` with workflow definitions and registration helpers
- Toolset owner packages under `gen/<service>/toolsets/<toolset>`
- Activity handlers for plan/execute loops
- Registration helpers

`AGENTS_QUICKSTART.md` is written unless disabled by `DisableAgentDocs()`.

## Quickstart Example

```go
package design

import (
    . "goa.design/goa/v3/dsl"
    . "goa.design/goa-ai/dsl"
)

var DocsToolset = Toolset("docs.search", func() {
    Tool("search", "Search indexed documentation", func() {
        Args(func() {
            Attribute("query", String, "Search phrase")
            Attribute("limit", Int, "Max results", func() { Default(5) })
            Required("query")
        })
        Return(func() {
            Attribute("documents", ArrayOf(String), "Matched snippets")
            Required("documents")
        })
        Tags("docs", "search")
    })
})

var AssistantSuite = MCPToolset("assistant", "assistant-mcp")

var _ = Service("orchestrator", func() {
    Description("Human front door for the knowledge agent.")

    Agent("chat", "Conversational runner", func() {
        Use(DocsToolset)
        Use(AssistantSuite)
        Export("chat.tools", func() {
            Tool("summarize_status", "Produce operator-ready summaries", func() {
                Args(func() {
                    Attribute("prompt", String, "User instructions")
                    Required("prompt")
                })
                Return(func() {
                    Attribute("summary", String, "Assistant response")
                    Required("summary")
                })
                Tags("chat")
            })
        })
        RunPolicy(func() {
            DefaultCaps(
                MaxToolCalls(8),
                MaxConsecutiveFailedToolCalls(3),
            )
            TimeBudget("2m")
        })
    })
})
```

Running `goa gen example.com/assistant/design` produces:

- `gen/orchestrator/agents/chat` workflow and planner activities.
- `gen/orchestrator/agents/chat/specs` tool catalog.
- Toolset and exported toolset packages.
- MCP-aware registration helpers when an MCP toolset is referenced.

## Typed Tool Identifiers

```go
const (
    Search tools.Ident = "orchestrator.search.search"
)
```

## Cross-Process Inline Composition

Agent composition creates run trees automatically. Child runs are linked with `run.Handle` and surfaced with child-run events.

## Agent Functions

### `Agent(name, description, dsl)`

Declares an agent inside a `Service`.

### `Use(value, dsl)`

Consumes toolsets:

- top-level `Toolset`
- `MCPToolset`
- inline toolset definition
- `AgentToolset`

### `Export(value, dsl)`

Declares exported toolsets for other agents.

### `AgentToolset(service, agent, toolset)` and `UseAgentToolset(...)`

References an exported toolset from another agent.

### `Passthrough(toolName, target, methodName)`

Deterministic forwarding to Goa service method.

### `DisableAgentDocs()`

Disables generation of `AGENTS_QUICKSTART.md`.

## Toolset Functions

### `Toolset(name, dsl)`

Top-level reusable provider-owned toolset.

### `FromMCP`, `FromRegistry`

Configure MCP-backed or registry-backed sourcing.

### `Tool(name, description, dsl)`

Defines callable capability with args/return, hints, metadata.

### `Args(...)` and `Return(...)`

Schema for payload and result.

### `ServerData(kind, val, args...)`

Typed server-only payload for observers; never sent to model.

### `ServerDataDefault(mode)`

Default optional emission mode `on` / `off`.

### `BoundedResult()`

Marks bounded results and canonical bounds contract.

### `Idempotent()`

Marks calls as transcript-safe retries candidates.

### `Confirmation(...)`

Require human approval before execution.

### `CallHintTemplate(...)` / `ResultHintTemplate(...)`

UI-facing hint templates.

### `ResultReminder(...)`

Static reminder appended after tool result.

### `Tags(...)`

Toolset/tool labels.

### `BindTo(...)`

Bind to Goa method.

### `Inject(...)`

Mark runtime-injected fields.

## Policy Functions

### `RunPolicy`, `DefaultCaps`, `MaxToolCalls`, `MaxConsecutiveFailedToolCalls`

### `TimeBudget`, `Timing(Budget, Plan, Tools)`

### `History(KeepRecentTurns|Compress)`

### `Cache(AfterSystem, AfterTools)`

### `InterruptsAllowed`

### `OnMissingFields`

## MCP Functions

### `MCPServer` / `MCP`

Enables MCP support on a service.

### `ProtocolVersion`

Configure MCP protocol version.

### `MCPTool`, `MCPToolset`, `Resource`, `WatchableResource`, `StaticPrompt`, `DynamicPrompt`, `Notification`, `Subscription`, `SubscriptionMonitor`

## Registry Functions

### `Registry`

Define registry sources and federation.

### `FromRegistry`, `Version`, `PublishTo`

Pin and publish registry-backed toolsets.

