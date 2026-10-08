# Repository Guidelines

## Common Rules

### Agent Behavior

- **Plan before acting**: For ≤2 files, state a brief plan then implement. For ≥3 files, write a step-by-step plan first.
- **Read before editing**: Always read files before modifying. Search over guessing.
- **Fix root causes**: Do not produce local workarounds—fix the real issue.
- **Be concise**: Give short status updates during multi-step work. Present a short summary when done.
- **Atomic tickets**: Implement each ticket as exactly one atomic commit. Never split one ticket across multiple commits or combine multiple tickets in one commit.
- **Review before every commit**: Before each commit, have a fresh agent review that commit's exact final diff. The reviewer must not have implemented the change. One reviewer may cover several ready commits in one pass, but it must give a separate verdict for each commit's exact diff against its intended parent; one review of combined changes never authorizes separate commits, and a commit written after a review needs its own review. This applies to every commit, including trivial, single-file, documentation, policy, and release commits.
  - Before review, the implementer states the scope and the acceptance rule, and runs the checks that prove it. For generator changes, compare affected checked-in fixtures and a bounded, representative selection of testdata designs at the base commit and with the change. Cover each distinct changed behavior and relevant normal, edge and rejected cases; compare output byte for byte and compile the generated code. List every intended difference. Reuse existing fixtures for new cases where practical. Follow the validation scope rules below; the full exported-design corpus is not a ticket acceptance requirement.
  - The reviewer assigns each finding a severity. The implementer may lower a severity or reject a finding only with evidence that the re-review accepts.
  - Fix every finding of medium or higher severity, rerun the affected gates, and get a re-review by an agent that did not implement the change; the original reviewer may do it. The re-review confirms each fix and checks the changed lines and their effect on the surrounding code; it does not re-audit the unchanged diff.
  - For a low-severity finding or a nit, make a mechanical fix and have the re-review confirm it, reject it with evidence that the re-review accepts, or file a GitHub issue and leave the code unchanged.
  - A commit is authorized when its latest review, or re-review, leaves no unresolved finding: every finding is fixed and confirmed, rejected with evidence that the re-review accepted, or, for a low finding or nit, filed as a GitHub issue. A rebase or conflict resolution that changes a reviewed diff needs a re-review of the change. Make no changes between that authorizing review and the commit.
- **Formal models and proofs as design tools**: For work with complex interactions or hard-to-predict side effects, such as concurrency, ownership, ordering, retries, or protocol state, use TLA+ or Lean to explore the design before committing to an implementation. Make assumptions and invariants explicit, expose corner cases, and compare candidate designs against the intended contract. When a bug is known, first model the current behavior and confirm that the checker reproduces it. Use counterexamples and proof obligations to revise the design, then implement the supported design and keep the model aligned as implementation reveals new constraints. Keep durable models and their configurations next to the code they describe, as in `pulse/pool/tla/`, and record how to run them and what design decisions they informed in that directory's README. Proofs establish properties within the model's assumptions; tests check the implementation and its correspondence to the model, including behavior outside the model's scope.
- **Loom naming only**: Do not introduce or keep legacy upstream-named aliases, env vars, scripts, targets, or compatibility shims in Loom-owned workflows. Use `loom` naming exclusively.
- **Proof maintenance and cleanup**: Keep proof sources, model configurations, theorem manifests, pinned toolchain declarations, run instructions and concise findings beside the owning code. Link models to implementation seams, regression tests and known limitations; update those records and rerun affected checks when a weakness is discovered. Do not commit compiled proofs, checker state/trace output, raw logs or temporary probe/build directories. As soon as a task's validation and independent review finish, delete its temporary artifacts, including ignored Lean `.lake` output, after confirming no active task or process still needs them. Preserve durable reproduction inputs and summarized evidence first. This is an agent cleanup responsibility; do not add cleanup automation solely for this policy. Keep reusable installed toolchains and shared caches, and use the managed worktree tools for worktree lifecycle operations.

### Architecture Decisions

- **Prioritize architectural fixes for actual bugs.** Choose work by the
  correctness problems it resolves, not by how easy it is to finish. Do not
  substitute cosmetic cleanup or mechanical refactoring for known bugs because
  the bugs are harder. Investigate related failures together and prefer a
  simpler, centralized rewrite of their shared owner when it eliminates a class
  of defects. Delete superseded paths instead of layering fixes over them.
  Establish the common cause with evidence and verify the corrected contract
  across its consumers; do not bundle unrelated changes under architecture.
