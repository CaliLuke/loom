# Test-server script identity

The miniredis struct shim changes Lua source and therefore its SHA-1. Current
go-redis rejects a SCRIPT LOAD reply with a different source digest. Restoring
the reply alone leaves EVALSHA and SCRIPT EXISTS looking up the wrong cache key;
the pool fault-injection tests then miss committed cached-script operations.

Keep the original identity visible to the client and its hooks. Publish an
immutable original-to-rewritten digest mapping before sending the source, and
translate digest lookups only in the test server's command hook. Miniredis owns
the actual cache. SCRIPT FLUSH clears that cache without removing translations;
an old translation cannot create a cache hit. The map belongs to one test server,
is shared by its client clones, and dies with the test. `sync.Map` publishes
entries safely; no mutable per-script state is shared across servers.

`ScriptIdentity.tla` models two distinct scripts, publication, successful load
or evaluation, cache flush and lookup in every interleaving. `legacy.cfg` models
reply-only normalization and must fail `LookupAgrees` after publication, commit
and lookup. `checked.cfg` adds server-side lookup translation. Failed operations
leave the cache unchanged. Hash collision freedom and equivalent Lua behavior
are assumptions; the model does not cover Redis transactions, network framing,
or Go hook composition. The implementation is `redistest.go`; the concrete
load/execute/exists/flush regression covers direct, pipeline and transaction
paths, ordinary scripts and cancellation. Pool fault tests retain original
EVALSHA identities and test actual committed-side effects and failures.

TLC 1.7.4 on 2026-11-05: the reply-only control fails after publish, commit and
lookup (21 generated / 7 distinct states). Server-side translation passes both
invariants (58 generated / 9 distinct states). The focused race regression
covering load/execute/flush and the eight affected pool fault scenarios passes.

Run explicitly with TLC 1.7.4 (never as a build or CI gate):

```sh
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/script-identity-checked \
  -config checked.cfg ScriptIdentity.tla
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -metadir /tmp/script-identity-legacy \
  -config legacy.cfg ScriptIdentity.tla
```

Use the installed jar with SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Remove temporary checker output after recording results. The shim remains a
test-only workaround for [miniredis #180](https://github.com/alicebob/miniredis/issues/180).
Remove it when released miniredis supports the required struct operations and
these regressions pass without rewriting.
