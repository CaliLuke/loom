# HTTP union declaration identity

This model checks the relationship between allocated Go names and emitted union
declarations. Four distinct type identities share two structural union shapes.
The allocator assigns a stable, unique name to each identity. Allocation and
collection each explore every traversal order.

The `shape` configuration models deduplicating declarations by the inner union
shape while references use the enclosing type identity. TLC reproduces a missing
declaration: all four references are allocated, but only two names are emitted.
The `declaration` configuration deduplicates by allocated declaration name and
checks that all references resolve after collection. It also checks name
uniqueness and termination under weak fairness.

Run from this directory with a TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config shape.cfg UnionDeclarations.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config declaration.cfg UnionDeclarations.tla
```

Expected results: `shape` violates `EveryReferenceDeclared`; `declaration`
passes with 425 distinct states (857 generated states).

The invariant extends beyond the bound: after collecting any prefix, each visited
identity's allocated name is in the emitted set. Initially the prefix is empty.
A new visit either inserts its allocated name or finds the same name already
emitted. Thus induction preserves coverage for every finite traversal. This
argument assumes stable, unique allocation; it does not prove the name allocator,
Go type hashing, recursion termination, branch field compatibility, or generated
program compilation. Direct Go tests and generated-module tests must check those
implementation bindings.

The Go binding tests are `TestHTTPUnionDeclarationsCoverAllocatedNames` (576
allocation/collection order pairs, repeated visits) and
`TestUnionResponseBodyGeneratedHTTP` (shared named unions in success/error bodies,
including required, optional, and nullable fields). Run them with:

```sh
LOOM_DIR="$PWD" go test ./http/codegen -run 'TestHTTPUnionDeclarationsCoverAllocatedNames|TestUnionResponseBodyGeneratedHTTP' -count=1
```

## Package reachability

Model for #784; the implementation correspondence is recorded below.

`PackageReachability.tla` separates a client's full response carrier from a
server's selected-view carrier. Both can also need the union for a request.
`BodyTypes` abstracts the already prepared body graph and its branch dependency.
The current shared registry chooses client-reachable unions for both packages;
its counterexample emits a server union without its branch declaration.
Package-specific collection from the same body graph used for declarations
passes the bounded obligations.

Run from this directory with Java 25 and the existing pinned TLC tool:

```sh
java -XX:+UseParallelGC -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-union-declarations-states -config package-checked.cfg PackageReachability.tla
```

Use distinct metadirs per check. Tool JAR SHA-256:
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Release 1.7.4 reports TLC 2.19.

- `package-legacy.cfg`: expected `DependenciesDeclared` failure; nine generated/distinct
  states. Client selects union, server excludes it, no request union. The server
  emits the union but its branch is absent.
- `package-checked.cfg`: all three invariants pass; twelve generated/distinct states.
  Reachable declarations remain complete, unreachable unions are omitted and
  every emitted union has its branch declaration.

The model has one union and one branch, and treats complete prepared body
reachability as input. It does not prove Go graph traversal, recursion, views,
name allocation, type identity, validators or wire behavior. Actual HTTP and
JSON-RPC module compilation reproduce the missing `KindCircleResponseBody` and
`KindNameResponseBody` declarations. Correspondence seams are
`registerBodyRoot`, `collectBodyUnionTypes`, `projectResponseBodyView`,
`collectServerResponseBodyTypes`, and the client/server type renderers.

Direct tests cover included/excluded views and shared request usage. The owner must retain prepared graph identity; a second
canonical traversal or unused-branch generation would not implement the model.

The final controls include tagged and explicitly untagged unions, nested maps,
and an omitted response union also used by a request. Included array/map and
untagged cases compile and vet; all three excluded parent cases fail with undefined
response branches and now compile after the registry repair. Existing HTTP allocated-name/order, nested-body, error and
recursive-projection controls pass. JSON-RPC independently reproduces the
omitted branch failure while its recursive-projection controls remain valid.

Independent reruns reproduce the legacy nine-state failure and candidate
twelve-state success. The implementation composes both models: allocate
each physical union once using the existing declaration identity, then retain
independent client/server membership and visited sets. Sharing allocation must
not suppress a second package's membership; separating membership must not call
the allocating builder twice. Branch declarations and required validators derive
from the actual prepared roots registered by the body declaration owner.


`registerBodyRoot` captures the actual prepared package graph. It records server
validation requirements before response type construction; ordinary branch
collection and union collection consume the same roots. `orderedBodyRoots`
preserves endpoint/request/stream/response/error order. `collectBodyUnionTypes`
uses one declaration map and fresh per-package visited/member maps. Renderers
consume `ServerUnionTypes` or `ClientUnionTypes`, never the other package's list.
`TestViewedUnionPackageMembership` binds package reachability;
`TestPreparedBodyUnionAllocationAndMembership` binds repeated registration,
shared allocation and independent membership;
`TestViewedUntaggedUnionValidatorDependencies` binds actual untagged validators.
HTTP/JSON-RPC compile tests and selected-view wire tests cover behavior outside
this finite model. Existing allocation/order proofs and tests remain intact.

The initial readiness option labeled untagged did not call `Untagged()`; its
result established only tagged object-branch behavior. That claim is superseded
by corrected, explicitly asserted untagged tests. `required_untagged_probe.go.txt`
retains a separate parent-reproduced service-view validation defect exposed by
the correction; the roadmap records reproduction and scope. The final fixture
uses optional constrained fields to exercise real branch validators independently.

## Validator discovery order

`ValidatorOrdering.tla` covers #1028: a declaration first allocated by an
ordinary response can later become a dependency of an untagged union validator.
The eager policy fixes eligibility at first allocation and loses that later
requirement. The deferred policy collects requirements before deciding which
validators to emit. An unrelated ordinary response must not acquire a validator.

Using the pinned tool above, run:

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/loom-validator-eager -config validator-eager.cfg ValidatorOrdering.tla
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/loom-validator-deferred -config validator-deferred.cfg ValidatorOrdering.tla
```

The eager configuration violates `DependenciesDeclared` (15 generated, 11 distinct
states). Deferred passes both invariants (14 generated, nine distinct states).
This checks all orders of three roots over a fixed dependency graph; it assumes
correct physical type identity and dependency extraction. It does not prove Go
layout, generated syntax, recursion termination or runtime validation.

`serverResponseValidation` retains each allocated declaration's physical context.
`finalizeServerResponseValidation` runs after prepared body registration and
attribute collection, using the complete server validation membership. The
declaration allocation order is unchanged. This replaces the earlier claim that
registration before each individual response's construction was sufficient:
later responses may introduce dependencies on earlier declarations.

`TestUnionResponseReusedValidator` checks emitted helper closure in both method
orders and with collection/root reuse. Its `Generated` companion compiles those
modules and checks valid and invalid nested values. The existing viewed union,
type-identity and ordinary-response controls cover package reachability and
absence of unnecessary response validation.