- **Architectural fixes before workarounds.** Identify the abstraction,
  representation, ownership boundary, or shared contract that causes the
  problem and correct it there. Make dependent layers derive their behavior
  from that owner. Do not compensate for a defective model with scattered
  checks, duplicated policy, per-consumer exceptions, or downstream patches.
  A local fix that passes its regression test is insufficient when the
  architectural cause remains. Validation must establish the corrected
  contract across the affected boundaries.
- **Backward compatibility is not a goal in itself.** Prefer coherent framework
  semantics over preserving defective behavior or a flawed API. Do not add
  compatibility modes, legacy APIs, aliases, or shims merely to avoid a breaking
  correction. Document the resulting behavior and required regeneration or
  migration. Resolve genuinely unsettled public semantics with the user;
  compatibility concerns alone do not justify retaining the defect.
- Derive behavior from the documented DSL and accepted contract semantics. Do not
  privilege accidental runtime behavior, the easiest patch, or a passing test.
- Favor predictable, compositional semantics and explicit author intent over
  clever context-dependent behavior. Do not silently discard authored intent or
  inherited guarantees unless the contract explicitly says to do so.
- Distinguish intended contract behavior from defects, and assess compatibility
  effects before changing either. Inspect actual consumer designs to understand
  adoption impact; never add consumer-specific exceptions.
- Put each decision in the shared layer that owns it. Do not repair the same
  semantic disagreement separately in transports or duplicate policy decisions
  across generators.
- When the accepted contract does not settle a semantic disagreement between
  layers, or a proposal would change that contract, classify it as a policy
  change. Briefly record the alternatives, recommendation, and concrete accepted
  and rejected examples with their effects in the owning design or documentation.
  Ask the user only when public behavior remains genuinely unresolved; make
  routine implementation choices autonomously. A heavyweight RFC or approval is
  not required for every change.
- Use models and proofs during architectural reasoning to expose consequences,
  compare alternatives, and establish invariants. Let that evidence inform the
  design before implementation; it does not decide product intent or authorize
  changes to the accepted public contract. Use tests to check the resulting
  implementation against the design.

### Skill Routing

- Use the [`loom` skill](.agents/skills/loom/SKILL.md) only for consuming Loom:
  authoring `design/`, running `loom gen`, implementing services outside
  `gen/`, and wiring generated transports or runtime packages in an
  application.
- Use the [`loom-framework` skill](.agents/skills/loom-framework/SKILL.md) for
  maintaining this repository: DSL implementation, `expr`, codegen, transport
  internals, OpenAPI generation, framework runtime packages, fixtures, and
  contributor workflows. Its capability section applies when adding or
  materially changing public framework behavior.
- Do not put contributor commands, generator architecture, fixture policy, or
  other maintainer-only directives in the consumer `loom` skill. Update that
  skill only when application developers must use or reason about Loom
  differently.
- For releases, use the [`release` skill](.agents/skills/release/SKILL.md) in
  addition to `loom-framework`.

### Go Code Style

- **Go 1.27+**. Format with `go fmt ./...`.
- **Imports**: Group stdlib separate from external. Let gofmt manage ordering.
- **Files**: Use `lower_snake_case.go`. Keep ≤1000 lines; split proactively.
- **Naming**: Packages are lowercase and short. Exported identifiers need GoDoc. Avoid stutter.
- **Types**: Use `any` over `interface{}`. Prefer concrete types over `interface{}`.
- **Errors**: Wrap with `%w`. Use `errors.Is/As`. **Never ignore errors or use `_ = call()`**.
- **Signatures**: Keep on one line when ≤100 columns. Only wrap genuinely long signatures.
- **Slice/map nil**: Do not check nil before `len`. `len(nil)` returns 0. Use `len(x) == 0` directly.

### Code Blocks and Literals

- Always place a newline after `{` and before `}` for `if`, `for`, `switch`, `func`, `type`.
- No single-line blocks: `if cond { do() }` → use multiple lines.
- Short struct literals are fine inline: `&T{A: 1}`. Break long literals to one field per line with trailing commas.

