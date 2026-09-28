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
