---
name: log-check-go
description: High-fidelity logging standards and review checklist for the Auto-K Go server, including slog patterns, operation tracing, and observability requirements.
---

# Log-Check Skill: High-Fidelity Logging for Auto-K (Go Server)

This skill provides a rigorous framework for auditing and implementing logging in the Auto-K Go server. It ensures 100% observability from HTTP handler to TypeDB query, matching the thoroughness of the Python `@log_call` patterns but adapted for Go's `log/slog` and the `StartOp`/`OpResult` idiom.

## Core Mandates

1. **No `fmt.Println` / `log.Printf`**: Always use `slog` via `logging.Component()` or `logging.StartOp()`.
2. **100% Handler Coverage**: Every HTTP handler and MCP tool handler MUST have structured logging for start, success, and error paths.
3. **No Visibility Gaps**: Every `if err != nil` block that returns an error response MUST log before returning. Every recovered panic MUST log at ERROR.
4. **Structured Dot Notation**: All event names follow `component.action.phase` (e.g., `graph.create_nodes.ok`, `transport.graph.list_graph.rejected`).
5. **Context Propagation**: Always pass `ctx` to `slog.*Context()` methods and `StartOp()` — never use bare `slog.Info()` in request-scoped code.
6. **Multi-Sink Awareness**: Logs are automatically sunk to **SQLite OTEL logs** (local persistent) and **Elasticsearch** (remote persistent) if enabled.
7. **Product Analytics**: MCP tool calls and key user actions MUST be captured via **PostHog**.
8. **Error Tracking**: All HTTP-triggered errors and panics are automatically captured by **Sentry**.

---

## 1. Instrumentation Patterns

### A. The `StartOp` / `End` Pattern (Service & Data Layer)

Use for any operation that can fail or takes measurable time. This is the Go equivalent of Python's `@log_call`.

```go
func (t *TypeDB) CreateNodes(ctx context.Context, projectID string,
    req CreateNodesRequest, dbNameOverride string) (result *CreateNodesResult, err error) {

    op := logging.StartOp(ctx, getGraphLog(), "graph.create_nodes",
        slog.String("project", projectID),
        slog.Int("node_count", len(req.Nodes)))
    defer func() { op.End(err) }()  // Named return captures final error

    // ... operation logic ...
    return &CreateNodesResult{Created: created}, nil
}
```

**Rules:**

- Always use **named return** `err error` so `defer func() { op.End(err) }()` captures the final error state.
- Pass initial context attributes (IDs, counts) to `StartOp`.
- Add discovered attributes (result counts, row counts) via `op.With()` before `End`.
- `End(nil)` logs `op.ok` at INFO; `End(err)` logs `op.error` at ERROR — both include `duration_ms`.

### B. HTTP Handler Pattern (Transport Layer)

Handlers that do their own work (not just proxying) need explicit lifecycle logging:

```go
func listGraphHandler(deps *Deps) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log := getGraphRESTLogger()  // logging.Component("transport.graph")
        start := time.Now()

        projectID := r.PathValue("projectID")
        // ... validation ...

        // Log start with all parsed parameters
        log.InfoContext(r.Context(), "list_graph.start",
            "project_id", projectID,
            "types", params.Types)

        result, err := deps.Graph.ListGraph(r.Context(), projectID, params, dbName)
        if err != nil {
            log.ErrorContext(r.Context(), "list_graph.failed",
                "project_id", projectID,
                "error", err,
                "duration_ms", time.Since(start).Milliseconds())
            writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
            return
        }

        log.InfoContext(r.Context(), "list_graph.ok",
            "project_id", projectID,
            "node_count", len(result.Nodes),
            "duration_ms", time.Since(start).Milliseconds())

        writeJSON(w, http.StatusOK, result)
    }
}
```

### C. MCP Tool Handler Pattern (PostHog Integration)

MCP tools are automatically instrumented with **PostHog** product analytics via the `wrapWithAnalytics` decorator in `mcp/server.go`.

**Rules for Manual Capture:**
If you need to capture custom product events (e.g., "prd_generated"), use the `Analytics` dependency:

```go
if deps.Analytics != nil {
    deps.Analytics.Capture(userID, "prd_generated", map[string]any{
        "project_id": projectID,
        "word_count": len(content),
    })
}
```

---

## 2. Sinks & Observability Stack

| Component         | Tech          | Purpose                                | Verification                                      |
| :---------------- | :------------ | :------------------------------------- | :------------------------------------------------ |
| **Console**       | `slog` JSON   | Live tailing, Podman logs              | `podman logs -f auto-k-server-dev`                |
| **Local Sink**    | SQLite        | Local persistence, cross-server tracing| `sqlite3 logs/logs.db`                            |
| **Remote Sink**   | Elasticsearch | Long-term storage, centralized search  | `curl localhost:9200/auto-k-logs/_search`         |
| **Error Tracking**| Sentry        | Exception alerting, request tracing    | https://auto-k.sentry.io                          |
| **Analytics**     | PostHog       | Product usage, feature adoption        | `posthog.init.ok` in logs, PostHog dashboard      |

---

## 3. Auditing Visibility Gaps (The `if err != nil` Check)

A **visibility gap** occurs when an error is handled but not logged, making it invisible.