### File Organization

Order declarations as:

1. Types (public, then private) in a single `type (...)` block when practical
2. Constants (public, then private)
3. Variables (public, then private)
4. Public functions
5. Public methods
6. Private functions
7. Private methods

No commented-out code—delete dead code.

### Loom DSL Rules

- **Never edit `gen/`**: Always regenerate.
- **DSL validation**: Put validations (lengths, enums, formats) in the design. Do not re-validate in code.
- **Avoid `Any`**: Use concrete types to enable gRPC generation.

### Codegen Implementation

- **Use NameScope helpers** for type references: `GoTypeRef`, `GoFullTypeRef`, `GoTypeName`. Never concatenate strings for types.
- Let Loom decide pointer/value semantics. Do not force `pointer=true` except in transport validation.
- **Keep helper visibility minimal**: If logic is shared only inside one codegen area, keep it package-private or move it under an `internal` package. Do not export helpers from a parent package just to share them across sibling generators.
- **Avoid pass-through wrappers**: When two helper functions differ only by forwarding arguments or hard-coding `nil`, collapse them into a single implementation instead of adding an extra layer.

### Documentation

- Every exported type, function, method, and field must have a GoDoc comment explaining its contract—like Go stdlib documentation.
- Public user documentation belongs under `docs/`.
- Active framework plans belong under `roadmap/`; keep only live, framework-owned work there.
- Do not add dated root-level review or improvement notes. Fold durable guidance into `roadmap/` or `docs/`, then remove the temporary note.

### Safety & Forbidden Operations

| Action | Policy |
|--------|--------|
| `git clean/stash/reset/checkout` | **FORBIDDEN** |
| `go clean -cache` | **FORBIDDEN** during normal work |
| Edit `gen/` directly | **FORBIDDEN** |
| Changes ≥3 files | Describe plan first |
| New dependencies | Explain why first |

### Git Remotes

- Treat `origin` as the canonical `CaliLuke/loom` remote unless the user explicitly reconfigures remotes.
- Check `git remote -v` before pushing if there is any ambiguity.

### Releases

- For Loom release work, use the repo-local [`release` skill](.agents/skills/release/SKILL.md).
- Cut releases with `make release VERSION=vX.Y.Z`. Do not rely on implicit or hardcoded version defaults.
- Treat the GitHub Release object as part of the release contract, not an optional follow-up after tag push.
- If a `v*` tag does not result in a matching GitHub Release entry, stop and fix the automation before cutting another release.

### Testing

- **Match validation to the change.** During implementation, run direct regression tests and affected package tests. Before ticket review, add representative generated-output, compilation, determinism and transport checks for the behavior actually changed. State which obligation each selected case covers; do not expand to every fixture merely because it imports a shared package.
- **Keep exhaustive audits separate from ticket validation.** Preserve the full exported-design corpus for occasional, deliberately selected audits after major cross-cutting changes. It is not a default ticket, commit, review or post-fix gate. Record the purpose and expected cost before starting such an audit. Existing CI jobs remain independent of the local ticket acceptance strategy.
- **Reuse valid evidence.** Record the source, inputs and toolchain a result covers. After a fix, rerun the checks it invalidates. Documentation-only edits do not require generator reruns. Independent-process determinism requires repeated generation, not repeated build/vet of identical output under identical source, toolchain and dependency inputs. Do not start a new harness or exhaustive sweep when existing focused checks can establish the acceptance rule.
- Write table-driven tests in `*_test.go`.
- Name tests `TestXxx`. Keep fast and deterministic.
- Use `testify/require` for assertions.
- Prefer `t.Errorf` over `t.Fatalf` so tests report multiple failures.
- For framework/codegen bugs, add the failing test first, then change implementation.
- Prefer direct seam tests for generator logic (`expr`, `http/codegen`, `codegen/service`) before leaning on broad goldens alone.
- When output shape matters, pair direct structural assertions with rendered/golden coverage.
- When changing a serialization library or API, preserve the prior error and
  wire-contract matrix. For JSON request decoding, cover empty, JSON-whitespace,
  malformed, truncated, and valid input at both the runtime seam and a generated
  required and optional body path. Generated transport tests must also assert
  HTTP status, problem code and detail, service invocation, and decoded payload
  state.
