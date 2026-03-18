---
name: mcp-analytics
description: Add product analytics and error tracking to MCP servers using PostHog. Use when instrumenting MCP tool calls with timing, error capture, and usage dashboards. Covers the withAnalytics wrapper pattern, PostHog provider implementation, feature flags, and dashboard setup.
---

# MCP Analytics with PostHog

Add product analytics and error tracking to any MCP server using a wrapper pattern that tracks every tool execution without touching core business logic.

Reference: [PostHog MCP Analytics Tutorial](https://posthog.com/tutorials/mcp-analytics) | [GitHub Repo](https://github.com/arda-e/mcp-posthog-analytics)

## What It Tracks

- Execution time for every tool call
- Success/failure status
- Errors with context and stack traces
- Feature flag evaluation for conditional tool registration

---

## Architecture Overview

```text
Tool Handler → withAnalytics() wrapper → AnalyticsProvider interface → PostHog SDK
```

MCP tools are async functions passed to `server.tool()`. Wrapping the handler is a clean way to add observability without coupling business logic to analytics.

**Key files:**

| File           | Purpose                                                   |
| -------------- | --------------------------------------------------------- |
| `tools.ts`     | Pure business logic, zero analytics deps                  |
| `analytics.ts` | `AnalyticsProvider` interface + `withAnalytics()` wrapper |
| `posthog.ts`   | `PostHogAnalyticsProvider` implementation                 |
| `server.ts`    | Tool registration with wrappers                           |
| `index.ts`     | Entry point, provider injection                           |

---

## Project Setup

### Dependencies

```json
{
  "dependencies": {
    "@modelcontextprotocol/sdk": "^1.20.0",
    "dotenv": "^17.2.3",
    "posthog-node": "^5.9.5",
    "zod": "^3.25.76"
  },
  "devDependencies": {
    "@types/node": "^24.7.1",
    "typescript": "^5.9.3"
  }
}
```

### tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "Node16",
    "moduleResolution": "node16",
    "esModuleInterop": true,
    "strict": true,
    "skipLibCheck": true,
    "outDir": "build",
    "rootDir": "./src"
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "build"]
}
```

### Environment

```bash
POSTHOG_API_KEY=phc_your_key_here
POSTHOG_HOST=https://us.i.posthog.com  # or eu.i.posthog.com
```

---

## 1. AnalyticsProvider Interface

Define a provider-agnostic interface so you can swap implementations (PostHog, file logger, no-op for tests).

```typescript
// analytics.ts
export interface AnalyticsProvider {
  trackTool(
    toolName: string,
    result: {
      duration_ms: number;
      success: boolean;
      [key: string]: any;
    },
  ): Promise<void>;

  trackError(
    error: Error,
    context: {
      tool_name: string;
      duration_ms: number;
      args?: Record<string, unknown>;
      [key: string]: any;
    },
  ): Promise<void>;

  close(): Promise<void>;

  isFeatureEnabled(feature: string): Promise<boolean>;
}
```

---

## 2. withAnalytics() Wrapper

The core pattern: a higher-order function that wraps any tool handler with automatic timing and error tracking.

```typescript
// analytics.ts
export async function withAnalytics<T>(
  analytics: AnalyticsProvider | undefined,
  toolName: string,
  handler: () => Promise<T>,
): Promise<T> {
  const start = Date.now();

  try {
    const result = await handler();
    const duration_ms = Date.now() - start;

    // Track successful completion
    await analytics?.trackTool(toolName, { duration_ms, success: true });
    return result;
  } catch (error) {
    const duration_ms = Date.now() - start;

    // Track the error with timing
    await analytics?.trackError(error as Error, {
      tool_name: toolName,
      duration_ms,
    });

    // Re-throw so the MCP server handles it normally
    throw error;
  }
}
```

Key design choices:

- **Gracefully handles undefined analytics** — works when disabled via optional chaining
- **Re-throws errors** — doesn't swallow them, MCP server still returns error responses
- **Zero coupling** — tools don't import or know about analytics

---

## 3. PostHog Provider Implementation

```typescript
// posthog.ts
import { PostHog } from "posthog-node";
import { AnalyticsProvider } from "./analytics.js";

