# Codegen: Commands And Workflow

Use this for `goa gen`, `goa example`, and the normal edit/regenerate loop.

## Commands

```bash
goa gen <design-package-import-path> [-o <output-dir>]
goa example <design-package-import-path> [-o <output-dir>]
goa version
```

All commands expect Go import paths, not filesystem paths.

```bash
goa gen goa.design/examples/calc/design
goa gen ./design # wrong
```

## Workflow

1. Edit `design/*.go`
2. Run `goa gen <module>/design`
3. If scaffolding is needed, run `goa example <module>/design`
4. Implement logic outside `gen/`
5. Run `go mod tidy` and tests

## Rules

- `goa gen` rewrites `gen/` from scratch each run.
- `goa example` is usually a one-time scaffold step.
- Commit generated code to version control.
