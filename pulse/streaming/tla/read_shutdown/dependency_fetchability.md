# Reproducible hook-preserving dependency

Issue: https://github.com/redis/go-redis/issues/4094
PR: https://github.com/redis/go-redis/pull/4095

## Adopted module

Use `github.com/CaliLuke/go-redis/v9 v9.23.1-0.20261105165422-21c4e561f514`.
This exact ordinary Go module pin
works transitively; consumers do not need a `replace` directive. The fork
follows upstream master (`5cba5e182946cf18a757718938bc43910e589466`), which
includes the merged hook-copy fix, with only module-declaration/self-import
renaming and provenance documentation in its final commit. The user-authorized
fork refresh includes current upstream changes; stable v9.23.0 still lacks the
fix. No duplicate patch is applied.

The renamed module has distinct Go type identity. Pulse clients and hook-facing
application code must import the fork consistently. Upstream extra instrumentation
modules remain separate and have not been ported. See `docs/pulse.md` for
application migration. The inspected Loom-MCP consumers use the root Redis client
and custom hooks, with no upstream extra instrumentation imports.

Return to the latest upstream stable release once it contains the correction and
the preserved-hook, finite-operation and synchronous-shutdown regressions pass.
Update the pin, active client imports, consumer guidance and dependent application
imports together. Do not retain compatibility aliases or a second Redis client
identity at Pulse boundaries.

## Why a real module fork

At the original publication, original-path version queries for the reviewed PR commits failed with `unknown
revision`: those commits were not reachable from upstream branches/tags. GitHub PR
refs do not make them valid original-module pseudo-versions. A Loom `replace`
directive would be ignored by downstream consumers, so it cannot deliver this
fix. The user authorized proceeding with delivery; the real fork supersedes the
previous original-module-only waiting condition. No caches, Git remotes or module
proxy responses are rewritten.

The upstream correction copies initialized hooks into `Client.WithTimeout`
facades while keeping their hook slices independent. The initialized-state guard
also preserves minimally constructed clients. Durable original-path reproduction
inputs and the upstream patch in this directory remain historical evidence;
they intentionally retain upstream imports.

## Validation

The original fork's exact 238-file module-path diff was independently reviewed; every
changed file is byte-for-byte the reviewed parent with only its root module path
substituted. All root-module packages compiled under `go test -race ./... -run
'^TestWithTimeoutPreservesHooks$' -count=1`; the hook regression passed. Full
proto/util/hashtag/hscan/push race suites also passed. Final integration checks
exercise the published pin rather than a local replacement.

The refreshed fork's 302 source/module changes are likewise byte-for-byte
upstream with only the root module path substituted, with a separate `FORK.md`
recording provenance. An independent review confirmed this, the merged fix's
ancestry, and unchanged nested modules. All root packages compile under race
instrumentation, the hook-preservation regression passes, and `go vet ./...`
passes on Go 1.27.3.
