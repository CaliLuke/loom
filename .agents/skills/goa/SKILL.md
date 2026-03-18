---
name: goa
description: Build and maintain goa.design services in Go. Use this skill when a user mentions Goa, Goa DSL, `goa gen`, generated `gen/` transport code, OpenAPI/proto generation, service implementation after DSL changes, or refactoring a project with a `design` package.
---
# Goa

Use this skill for `goa.design/goa/v3` work only. It does not cover Goa-AI.

## Non-Negotiables

- Treat `design/*.go` as the source of truth.
- Regenerate after every design change with `goa gen <module-import-path>/design`.
- Never hand-edit generated `gen/` files.
- Implement business logic in non-generated files.
- Use Go import paths for Goa commands, not filesystem paths.
- Commit generated code; do not rely on CI to regenerate it.

## Runtime Gotchas

- SSE server streams do not expose a generated `Open()` hook. Goa emits SSE headers on the first `Send`, so long-idle streams that must return `200` before the first domain event need a non-generated transport/runtime flush strategy or an explicit initial event designed into the contract.
- Do not "fix" SSE by hand-editing generated stream files. Keep the fix in `design/*.go` or non-generated transport/runtime code.
- Do not map multi-cookie responses through ad hoc `Header("set_cookies:Set-Cookie")` bags and then patch generated encoders. Prefer idiomatic Goa cookies in the DSL when feasible. If the response shape still depends on raw cookie header values, emit them from non-generated transport code on the live `http.ResponseWriter` instead of editing generated files.

## Default Workflow

1. Detect the Goa surface: `go.mod`, `design/`, DSL imports, or `gen/` folders.
2. Edit the DSL in `design/`.
3. Run `goa gen <module>/design`.
4. Run `goa example <module>/design` only when scaffolding a new service or new starter files are explicitly wanted.
5. Implement logic outside `gen/`.
6. Verify with `go mod tidy` and project tests.

## Command Reminders

```bash
go install goa.design/goa/v3/cmd/goa@latest
goa version
goa gen <module-import-path>/design
goa example <module-import-path>/design
```

- Correct: `goa gen example.com/myapi/design`
- Incorrect: `goa gen ./design`

## References

- Framework/source map: `references/repo-map.md`
- Use only the original full guide pages under `references/user-guides/*.md`.
- For framework/runtime internals, inspect the Goa source tree described in `references/repo-map.md`.

## Original Guide Pages

- `references/user-guides/quickstart.md`
- `references/user-guides/dsl-reference.md`
- `references/user-guides/code-generation.md`
- `references/user-guides/http-guide.md`
- `references/user-guides/grpc-guide.md`
- `references/user-guides/error-handling.md`
- `references/user-guides/interceptors.md`
- `references/user-guides/production.md`

## Selection Rules

- Start with the one full guide page that best matches the immediate task.
- Load additional full guide pages only if the first one is insufficient.
- Prefer `references/repo-map.md` and the Goa source tree for framework internals or runtime behavior.
