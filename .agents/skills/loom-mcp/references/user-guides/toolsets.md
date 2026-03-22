# TOOLSETS

Learn about toolset types, execution models, validation, retry hints, and tool catalogs in Goa-AI.

Toolsets are collections of tools that agents can use. Goa-AI supports several execution models.

## Toolset Types

### Service-Owned Toolsets (Method-Backed)

Declared via `Toolset("name", func() { ... })`; tools may `BindTo` Goa service methods or be implemented by custom executors.

- Codegen emits per-toolset specs/types/codecs/transforms under `gen/<service>/toolsets/<toolset>/`.
- Agents that `Use` these toolsets get typed builders and executor factories.
- Applications register executors that decode typed args and call service clients or service logic.

### Agent-Implemented Toolsets (Agent-as-Tool)

Defined in an agent `Export` block, consumed by other agents.

- Codegen emits provider-side export packages under `gen/<service>/agents/<agent>/exports/<export>`.
- Execution happens inline from caller perspective while maintaining nested-agent child-run behavior.

### MCP Toolsets

Declared via `MCPToolset(service, suite)` and referenced via `Use(MCPToolset(...))`.

- Generated wrappers handle JSON schemas, transports, and retries.
- Decoding uses MCP executor path with raw JSON input semantics.

## BindTo vs Inline

Use `BindTo` when calling existing Goa methods, using transforms and generated mapping.
Use inline implementations when custom orchestration/computation is required.

## Bounded Tool Results

`BoundedResult()` marks bounded-list views and canonical bounds contract:

```go
type Bounds struct {
    Returned       int
    Total          *int
    Truncated      bool
    RefinementHint string
}
```

### Declaring Bounded Tools

```go
Tool("list_devices", "List devices with pagination", func() {
    Args(func() {
        Attribute("site_id", String, "Site identifier")
        Attribute("status", String, "Filter by status", func() {
            Enum("online", "offline", "unknown")
        })
        Attribute("limit", Int, "Maximum results", func() {
            Default(50)
            Maximum(500)
        })
        Required("site_id")
    })
    Return(func() {
        Attribute("devices", ArrayOf(Device), "Matching devices")
        Attribute("returned", Int, "Returned count")
        Attribute("total", Int, "Total matching devices")
        Attribute("truncated", Boolean, "Whether capped")
        Attribute("refinement_hint", String, "How to narrow results")
        Required("devices", "returned", "truncated")
    })
    BoundedResult()
    BindTo("DeviceService", "ListDevices")
})
```

## Injected Fields

`Inject(fields...)` hides fields from LLM and lets runtime supply values via interceptor.

```go
Tool("get_user_data", "Get user data", func() {
    Args(func() {
        Attribute("session_id", String, "Current session ID")
        Attribute("query", String, "Data query")
        Required("session_id", "query")
    })
    Return(func() {
        Attribute("data", ArrayOf(String), "Query results")
        Required("data")
    })
    BindTo("UserService", "GetData")
    Inject("session_id")
})
```

## Execution Models

### Activity-Based Execution (Default)

Planner returns tool calls, runtime schedules `ExecuteToolActivity`, decodes payload, calls executor, encodes result.

### Inline Execution (Agent-as-Tool)

Toolset marked inline runs provider agent as child run and returns consolidated `planner.ToolResult` with `RunLink`.

### Executor-First Model

Service toolsets expose generic registration:

```go
New<Agent><Toolset>ToolsetRegistration(exec runtime.ToolCallExecutor)
```

## Tool Call Metadata

Executors receive explicit `ToolCallMeta` with:

- `RunID`, `SessionID`, `TurnID`, `ToolCallID`, `ParentToolCallID`.

This avoids hidden context and improves traceability.

## Tool Validation and Retry Hints

Validation failures at decode-time become retry hints and allow planners to recover.

Core fields include:

```go
type RetryHint struct {
    Reason             RetryReason
    Tool               tools.Ident
    RestrictToTool     bool
    MissingFields      []string
    ExampleInput       map[string]any
    PriorInput         map[string]any
    ClarifyingQuestion string
    Message            string
}
```

## Tool Catalogs and Schemas

Generated runtime APIs expose:

- `rt.ListAgents()`
- `rt.ListToolsets()`
- `rt.ToolSpec(toolID)`
- `rt.ToolSchema(toolID)`
- `rt.ToolSpecsForAgent(agentID)`

`tool_schemas.json` contains canonical JSON schemas.

## Server Data and UI Artifacts

`ServerData` lets tools return observer-facing payloads separately from model result.

Model-facing result stays bounded; full artifact can be projected into UI without token overhead.

Artifact type:

```go
type Artifact struct {
    Kind       string
    Data       any
    SourceTool tools.Ident
    RunLink    *run.Handle
}
```

## Best Practices

- Put validations in design, not planners.
- Return `ToolError + RetryHint` from executors.
- Keep hints concise and actionable.
- Avoid re-validating in services; assume boundary validation.
- Use explicit tool IDs (`tools.Ident`) over string literals.

