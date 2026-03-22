# Repository Guidelines

## Common Rules

### Agent Behavior

- **Plan before acting**: For ≤2 files, state a brief plan then implement. For ≥3 files, write a step-by-step plan first.
- **Read before editing**: Always read files before modifying. Search over guessing.
- **Fix root causes**: Do not produce local workarounds—fix the real issue.
- **Be concise**: Give short status updates during multi-step work. Present a short summary when done.
- **Loom naming only**: Do not introduce or keep legacy upstream-named aliases, env vars, scripts, targets, or compatibility shims in Loom-owned workflows. Use `loom` naming exclusively.

### Go Code Style

- **Go 1.26+**. Format with `go fmt ./...`.
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

### Testing

- Write table-driven tests in `*_test.go`.
- Name tests `TestXxx`. Keep fast and deterministic.
- Use `testify/require` for assertions.
- Prefer `t.Errorf` over `t.Fatalf` so tests report multiple failures.
- For framework/codegen bugs, add the failing test first, then change implementation.
- Prefer direct seam tests for generator logic (`expr`, `http/codegen`, `codegen/service`) before leaning on broad goldens alone.
- When output shape matters, pair direct structural assertions with rendered/golden coverage.
- For OpenAPI contract work, validate rendered specs with `libopenapi` and lint the specimen outputs with Redocly.
- Keep the OpenAPI specimen matrix meaningful. Reuse or extend the non-trivial fixtures under `http/codegen/testdata` instead of inventing throwaway one-off examples when a real contract shape is under test.
- For external temp-module or fake-app generation loops, pin `github.com/CaliLuke/loom` to a pushed GitHub commit, not the local working tree, so CI can reproduce the result.
- For this repo, the standard JSON-RPC integration toggle is `make loom-local` for local iteration and `make loom-remote` for pinned-remote parity. `make loom-status` shows the current mode.
- `make loom-local` writes a repo-local source-mode file for the JSON-RPC temp-module generator; `LOOM_DIR=/absolute/path` still overrides that mode for one-off runs.
- While developing an unpushed framework change, use local mode or set `LOOM_DIR=/absolute/path/to/repo` explicitly so verification exercises the code you just changed rather than the last pushed commit.
- Distinguish the two SSE verification paths:
  - the JSON-RPC temp-module generator must honor the local-vs-remote switch (`make loom-local`, `make loom-remote`, or `LOOM_DIR=...`).
  - temp-copy regeneration smoke tests for checked-in fixtures are intentionally local-only and should rewrite the copied fixture `replace github.com/CaliLuke/loom => ...` to the current repo root before running `loom gen`
- Treat the checked-in SSE fixtures as part of the transport regression surface, not as demos:
  - `http/integration_tests/fixtures/ticktock`
  - `jsonrpc/integration_tests/fixtures/ticktock`
- Be explicit about fixture scope. The checked-in JSON-RPC ticktock fixture only proves POST-initiated SSE behavior; it does **not** cover the raw `events/stream` GET listener contract. That branch currently needs separate codegen or dedicated-fixture coverage.
- Happy-path SSE tests are not enough. When changing SSE behavior, add or update adversarial coverage for:
  - pre-stream endpoint failures
  - event-type compatibility for protocol-level errors
  - compile-after-generation of the emitted fixture app
  - any branch-specific connection timing semantics the fixture actually supports
- For new or changed `loom` framework capabilities, use the [`framework-capability` skill](/Users/luca/code/loom/.agents/skills/framework-capability/SKILL.md).

---

## Loom-Specific Rules

### Build & Test

```bash
make lint          # Run linters
make test          # Run tests
cd cmd/loom && go install .  # Install CLI locally
```

### Code Generation Behavior

- After modifying Loom source, `loom gen` and `loom example` automatically compile and use your changes—no manual rebuild needed.
- `loom gen` deletes and recreates the entire `gen/` directory.
- `loom example` only creates new files; it does not overwrite existing `cmd/` files.

### Slices/Maps and Required Fields

Do not rely on nil vs empty to encode presence. Loom uses `omitempty`—both nil and empty serialize as "missing". If empty is valid, do not mark the field as required.