- Deterministic generated artifacts require exact-byte comparison across
  process-isolated generations. Semantic decoding and repeated generation in
  one process are insufficient. Custom marshalers that can contain maps must
  preserve deterministic options or sort nested output explicitly. The JSON v2
  lint requires direct `json.Marshal` calls inside `MarshalJSON` methods to pass
  `json.Deterministic(true)`.
- For OpenAPI contract work, validate rendered specs with `libopenapi` and lint the specimen outputs with Redocly.
- Keep the OpenAPI specimen matrix meaningful. Reuse or extend the non-trivial fixtures under `http/codegen/testdata` instead of inventing throwaway one-off examples when a real contract shape is under test.
- For external temp-module or fake-app generation loops, pin `github.com/CaliLuke/loom` to a pushed GitHub commit, not the local working tree, so CI can reproduce the result.
- For this repo, the shared HTTP/JSON-RPC temp-module source toggle is `make loom-local` for local iteration and `make loom-remote` for pinned-remote parity. `make loom-status` shows the current mode.
- Source mode is stored as worktree-local, untracked Git metadata. `LOOM_DIR=/absolute/path` still overrides that persisted mode for one-off runs.
- While developing an unpushed framework change, use local mode or set `LOOM_DIR=/absolute/path/to/repo` explicitly so verification exercises the code you just changed rather than the last pushed commit.
- Distinguish the two SSE verification paths:
  - HTTP and JSON-RPC temp-module generators must honor the shared local-vs-remote switch (`make loom-local`, `make loom-remote`, or `LOOM_DIR=...`).
  - temp-copy regeneration smoke tests for checked-in fixtures are intentionally local-only and should rewrite the copied fixture `replace github.com/CaliLuke/loom => ...` to the current repo root before running `loom gen`
- Treat the checked-in SSE fixtures as part of the transport regression surface, not as demos:
  - `http/integration_tests/fixtures/ticktock`
  - `jsonrpc/integration_tests/fixtures/ticktock`
- Be explicit about fixture scope. The checked-in JSON-RPC ticktock fixture only proves POST-initiated SSE behavior. The raw `events/stream` GET listener contract is covered behaviorally by `jsonrpc/integration_tests/tests/sse_get_listener_test.go`, which regenerates a temp events/stream variant of the fixture; extend that test when changing the GET-listener branch.
- Happy-path SSE tests are not enough. When changing SSE behavior, add or update adversarial coverage for:
  - pre-stream endpoint failures
  - event-type compatibility for protocol-level errors
  - compile-after-generation of the emitted fixture app
  - any branch-specific connection timing semantics the fixture actually supports
- For framework work, follow the skill routing above before applying the
  testing rules in this section.

---

## Loom-Specific Rules

### Build & Test

```bash
make fmt           # Format hand-written Go files and group imports; skip generated files
make lint          # Run linters (filesize, gofmt, namescope, staticcheck, golangci-lint; root module, testdata packages, integration-test modules)
make test          # Run tests
make ci-local      # Run all meaningful direct-main GitHub CI gates locally, except pulse-redis (needs Docker)
./check.sh         # Thin wrapper: make lint + make test
./check.sh --fix   # Auto-fix imports/formatting, then check
./check.sh --full  # Stable wrapper for make ci-local (slow)
make test-pulse-redis  # Pulse suites against the latest stable Redis (pinned in scripts/test_pulse_redis.sh) (Docker, opt-in). FLUSHES DBs 1-3 of LOOM_PULSE_REDIS_ADDR; loopback only unless LOOM_PULSE_REDIS_ALLOW_REMOTE=1
cd cmd/loom && go install .  # Install CLI locally
```

Gate logic lives in `Makefile`, `.golangci.yml`, and `scripts/lint_*.sh`.
Do not duplicate it into `check.sh` — `check.sh` forwards only.

### Code Generation Behavior

- After modifying Loom source, `loom gen` and `loom example` automatically compile and use your changes—no manual rebuild needed.
- `loom gen` deletes and recreates the entire `gen/` directory.
- `loom example` only creates new files; it does not overwrite existing `cmd/` files.

### Slices/Maps and Required Fields

Do not rely on nil vs empty to encode presence. Loom uses `omitempty`—both nil and empty serialize as "missing". If empty is valid, do not mark the field as required.
