MCP INTEGRATION
===============

Integrate external MCP servers into your agents with generated wrappers and callers.

MCP Integration
===============

Goa-AI provides first-class support for integrating MCP (Model Context Protocol) servers into your agents. MCP toolsets allow agents to consume tools from external MCP servers through generated wrappers and callers.


Overview
========

MCP integration follows this workflow:


1. Service design: Declare the MCP server via Goa’s MCP DSL
2. Agent design: Reference that suite with Use(MCPToolset("service", "suite"))
3. Code generation: Produces the MCP JSON-RPC server (when Goa-backed) plus runtime registration helpers and toolset-owned specs/codecs for the suite
4. Runtime wiring: Instantiate an mcpruntime.Caller transport (HTTP/SSE/stdio). Generated helpers register the toolset and adapt JSON-RPC errors into planner.RetryHint values
5. Planner execution: Planners simply enqueue tool calls with canonical JSON payloads; the runtime forwards them to the MCP caller, persists results via hooks, and surfaces structured telemetry


Declaring MCP Toolsets
======================


In Service Design
-----------------

First, declare the MCP server in your Goa service design:

--- CODE ---
package design

import (
    . "goa.design/goa/v3/dsl"
    . "goa.design/goa-ai/dsl"
)

var _ = Service("assistant", func() {
    Description("MCP server for assistant tools")
    
    MCPServer("assistant", "1.0.0", ProtocolVersion("2025-06-18"))
    
    Method("search", func() {
        Payload(func() {
            Attribute("query", String, "Search query")
            Required("query")
        })
        Result(func() {
            Attribute("results", ArrayOf(String), "Search results")
            Required("results")
        })
        MCPTool("search", "Search documents by query")
    })
})

--- END CODE ---


In Agent Design
---------------

Then reference the MCP suite in your agent:

--- CODE ---
var AssistantSuite = MCPToolset("assistant", "assistant-mcp")

var _ = Service("orchestrator", func() {
    Agent("chat", "Conversational runner", func() {
        Use(AssistantSuite)
        RunPolicy(func() {
            DefaultCaps(MaxToolCalls(8))
            TimeBudget("2m")
        })
    })
})

--- END CODE ---


External MCP Servers with Inline Schemas
---------------------------------------

For external MCP servers (not Goa-backed), declare tools with inline schemas:

--- CODE ---
var RemoteSearch = MCPToolset("remote", "search", func() {
    Tool("web_search", "Search the web", func() {
        Args(func() { Attribute("query", String) })
        Return(func() { Attribute("results", ArrayOf(String)) })
    })
})

Agent("helper", "", func() {
    Use(RemoteSearch)
})

--- END CODE ---


Runtime Wiring
==============

At runtime, instantiate an MCP caller and register the toolset:

--- CODE ---
import (
    mcpruntime "goa.design/goa-ai/runtime/mcp"
    mcpassistant "example.com/assistant/gen/assistant/mcp_assistant"
)

// Create an MCP caller (HTTP, SSE, or stdio)
caller, err := mcpruntime.NewHTTPCaller(ctx, mcpruntime.HTTPOptions{
    Endpoint: "https://assistant.example.com/mcp",
})
if err != nil {
    log.Fatal(err)
}

// Register the MCP toolset
if err := mcpassistant.RegisterAssistantAssistantMcpToolset(ctx, rt, caller); err != nil {
    log.Fatal(err)
}
--- END CODE ---


MCP Caller Types
================

Goa-AI supports multiple MCP transport types through the runtime/mcp package. All callers implement the Caller interface:

--- CODE ---
type Caller interface {
    CallTool(ctx context.Context, req CallRequest) (CallResponse, error)
}
--- END CODE ---


HTTP Caller
-----------

For MCP servers accessible via HTTP JSON-RPC:

--- CODE ---
import mcpruntime "goa.design/goa-ai/runtime/mcp"

// Basic usage with defaults
caller, err := mcpruntime.NewHTTPCaller(ctx, mcpruntime.HTTPOptions{
    Endpoint: "https://assistant.example.com/mcp",
})

// Full configuration
caller, err := mcpruntime.NewHTTPCaller(ctx, mcpruntime.HTTPOptions{
    Endpoint:        "https://assistant.example.com/mcp",
    Client:          customHTTPClient,        // Optional: custom *http.Client
    ProtocolVersion: "2024-11-05",            // Optional: MCP protocol version
    ClientName:      "my-agent",              // Optional: client name for handshake
    ClientVersion:   "1.0.0",                 // Optional: client version
    InitTimeout:     10 * time.Second,        // Optional: initialize handshake timeout
})
--- END CODE ---

The HTTP caller performs the MCP initialize handshake on creation and uses synchronous JSON-RPC over HTTP POST for tool calls.


SSE Caller
----------

For MCP servers using Server-Sent Events streaming:

--- CODE ---
import mcpruntime "goa.design/goa-ai/runtime/mcp"

// Basic usage
caller, err := mcpruntime.NewSSECaller(ctx, mcpruntime.HTTPOptions{
    Endpoint: "https://assistant.example.com/mcp",
})

// Full configuration (same options as HTTP)
caller, err := mcpruntime.NewSSECaller(ctx, mcpruntime.HTTPOptions{
    Endpoint:        "https://assistant.example.com/mcp",
    Client:          customHTTPClient,
    ProtocolVersion: "2024-11-05",
    ClientName:      "my-agent",
    ClientVersion:   "1.0.0",
    InitTimeout:     10 * time.Second,
})
--- END CODE ---


Stdio Caller
------------

For MCP servers running as subprocesses communicating via stdin/stdout:

--- CODE ---
import mcpruntime "goa.design/goa-ai/runtime/mcp"

