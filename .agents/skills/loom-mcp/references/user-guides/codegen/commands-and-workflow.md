# Codegen: Commands And Workflow

Use this for `loom gen`, `loom example`, and the normal edit/regenerate loop.

## Commands

```bash
loom gen <design-package-import-path> [-o <output-dir>]
loom example <design-package-import-path> [-o <output-dir>]
goa version
```

All commands expect Go import paths, not filesystem paths.

```bash
loom gen example.com/calc/design
loom gen ./design # wrong
```

## Workflow

1. Edit `design/*.go`
2. Run `loom gen <module>/design`
3. If scaffolding is needed, run `loom example <module>/design`
4. Implement logic outside `gen/`
5. Run `go mod tidy` and tests

## Rules

- `loom gen` rewrites `gen/` from scratch each run.
- `loom example` is usually a one-time scaffold step.
- Commit generated code to version control.
