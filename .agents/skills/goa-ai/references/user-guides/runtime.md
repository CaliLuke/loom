# Runtime

Understand how the Goa-AI runtime orchestrates agents, enforces policies, and manages state.

## Architecture Overview

The Goa-AI runtime orchestrates the plan/execute/resume loop, enforces policies, manages state, and coordinates with engines, planners, tools, memory, hooks, and feature modules.

| Layer | Responsibility |
| --- | --- |
| DSL + Codegen | Produce agent registries, tool specs/codecs, workflows, MCP adapters |
| Runtime Core | Orchestrates plan/start/resume loop, policy enforcement, hooks, memory, streaming |
| Workflow Engine Adapter | Temporal adapter implements engine.Engine; other engines can plug in |
| Feature Modules | Optional integrations (MCP, Pulse, Mongo stores, model providers) |

## High-Level Agentic Architecture

- Agents: Orchestrators identified by `agent.Ident`
- Runs: Identified by RunID and grouped by SessionID/TurnID
- Toolsets & tools: Named tool capabilities
- Planners: implement `PlanStart` / `PlanResume`
- Run tree & agent-as-tool: child runs for nested agents
- Session-owned streams: typed events with explicit end marker

## Quick Start

```go
package main

import (
    "context"

    chat "example.com/assistant/gen/orchestrator/agents/chat"
    "goa.design/goa-ai/runtime/agent/model"
    "goa.design/goa-ai/runtime/agent/runtime"
)

func main() {
    rt := runtime.New()
    ctx := context.Background()
    err := chat.RegisterChatAgent(ctx, rt, chat.ChatAgentConfig{Planner: newChatPlanner()})
    if err != nil { panic(err) }

    if _, err := rt.CreateSession(ctx, "session-1"); err != nil { panic(err) }

    client := chat.NewClient(rt)
    out, err := client.Run(ctx, "session-1", []*model.Message{{
        Role: model.ConversationRoleUser,
        Parts: []model.Part{model.TextPart{Text: "Summarize the latest status."}},
    }})
    if err != nil { panic(err) }
}
```

## Client-Only vs Worker

Client-only: call runs with `runtime.WithEngine(temporalClient)` and a generated client.  
Worker-only: register agents and run engine worker loop.

## Plan → Execute → Resume Loop

1. Start workflow, record context (`RunID`, `SessionID`, `TurnID`, labels, caps)
2. `PlanStart`
3. Tool calls scheduled
4. `PlanResume` with tool results until final response or policy stop
5. Hooks and stream events emitted for each step

## Run Phases

`prompted`, `planning`, `executing_tools`, `synthesizing`, `completed`, `failed`, `canceled`

A typical successful progression:

`prompted -> planning -> executing_tools -> planning -> synthesizing -> completed`

`run.Phase` is the fine-grained loop state. `run.Status` is terminal lifecycle.

## Policies, Caps, and Labels

Design-time policy is configured in `RunPolicy(...)`.

Runtime overrides with `rt.OverridePolicy(agentID, runtime.RunPolicy{...})` (local to process only).

## Tool Execution

- Service-native toolsets: runtime decodes payloads and dispatches to toolset executors.
- Agent-as-tool: executes provider agent as child run.
- MCP toolsets: forwards canonical JSON to generated callers.

## Prompt runtime contracts

Runtime prompt flow:

- `PromptRegistry` stores specs
- `WithPromptStore` adds override store
- planner renders prompt via `RenderPrompt(...)`
- `PromptRefs` attached on model requests

## Memory, Streaming, Telemetry

- Hook bus publishes structured events (run lifecycle, phases, tool events, awaits, thinking)
- `memory.Store`, `runlog.Store`, `stream.Sink`
- Stream profile controls visibility by audience

## Tool call display hints

`CallHintTemplate` / `ResultHintTemplate` become `DisplayHint` and can be overridden in runtime.

## Engine abstraction

In-memory for dev, Temporal for production durability and retries.

## Pause & Resume

`rt.PauseRun(...)` and `rt.ResumeRun(...)`, with `run_paused` / `run_resumed` events.

## Tool Confirmation

`Confirmation(...)` (DSL) and `runtime.WithToolConfirmation(...)` (runtime override) implement await/decision.

Decision payload carries `run_id`, `id`, `approved`, `requested_by`.

## LLM integration

Planners use model clients from context:

- `Complete(ctx, *model.Request)`
- `Stream(ctx, *model.Request)`

Decorated clients auto-emit events for assistant chunks/thinking/usage.

## Next steps

- Learn Toolsets
- Runtime streaming and policies
- Memory, sessions, and persistence
