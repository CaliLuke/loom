# Shared interceptor accessor coverage

The model separates generated declaration coverage from invocation applicability.
Three method identities enumerate every nonempty pair of server/client sets,
including single-side, disjoint, overlap and identical sets. Canonical service
order is represented by method numbers. Each method stands for all declared
payload/result/message accessor records; actual type correctness and native
carrier extraction are outside this set model and require generated Go tests.

`current` selects server declarations whenever that side exists, reproducing
#782's missing client coverage. `append` simply concatenates both lists and
violates declaration uniqueness on overlap. `broaden` uses the correct union
for declarations but also widens invocation sets, violating authored side
applicability. `owned` unions/deduplicates declarations in service order and
leaves invocation sets unchanged. The invariants require complete coverage,
unique declarations, stable order and exactly authored side-local calls.

Run each configuration using a separate temporary checker directory:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-782-model-owned \
  -config owned.cfg InterceptorAccessorCoverage.tla
```

The implementation seam is service interceptor analysis, consumed by
`interceptors.go` and the public/private accessor renderers. Side-local
`InterceptorData.Methods`, stream wrapper collection and #867's client endpoint
bindings must retain their existing meaning. `TestInterceptorAccessorCoverage`
checks emitted type membership and unchanged method-side applicability;
`TestGeneratedInterceptorAccessorCoverage` exercises actual read/write access,
client/server stream operations and endpoint carrier extraction.

No claim about incompatible field schemas (#767), JSON-RPC carrier shape
(#785), user callback behavior or native transport cancellation is made. The
model does not prove Go compilation, exact-byte determinism or runtime casts.
The current, append and broaden configurations exit 12 with the named
invariants (73, 72 and 64 distinct states respectively); owned exits 0 with
126 distinct states. `interceptor_declarations.go` implements the shared
coverage owner, initialized in service analysis and rebuilt by file alias
analysis. The renderer consumes its two placement lists; original side-local
invocation records remain unchanged. Alias and nonapplication tests extend the
implementation checks without changing the finite set model.
