---
name: architecture-review
description: Architecture-focused review workflow for the Auto-K Go backend, prioritizing correctness, security, reliability, and maintainability with evidence-backed findings.
---

# Architecture Review

Run a deep architecture-focused code review of the Auto-K Go backend.

## Context

- Project: Auto-K Server
- Stack: Go, Goa, native `net/http`, mcp-go, pgx, TypeDB (via go-typeql), slog
- Domain: Headless API for governed knowledge graphs and document generation
- Constraints:
  - No raw TypeQL string building; use go-typeql AST builders
  - Security fails closed
  - Structured slog logging is required
  - `schema.tql` is source of truth; schema changes require migrations + regeneration
  - Python is deprecated; do not propose adding new Python

## Review Goals

1. Identify high-impact architecture and runtime risks.
2. Evaluate reliability, security, and maintainability.
3. Produce actionable findings with concrete evidence.
4. Prioritize risk reduction over style preference.

## Method

1. Build a system map: entrypoints, middleware, auth, MCP boundaries, graph/data boundaries.
2. Trace critical flows end-to-end across at least three key journeys.
3. Review risk surfaces:
   - Correctness and invariants
   - Security (authn/authz, validation, fail-closed behavior)
   - Data integrity (consistency, migrations, transactional boundaries)
   - Reliability/operability (timeouts, retries, cancellation, error propagation)
   - Observability (logs, correlation IDs, diagnosability)
   - Performance/scalability hotspots
   - Test coverage relative to risk
4. Validate findings against code and tests.
5. Separate confirmed issues from assumptions/questions.

## Severity

- S0: exploitable security flaw, data corruption/loss, systemic outage risk
- S1: major correctness/reliability risk likely in production
- S2: meaningful maintainability/performance/operability risk
- S3: minor issue, local improvement

## Output Format

1. Findings (ordered by severity)
   - ID (`CR-###`)
   - Severity (`S0|S1|S2|S3`)
   - Title
   - Impact
   - Evidence (file paths/functions/behavior)
   - Recommendation (specific fix)
   - Confidence (`High|Medium|Low`)
2. Architecture Assessment Summary
3. Test/Verification Gaps
4. Prioritized Remediation Plan
5. Open Questions / Assumptions

## Principles

- Prefer correctness and risk mitigation over stylistic opinions.
- Avoid broad rewrites unless justified by clear risk.
- Tie major recommendations to measurable operational benefit.
- If no major issues are found in an area, state that explicitly.