export class PostHogAnalyticsProvider implements AnalyticsProvider {
  private client: PostHog | null;
  private mcpInteractionId: string;

  constructor(
    apiKey: string,
    options?: { host?: string; anonymizeData?: boolean },
  ) {
    this.client = new PostHog(apiKey, { host: options?.host });
    this.mcpInteractionId = `mcp_${Date.now()}_${process.pid}`;
    console.error(
      `[Analytics] Initialized with session ID: ${this.mcpInteractionId}`,
    );
  }

  async trackTool(
    toolName: string,
    result: { duration_ms: number; success: boolean; [key: string]: any },
  ): Promise<void> {
    this.client?.capture({
      distinctId: this.mcpInteractionId,
      event: "tool_executed",
      properties: { tool_name: toolName, ...result },
    });
    console.error(
      `[Analytics] ${toolName}: ${result.success ? "✓" : "✗"} (${result.duration_ms}ms)`,
    );
  }

  async trackError(
    error: Error,
    context: {
      tool_name: string;
      duration_ms: number;
      args?: Record<string, unknown>;
      [key: string]: any;
    },
  ): Promise<void> {
    this.client?.captureException(error, this.mcpInteractionId, {
      duration_ms: context.duration_ms,
      tool_name: context.tool_name,
    });
    console.error(
      `[Analytics] ERROR in ${context.tool_name}: ${error.message}`,
    );
  }

  async isFeatureEnabled(flagName: string): Promise<boolean> {
    const enabled = await this.client?.isFeatureEnabled(
      flagName,
      this.mcpInteractionId,
    );
    return enabled ?? false;
  }

  async close(): Promise<void> {
    try {
      await this.client?.shutdown();
      console.error("[Analytics] Closed");
    } catch (error) {
      console.error("[Analytics] Error during close:", error);
    }
  }
}
```

- `distinctId` uses a session-scoped ID (`mcp_<timestamp>_<pid>`) to group events per MCP session
- `captureException` sends errors to PostHog's built-in error tracking
- `shutdown()` flushes pending events before exit (use `flush()` if you want to keep the client alive)

---

## 4. Tool Registration with Wrappers

```typescript
// server.ts
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { AnalyticsProvider, withAnalytics } from "./analytics.js";
import * as tools from "./tools.js";

async function buildStdioServer(
  analytics?: AnalyticsProvider,
): Promise<McpServer> {
  const server = new McpServer({
    name: "mcp-analytics-server",
    version: "1.0.0",
    capabilities: { resources: {}, tools: {} },
  });

  // Each tool handler wrapped with analytics
  server.tool(
    "getInventory",
    {},
    { title: "Get product inventory" },
    async () =>
      withAnalytics(analytics, "getInventory", () => tools.getInventory()),
  );

  server.tool(
    "checkStock",
    { productId: z.string() },
    { title: "Get stock for a specified product" },
    async (args) =>
      withAnalytics(analytics, "checkStock", () =>
        tools.checkStock(args.productId),
      ),
  );

  // Feature-flagged tool registration
  const isFeatureEnabled =
    (await analytics?.isFeatureEnabled("experimental_tools")) || false;
  if (isFeatureEnabled) {
    server.tool(
      "risky_operation",
      {},
      { title: "Operation that sometimes fails" },
      async () =>
        withAnalytics(analytics, "risky_operation", () =>
          tools.riskyOperation(),
        ),
    );
  }

  return server;
}
```

---

## 5. Entry Point with Provider Injection

```typescript
// index.ts
import "dotenv/config";
import { startStdioServer, stopStdioServer } from "./server.js";
import { AnalyticsProvider } from "./analytics.js";
import { PostHogAnalyticsProvider } from "./posthog.js";

const apiKey = process.env.POSTHOG_API_KEY;
const host = process.env.POSTHOG_HOST;

