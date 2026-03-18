# Temporal Setup

Durable production execution for Goa-AI runs.

## Overview

Temporal backs agent runs as workflows and tool calls as activities:

- workflow history is durable and replayable
- tool calls use per-activity retry policies
- a restarted worker resumes without repeating successful work

## How Durability Works

| Component | Role | Durability |
| --- | --- | --- |
| Workflow | Agent run orchestration | Event-sourced; survives restarts |
| Plan Activity | LLM inference | Retries transient failures |
| Execute Tool Activity | Tool invocation | Per-tool retry policies |
| State | Turn history, tool results | Persisted in workflow history |

Concrete example: if 3 tools are planned and one crashes, only that tool retries instead of replaying the entire run.

## What Survives Failures

| Failure Scenario | Without Temporal | With Temporal |
| --- | --- | --- |
| Worker process crashes | Run lost, restart from zero | Replay from history, continues |
| Tool call timeout | Run fails | Automatic retry with backoff |
| Rate limit 429 | Run fails | Backs off and retries |
| Network partition | Partial progress lost | Resumes after reconnect |
| Deploy during run | In-flight runs fail | Draining workers and resume |

## Installation

### Option 1: Docker (Development)

--- CODE ---
docker run --rm -d --name temporal-dev -p 7233:7233 temporalio/auto-setup:latest
--- END CODE ---

### Option 2: Temporalite (Development)

--- CODE ---
go install go.temporal.io/server/cmd/temporalite@latest
temporalite start
--- END CODE ---

### Option 3: Temporal Cloud

Use Temporal Cloud and configure client credentials.

### Option 4: Self-Hosted

Use Docker Compose or Kubernetes depending on your ops baseline.

## Runtime Configuration

Goa-AI uses the `Engine` interface. Swap engines without changing planner code.

In-memory:

--- CODE ---
// Default: no external dependencies
rt := runtime.New()
--- END CODE ---

Temporal:

--- CODE ---
import (
    runtimeTemporal "goa.design/goa-ai/runtime/agent/engine/temporal"
    "go.temporal.io/sdk/client"

    // Generated specs package from the generated agent
    specs "github.com/example/module/gen/agent/specs"
)

temporalEng, err := runtimeTemporal.New(runtimeTemporal.Options{
    ClientOptions: &client.Options{
        HostPort:  "127.0.0.1:7233",
        Namespace: "default",
        // Required: enforce goa-ai's workflow boundary contract.
        // Tool results, server-data, and UI artifacts cross boundaries as canonical JSON bytes
        // (api.ToolEvent/api.ToolArtifact).
        DataConverter: runtimeTemporal.NewAgentDataConverter(specs.Spec),
    },
    WorkerOptions: runtimeTemporal.WorkerOptions{
        TaskQueue: "orchestrator.chat",
    },
})
if err != nil {
    panic(err)
}
defer temporalEng.Close()

rt := runtime.New(runtime.WithEngine(temporalEng))
--- END CODE ---

## Configuring Activity Retries

Tune reliability per toolset in DSL:

--- CODE ---
Use("external_apis", func() {
    ActivityOptions(engine.ActivityOptions{
        Timeout: 30 * time.Second,
        RetryPolicy: engine.RetryPolicy{
            MaxAttempts:        5,
            InitialInterval:    time.Second,
            BackoffCoefficient: 2.0,
        },
    })

    Tool("fetch_weather", "Get weather data", func() { /* ... */ })
    Tool("query_database", "Query external DB", func() { /* ... */ })
})

Use("local_compute", func() {
    ActivityOptions(engine.ActivityOptions{
        Timeout: 5 * time.Second,
        RetryPolicy: engine.RetryPolicy{
            MaxAttempts: 2,
        },
    })

    Tool("calculate", "Pure computation", func() { /* ... */ })
})
--- END CODE ---

## Worker Setup

Workers poll task queues and execute workflows/activities for registered agents.

## Best Practices

- Use separate environments (`dev`, `staging`, `prod`) for namespace scoping.
- Configure retries based on service reliability.
- Balance activity timeout values for reliability vs. failure detection speed.
- Use Temporal Cloud if you want hosted durability operations.
