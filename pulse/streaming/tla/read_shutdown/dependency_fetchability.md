# Reproducible hook-preserving dependency

Issue: https://github.com/redis/go-redis/issues/4094
PR: https://github.com/redis/go-redis/pull/4095

## Adopted module

Use `github.com/CaliLuke/go-redis/v9 v9.23.1-0.20261014134601-1073297e76ea`
(commit `1073297e76ea3d39d2d012f1ac76e9ff9e549f72`). This exact ordinary Go module pin
works transitively; consumers do not need a `replace` directive. The fork
contains stable v9.23.0 (`6f2b2639e563b11709e06f42bc0addbfb5a1abb6`), the
reviewed hook-copy fix (`b77660a3c76c01f05b19204f9a72f790a1394ddf`), and
only module-declaration/self-import renaming in its final commit. There is no
unrelated upstream-master dependency.

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

Original-path version queries for the reviewed PR commits fail with `unknown
revision`: the commits are not reachable from upstream branches/tags. GitHub PR
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

The fork's exact 238-file module-path diff was independently reviewed; every
changed file is byte-for-byte the reviewed parent with only its root module path
substituted. All root-module packages compiled under `go test -race ./... -run
'^TestWithTimeoutPreservesHooks$' -count=1`; the hook regression passed. Full
proto/util/hashtag/hscan/push race suites also passed. Final integration checks
exercise the published pin rather than a local replacement.