// Basic usage
caller, err := mcpruntime.NewStdioCaller(ctx, mcpruntime.StdioOptions{
    Command: "mcp-server",
})

// Full configuration
caller, err := mcpruntime.NewStdioCaller(ctx, mcpruntime.StdioOptions{
    Command:         "mcp-server",
    Args:            []string{"--config", "config.json"},
    Env:             []string{"MCP_DEBUG=1"},  // Additional environment variables
    Dir:             "/path/to/workdir",       // Working directory
    ProtocolVersion: "2024-11-05",
    ClientName:      "my-agent",
    ClientVersion:   "1.0.0",
    InitTimeout:     10 * time.Second,
})
defer caller.Close() // Clean up subprocess
--- END CODE ---

The stdio caller launches the command as a subprocess, performs the MCP initialize handshake, and maintains the session across tool invocations. Call Close() to terminate the subprocess when done.


CallerFunc Adapter
------------------

For custom caller implementations or testing:

--- CODE ---
import mcpruntime "goa.design/goa-ai/runtime/mcp"

caller := mcpruntime.CallerFunc(func(ctx context.Context, req mcpruntime.CallRequest) (mcpruntime.CallResponse, error) {
    // Custom implementation
    result, err := myCustomMCPCall(ctx, req.Suite, req.Tool, req.Payload)
    if err != nil {
        return mcpruntime.CallResponse{}, err
    }
    return mcpruntime.CallResponse{Result: result}, nil
})
--- END CODE ---


Goa-Generated JSON-RPC Caller
-----------------------------

For Goa-generated MCP clients that wrap service methods:

--- CODE ---
caller := mcpassistant.NewCaller(client) // Uses Goa-generated client
--- END CODE ---


Tool Execution Flow
===================

1. Planner returns tool calls referencing MCP tools (payload is json.RawMessage)
2. Runtime detects MCP toolset registration
3. Forwards canonical JSON payload to MCP caller
4. Invokes MCP caller with tool name and payload
5. MCP caller handles transport (HTTP/SSE/stdio) and JSON-RPC protocol
6. Decodes result using generated codec
7. Returns ToolResult to planner


Error Handling
==============

Generated helpers adapt JSON-RPC errors into planner.RetryHint values:

- Validation errors → RetryHint with guidance for planners
- Network errors → Retry hints with backoff recommendations
- Server errors → Error details preserved in tool results

This allows planners to recover from MCP errors using the same retry patterns as native toolsets.


Complete Example
================


Design
------

--- CODE ---
package design

import (
    . "goa.design/goa/v3/dsl"
    . "goa.design/goa-ai/dsl"
)

// MCP server service
var _ = Service("assistant", func() {
    Description("MCP server for assistant tools")
    
    MCPServer("assistant", "1.0.0", ProtocolVersion("2025-06-18"))
    
    Method("search", func() {
        Payload(func() {
            Attribute("query", String, "Search query")
            Required("query")
        })
        Result(func() {
            Attribute("results", ArrayOf(String), "Search results")
            Required("results")
        })
        MCPTool("search", "Search documents by query")
    })
})

// Agent that uses MCP tools
var AssistantSuite = MCPToolset("assistant", "assistant-mcp")

var _ = Service("orchestrator", func() {
    Agent("chat", "Conversational runner", func() {
        Use(AssistantSuite)
        RunPolicy(func() {
            DefaultCaps(MaxToolCalls(8))
            TimeBudget("2m")
        })
    })
})
--- END CODE ---


Runtime
-------

--- CODE ---
package main

import (
    "context"
    "log"
    
    mcpruntime "goa.design/goa-ai/runtime/mcp"
    chat "example.com/assistant/gen/orchestrator/agents/chat"
    mcpassistant "example.com/assistant/gen/assistant/mcp_assistant"
    "goa.design/goa-ai/runtime/agent/runtime"
)

func main() {
    rt := runtime.New()
    ctx := context.Background()
    
    // Wire MCP caller
    caller, err := mcpruntime.NewHTTPCaller(ctx, mcpruntime.HTTPOptions{
        Endpoint: "https://assistant.example.com/mcp",
    })
    if err != nil {
        log.Fatal(err)
    }
    if err := mcpassistant.RegisterAssistantAssistantMcpToolset(ctx, rt, caller); err != nil {
        log.Fatal(err)
    }
    
    // Register agent
    if err := chat.RegisterChatAgent(ctx, rt, chat.ChatAgentConfig{
        Planner: &MyPlanner{},
    }); err != nil {
        log.Fatal(err)
    }
    
    // Run agent
    client := chat.NewClient(rt)
    // ... use client ...
}
--- END CODE ---


Planner
-------

Your planner can reference MCP tools just like native toolsets:

--- CODE ---
func (p *MyPlanner) PlanStart(ctx context.Context, in *planner.PlanInput) (*planner.PlanResult, error) {
    return &planner.PlanResult{
        ToolCalls: []planner.ToolRequest{
            {
                Name:    "assistant.assistant-mcp.search",
                Payload: []byte(`{"query": "golang tutorials"}`),
            },
        },
    }, nil
}
--- END CODE ---


Best Practices
==============

- Let codegen manage registration: Use the generated helper to register MCP toolsets; avoid hand-written glue so codecs and retry hints stay consistent
- Use typed callers: Prefer Goa-generated JSON-RPC callers when available for type safety
- Handle errors gracefully: Map MCP errors to RetryHint values to help planners recover
- Monitor telemetry: MCP calls emit structured telemetry events; use them for observability
- Choose the right transport: Use HTTP for simple request/response, SSE for streaming, stdio for subprocess-based servers


Next Steps
==========

- Toolsets - Understand tool execution models
- Memory & Sessions - Manage state with transcripts and memory stores
- Production - Deploy with Temporal and streaming UI
