# Goa References Map

Use Goa framework source as the authoritative reference for DSL and runtime behavior when the bundled guide fragments are insufficient.

## Preferred Source Locations

- First choice: a vendored or sibling `references/goa` clone in the workspace, if present
- Otherwise: the checked-out `goa.design/goa/v3` module in the local Go module cache
- Otherwise: the Goa repository source available to the agent in the current environment

## What To Inspect

- `README.md`: install + quick-start commands (`goa gen`, `goa example`)
- `dsl/`: DSL surface (`Service`, `Method`, `HTTP`, `GRPC`, `JSONRPC`, security)
- `codegen/`: generator internals and conventions
- `http/`, `grpc/`, `jsonrpc/`: transport packages and patterns
- `middleware/`: reusable middleware components
- `expr/`: design expression model

## Skill-Bundled User Guides

- `SKILL.md`: primary routing index for bundled Goa docs
- `references/user-guides/<topic>/...`: task-sized fragments for quick lookup
- `references/user-guides/*.md`: full transcripts kept as fallbacks when a fragment is insufficient
- Repo-specific `goa-light` contract behavior now lives in the skill itself under `Goa-Light Contract Rules`, not in a separate delta appendix.

## Suggested Lookup Flow

1. If the task needs end-user doc steps, open the matching fragment listed in `SKILL.md`.
2. Check the available Goa source tree for DSL and generation behavior.
3. Confirm transport behavior in `http/`, `grpc/`, or `jsonrpc/`.
4. Apply changes in user code by editing DSL first, then regenerating with `goa gen`.

## Useful Search Commands

```bash
# search DSL and transport implementations
rg -n "Service\(|Method\(|HTTP\(|GRPC\(|JSONRPC\(" <goa-source>/dsl <goa-source>/expr

# search generation/runtime behavior
rg -n "Generate|Mount|Decode|Encode|OpenAPI" <goa-source>/codegen <goa-source>/http <goa-source>/grpc <goa-source>/jsonrpc
```
