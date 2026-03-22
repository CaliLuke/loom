# Model Rate Limiting

Every model provider enforces rate limits. Exceed them and requests fail with 429 errors. In replica sets this is often amplified by independent token budgets.

## The Problem

Scenario: you deploy multiple replicas of your agent service. Each replica believes it has a generous local quota, so the aggregate exceeds provider limits.

Without rate limiting:

- Requests fail unpredictably with 429s.
- Retry storms increase load and congestion.
- User experience degrades under burst conditions.

With adaptive rate limiting:

- Replicas share a coordinated budget.
- Requests queue until capacity is available.
- Backoff is propagated across the cluster.
- Traffic degrades gracefully under overload.

## Overview

Use the adaptive limiter in `features/model/middleware`. It estimates token cost, blocks until capacity is available, and adjusts the budget in response to rate-limit feedback.

## AIMD Strategy

The limiter uses an Additive Increase / Multiplicative Decrease strategy.

| Event | Action | Formula |
| --- | --- | --- |
| Success | Probe (additive increase) | `TPM += recoveryRate (5% of initial)` |
| ErrRateLimited | Backoff (multiplicative decrease) | `TPM *= 0.5` |

Effective tokens-per-minute is bounded by:

- Minimum: 10% of initial TPM (floor to avoid starvation)
- Maximum: configured `maxTPM`

## Basic Usage

Create one limiter per process and wrap a model client:

--- CODE ---
import (
    "context"

    "github.com/CaliLuke/loom-mcp/features/model/middleware"
    "github.com/CaliLuke/loom-mcp/features/model/bedrock"
)

func main() {
    ctx := context.Background()

    // Create the adaptive rate limiter
    // Parameters: context, rmap (nil for local), key, initialTPM, maxTPM
    limiter := middleware.NewAdaptiveRateLimiter(
        ctx,
        nil,     // nil = process-local limiter
        "",      // key (unused when rmap is nil)
        60000,   // initial tokens per minute
        120000,  // maximum tokens per minute
    )

    // Create your underlying model client
    bedrockClient, err := bedrock.NewClient(bedrock.Options{
        Region: "us-east-1",
        Model:  "anthropic.claude-sonnet-4-20250514-v1:0",
    })
    if err != nil {
        panic(err)
    }

    // Wrap with rate limiting middleware
    rateLimitedClient := limiter.Middleware()(bedrockClient)

    // Use rateLimitedClient with your runtime or planners
    rt := runtime.New(
        runtime.WithModelClient("claude", rateLimitedClient),
    )
}
--- END CODE ---

## Cluster-Aware Rate Limiting

Coordinate budgets across processes with a Pulse replicated map.

--- CODE ---
import (
    "context"

    "github.com/CaliLuke/loom-mcp/features/model/middleware"
    "goa.design/pulse/rmap"
)

func main() {
    ctx := context.Background()

    // Create a Pulse replicated map backed by Redis
    rm, err := rmap.NewMap(ctx, "rate-limits", rmap.WithRedis(redisClient))
    if err != nil {
        panic(err)
    }

    // Create cluster-aware limiter
    // All processes sharing this map and key coordinate their budgets
    limiter := middleware.NewAdaptiveRateLimiter(
        ctx,
        rm,
        "claude-sonnet",  // shared key for this model
        60000,            // initial TPM
        120000,           // max TPM
    )

    // Wrap your client as before
    rateLimitedClient := limiter.Middleware()(bedrockClient)
}
--- END CODE ---

When cluster-aware:

- backoff propagates globally
- successful requests increase the shared budget
- watchers reconcile limiter state when updates arrive

## Token Estimation

Heuristic estimation:

- count characters in text parts and string tool results
- convert at roughly 3 chars/token
- add 500-token provider/system overhead buffer

## Integration with Runtime

Register rate-limited clients on the runtime:

--- CODE ---
// Create limiters for each model you use
claudeLimiter := middleware.NewAdaptiveRateLimiter(ctx, nil, "", 60000, 120000)
gptLimiter := middleware.NewAdaptiveRateLimiter(ctx, nil, "", 90000, 180000)

// Wrap underlying clients
claudeClient := claudeLimiter.Middleware()(bedrockClient)
gptClient := gptLimiter.Middleware()(openaiClient)

// Configure runtime with rate-limited clients
rt := runtime.New(
    runtime.WithEngine(temporalEng),
    runtime.WithModelClient("claude", claudeClient),
    runtime.WithModelClient("gpt-4", gptClient),
)
--- END CODE ---

## What Happens Under Load

| Traffic Level | Without Limiter | With Limiter |
| --- | --- | --- |
| Below quota | Requests succeed | Requests succeed |
| At quota | Random 429 failures | Requests queue, then succeed |
| Burst above quota | Cascade of failures | Backoff absorbs burst, gradual recovery |
| Sustained overload | All requests fail | Requests queue with bounded latency |

## Tuning

| Parameter | Default | Description |
| --- | --- | --- |
| initialTPM | (required) | Starting tokens/minute |
| maxTPM | (required) | Ceiling for probing |
| Floor | 10% of initial | Minimum budget |
| Recovery rate | 5% of initial | Additive increase per success |
| Backoff factor | 0.5 | Multiplicative decrease on 429 |

Example with `initialTPM=60000`, `maxTPM=120000`:

- Floor: `6000` TPM
- Recovery: `+3000 TPM` per success
- Backoff: halve current TPM on 429

## Monitoring

Track queue time, backoff frequency, and current TPM.

--- CODE ---
// Example: export current capacity to Prometheus
currentTPM := limiter.CurrentTPM()
--- END CODE ---

## Best Practices

- One limiter per model/provider.
- Use realistic initial TPM estimates.
- Enable cluster-aware limiting in production.
- Emit metrics/logs on backoff events.
- Set `maxTPM` above initial for probe headroom.