async function main() {
  let analytics: AnalyticsProvider | undefined = undefined;

  if (!apiKey) {
    console.error(
      "[SERVER] POSTHOG_API_KEY not set, continuing without analytics",
    );
  }

  try {
    if (apiKey)
      analytics = new PostHogAnalyticsProvider(apiKey, {
        host,
        anonymizeData: true,
      });
    const handle = await startStdioServer(analytics);

    process.on("SIGINT", async () => await stopStdioServer(handle, analytics));
    process.on("SIGTERM", async () => await stopStdioServer(handle, analytics));

    await new Promise(() => {}); // Keep alive
  } catch (err) {
    console.error("[SERVER] Error during server startup:", err);
    process.exit(1);
  }
}

main();
```

---

## 6. Tool Definitions (Business Logic Only)

Tools contain zero analytics code — the wrapper handles everything.

```typescript
// tools.ts
const products = [
  { id: "1", name: "Laptop", price: 999, stock: 5 },
  { id: "2", name: "Mouse", price: 29, stock: 50 },
  { id: "3", name: "Keyboard", price: 79, stock: 25 },
];

export async function getInventory() {
  return {
    content: [
      { type: "text" as const, text: JSON.stringify(products, null, 2) },
    ],
  };
}

export async function checkStock(productId: string) {
  const product = products.find((p) => p.id === productId);
  if (!product) throw new Error(`Product ${productId} not found`);
  return {
    content: [
      {
        type: "text" as const,
        text: `${product.name}: ${product.stock} units in stock`,
      },
    ],
  };
}
```

---

## 7. Claude Desktop Configuration

```json
{
  "mcpServers": {
    "mcp-analytics-server": {
      "command": "node",
      "args": ["/absolute/path/to/build/index.js"],
      "env": {
        "POSTHOG_API_KEY": "phc_your_key_here",
        "POSTHOG_HOST": "https://us.i.posthog.com"
      }
    }
  }
}
```

---

## 8. PostHog Dashboard Ideas

Once events flow in, create insights for:

| Dashboard       | Metrics                                       |
| --------------- | --------------------------------------------- |
| **Performance** | P95 duration by tool, slow tool trends        |
| **Reliability** | Error rate by tool, error types, stack traces |
| **Usage**       | Tool calls over time, most/least used tools   |

Events captured:

- `tool_executed` — every successful call with `tool_name`, `duration_ms`, `success`
- Exceptions — via `captureException` with tool context

---

## Adapting This Pattern

To use a different analytics provider:

1. Implement the `AnalyticsProvider` interface
2. Swap the import in `index.ts`
3. No changes to tools or wrapper needed

For testing, create a no-op provider:

```typescript
class NoOpAnalytics implements AnalyticsProvider {
  async trackTool() {}
  async trackError() {}
  async close() {}
  async isFeatureEnabled() {
    return false;
  }
}
```

---

## Python SDK Reference

Install: `pip install posthog` (requires Python 3.10+; v7.x dropped 3.9 support).

### Initialization

```python
from posthog import PostHog

posthog = PostHog(
    api_key="phc_your_key",
    host="https://us.i.posthog.com",  # or eu.i.posthog.com
    debug=False,           # verbose logging
    sync_mode=False,       # True for serverless (Lambda/Render)
    historical_migration=False,
    disable_geoip=True,    # default since v3.0; set False to use server IP
)
```

### Capturing Events

```python
# Basic event (use "[object] [verb]" naming: "tool executed", "user signed up")
posthog.capture(distinct_id="user_123", event="tool executed", properties={
    "tool_name": "getInventory",
    "duration_ms": 42,
    "success": True,
})

# Anonymous event (no person profile created)
posthog.capture(event="tool executed", properties={
    "$process_person_profile": False,
    "tool_name": "checkStock",
})

# With person properties
posthog.capture(distinct_id="user_123", event="tool executed", properties={
    "$set": {"role": "admin"},
    "$set_once": {"first_tool_use": "2025-01-01"},
})
```

### Contexts (v6.x+)

Contexts manage shared state (user, session, tags) across events within a scope.

```python
import posthog

# Basic context with user identification
with posthog.new_context() as ctx:
    posthog.identify_context("user_123")
    posthog.capture(event="tool executed")  # auto-associated with user_123

# Nested contexts inherit parent state
with posthog.new_context(tags={"environment": "prod"}) as parent:
    posthog.identify_context("user_123")
    with posthog.new_context(tags={"tool": "inventory"}) as child:
        posthog.capture(event="tool executed")
        # Has both environment=prod and tool=inventory tags

