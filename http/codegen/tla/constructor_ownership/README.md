# Constructor identity and package ownership (#735)

A transport body declaration is not a conversion. A named `Text` body can be
built from different projected argument types or from different fields of the
same argument type. Each call must resolve to its own conversion contract.
Separately, payload constructors share a Go package with handler constructors,
body constructors, types and other generated declarations.

`ConstructorOwnership.tla` explores all admission orders for eight requests:
handler/payload and response/payload name collisions, required/optional projected
sources, an exact repeated constructor, and left/right selectors with the same
signature. A preoccupied suffix checks allocation past an existing declaration.
The two invariants require unambiguous package declarations and every admitted
call resolving to the requested conversion. Body type reuse is independent.

| Mode | Expected result | Recorded states generated/distinct |
| --- | --- | --- |
| current | Unique fails: unreserved handler name | 2/2 |
| identity | Unique fails: semantic identity alone does not reserve package names | 2/2 |
| lexical | Resolved fails: unique constructor suppressed by body-type deduplication | 34/29 |
| signature | Unique fails: same signature, different selected-field behavior | 58/43 |
| owner | Both invariants pass | 4560/1630 |

The owner interns a complete finalized constructor plan (ordered
parameter types/names, return carrier and complete emitted conversion body),
allocates its name after reserving other declarations, and records that name on
every call occurrence. Identical plans may share one declaration. Distinct plans
must not be merged merely because their body or signature agrees. Renderer
collection visits every constructor independently of whether its type declaration
has already been emitted.

At the readiness checkpoint, production was unchanged. The implemented seams are
`ServiceData.finalizeConstructors`, `constructorPlan` and the server/client type
collectors. The isolated client naming path has been removed. Allocation runs
after wire and CLI declaration collection; response encoding plans are rebuilt
from allocated constructors before analysis escapes. JSON-RPC consumes this same
HTTP analysis and type rendering, including import-alias reanalysis.

`TestConstructorOwnershipPlans` checks the allocation contradictions directly.
`ConstructorOwnership` testdata and HTTP/JSON-RPC generated-module tests reproduce
eight compile failures (four cases per transport), with distinct-alias, same-plan
and ordinary controls. The selector specimen compiles but its generated right
encoder writes `"left"`; both transports' runtime regressions require `"right"`.
Existing client body initializer tests cover shared sources, different source
types and selected fields, and flat/nested/WebSocket collections.

The model abstracts Go normalization, concrete suffix syntax, complete registry
inventory and constructor-plan extraction. It does not prove Go compilation,
stream timing, constraints, imported identifier binding or JSON contents. Tests
must establish those concrete seams. This is bounded design evidence, not a
proof of every possible design.

Run from the repository with Corretto Java 25.0.4.1 and TLC 1.7.4 (TLC2 2.19),
JAR SHA256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
for mode in current identity lexical signature owner; do
  java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
    tlc2.TLC -workers 1 -metadir "/tmp/loom-735-model-$mode" \
    -config "http/codegen/tla/constructor_ownership/$mode.cfg" \
    http/codegen/tla/constructor_ownership/ConstructorOwnership.tla
done
```

The four negative modes must exit 12 with the stated invariant; owner exits 0.
`CHECK_DEADLOCK FALSE` permits the completed finite allocation state. Preserve
sources and summarized results; checker state and logs stay outside the tree.