| Pattern                                    | Audit Action                                             |
| :----------------------------------------- | :------------------------------------------------------- |
| `if err != nil { return err }`             | OK if parent has `StartOp` with `defer op.End(err)`      |
| `if err != nil { writeJSON(w, 500, ...) }` | MUST log at ERROR before `writeJSON`                     |
| `if err != nil { /* fallback */ }`         | MUST log at WARN — this is a best-effort/swallowed error |
| `if err != nil { continue }` in loops      | MUST log at WARN with item identifier                    |
| `defer func() { recover() }()`             | MUST log recovered panic at ERROR with stack             |

---

## 4. Verification

### SQLite Queries (preferred — persistent, queryable)

```bash
# Recent Go+Python logs (last 30 seconds)
sqlite3 logs/logs.db "SELECT timestamp, severity_text, body FROM otel_logs WHERE timestamp > datetime('now', '-30 seconds') ORDER BY timestamp"

# Trace a request by request_id carried in OTEL attributes
sqlite3 logs/logs.db "SELECT timestamp, severity_text, body FROM otel_logs WHERE json_extract(attributes_json, '$.request_id') = 'abc-123' ORDER BY timestamp"
```

### Elasticsearch Queries

```bash
# Verify ES is receiving logs
curl -s "http://localhost:9200/auto-k-logs/_search?size=1&sort=@timestamp:desc" | jq .

# Find errors in ES
curl -s "http://localhost:9200/auto-k-logs/_search?q=level:ERROR" | jq '.hits.hits[]._source'
```

### Sentry Verification

Check for the startup log: `sentry.init.ok environment=production`.
Note: Background goroutines **must** manually capture errors if they want them in Sentry, as the HTTP middleware won't see them.

```go
if err != nil {
    sentry.CaptureException(err)
    log.ErrorContext(ctx, "background_task.failed", "error", err)
}
```

### PostHog Verification

Check for `posthog.init.ok` at startup. Verify tool calls are being captured:
`podman logs auto-k-server-dev 2>&1 | grep "mcp tool called"`

---

## 5. Audit Checklist

| Check                   | Priority | Requirement                                                                      |
| :---------------------- | :------- | :------------------------------------------------------------------------------- |
| **Handler Coverage**    | HIGH     | Every HTTP handler logs `.start` / `.ok` / `.failed` with `duration_ms`          |
| **StartOp Coverage**    | HIGH     | Every graph/TypeDB/storage function uses `StartOp` with `defer op.End(err)`      |
| **Visibility Gaps**     | HIGH     | No `if err != nil` that writes HTTP response without logging first               |
| **Analytics Capture**   | HIGH     | Key product events (beyond tool calls) are captured via `deps.Analytics.Capture` |
| **Sentry for BG Tasks** | MEDIUM   | Critical background tasks manually call `sentry.CaptureException(err)`           |
| **Bulk Ops**            | MEDIUM   | Log counts, not per-item details                                                 |
| **Context Usage**       | HIGH     | All request-scoped code uses `slog.*Context(ctx, ...)` not bare `slog.Info(...)` |

---

## 6. Runtime Debug Runbook (SQLite + SigNoz)

Use this runbook when local and deployed behavior diverge or when debugging auth/session loops.

### Local first-pass queries (SQLite OTEL logs)

```bash
# Last 30 seconds (most useful)
sqlite3 logs/logs.db "SELECT timestamp, severity_text, body FROM otel_logs WHERE timestamp > datetime('now', '-30 seconds') ORDER BY timestamp"

# Errors and warnings, last 5 minutes
sqlite3 logs/logs.db "SELECT timestamp, severity_text, body FROM otel_logs WHERE severity_text IN ('ERROR', 'WARN') AND timestamp > datetime('now', '-5 minutes') ORDER BY timestamp"

# By request ID
sqlite3 logs/logs.db "SELECT timestamp, severity_text, body FROM otel_logs WHERE json_extract(attributes_json, '$.request_id') = 'your-id' ORDER BY timestamp"

# MCP-related
sqlite3 logs/logs.db "SELECT timestamp, severity_text, body FROM otel_logs WHERE body LIKE '%mcp%' ORDER BY timestamp DESC LIMIT 30"
```

Terminology rule:

- `wipe db` / `clear db` means remove rows only; keep `logs/logs.db` and schema.
- `delete db` means remove the database file.
- If wording is ambiguous, ask before destructive actions.

### Staging/Prod workflow (SigNoz)

1. Start with frontend symptom logs (`service.name='auto-k-frontend'`) in the user-reported time window.
2. Find transition events (`consent.*`, `oauth.callback.*`, `auth.signin.*`) and extract `correlationId`/`requestId`.
3. Cross-check backend logs (`service.name='auto-k-server'`) in the same window.
4. Aggregate failure reasons before patching.
5. Classify mismatch vs transport:
   - `token_session_mismatch` / `not_authenticated` implies session-binding/auth state path.
   - network/5xx patterns imply connector/infrastructure path.
6. After deploying a fix, rerun the exact same SigNoz queries and verify reason counts drop to zero.

Auth-loop pattern:

- Frontend may show `oauth.callback.success` while backend still rejects consent with `reason=token_session_mismatch`.
- In that case, debug cookie/session-binding precedence and token rotation first.