# Fresh context (no inheritance)
with posthog.new_context(fresh=True) as ctx:
    pass  # clean slate

# Session tracking (UUIDv7 required)
with posthog.new_context() as ctx:
    posthog.set_context_session("01912345-6789-7abc-def0-123456789abc")
    posthog.capture(event="tool executed")
    # Events appear on session replay timeline
```

### Exception / Error Tracking

```python
# Auto-capture exceptions in context (default: enabled)
with posthog.new_context() as ctx:
    raise ValueError("something broke")
    # Automatically captured to error tracking dashboard

# Disable auto-capture for a context
with posthog.new_context(capture_exceptions=False) as ctx:
    pass

# Manual exception capture
posthog.capture_exception(error, distinct_id="user_123")

# Global auto-capture on init
posthog = PostHog(api_key="...", enable_exception_autocapture=True)

# Capture local variables at exception time (debugger-like view)
posthog = PostHog(api_key="...", enable_code_variables_capture=True)
```

### Function Decorator

```python
import posthog

@posthog.new_context(tags={"component": "mcp"})
def handle_tool_call(tool_name: str, args: dict):
    posthog.identify_context("user_123")
    posthog.capture(event="tool executed", properties={"tool_name": tool_name})
    # Exceptions auto-captured, context auto-cleaned up
```

### Feature Flags

```python
# Boolean flag
if posthog.feature_enabled("experimental_tools", "user_123"):
    register_risky_tool()

# Multivariate flag
variant = posthog.get_feature_flag("checkout_flow", "user_123")

# Include flag info in events
posthog.capture(
    distinct_id="user_123",
    event="tool executed",
    properties={"$feature/experimental_tools": True},
)

# Or auto-send all flags (requires local evaluation for perf)
posthog.capture(
    distinct_id="user_123",
    event="tool executed",
    send_feature_flags=True,
)

# Evaluate with overridden properties
posthog.feature_enabled(
    "geo_flag", "user_123",
    person_properties={"$geoip_country_code": "US"},
)

# Local evaluation (faster, fewer API calls)
posthog = PostHog(api_key="...", personal_api_key="phx_your_personal_key")
```

### Group Analytics (paid)

```python
posthog.capture(
    distinct_id="user_123",
    event="tool executed",
    groups={"company": "company_id_123"},
)

posthog.group_identify("company", "company_id_123", properties={
    "name": "Acme Corp",
    "plan": "enterprise",
})
```

### Shutdown and Serverless

```python
# Always call before process exit to flush buffered events
posthog.shutdown()

# For serverless (Lambda, Render): use sync_mode or call shutdown per request
posthog = PostHog(api_key="...", sync_mode=True)  # every capture() is synchronous
```

### Python Wrapper Pattern for MCP

Equivalent of the TypeScript `withAnalytics()` for Python MCP servers:

```python
import time
from typing import Any, Callable, Awaitable, TypeVar
from posthog import PostHog

T = TypeVar("T")

async def with_analytics(
    posthog_client: PostHog | None,
    tool_name: str,
    distinct_id: str,
    handler: Callable[[], Awaitable[T]],
) -> T:
    start = time.monotonic()
    try:
        result = await handler()
        duration_ms = (time.monotonic() - start) * 1000
        if posthog_client:
            posthog_client.capture(
                distinct_id=distinct_id,
                event="tool executed",
                properties={
                    "tool_name": tool_name,
                    "duration_ms": round(duration_ms, 1),
                    "success": True,
                },
            )
        return result
    except Exception as e:
        duration_ms = (time.monotonic() - start) * 1000
        if posthog_client:
            posthog_client.capture_exception(e, distinct_id)
            posthog_client.capture(
                distinct_id=distinct_id,
                event="tool executed",
                properties={
                    "tool_name": tool_name,
                    "duration_ms": round(duration_ms, 1),
                    "success": False,
                    "error": str(e),
                },
            )
        raise
```

### Disabling in Tests

```python
posthog = PostHog(api_key="...", disabled=True)  # no events sent
```
