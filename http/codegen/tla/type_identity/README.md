# Generated type identity in HTTP caches

Request, response, error, and view variants can retain one design identifier
while referring to distinct Go types. The HTTP type layout registry and NameScope
already use generated type hashes for these identities. Validator eligibility,
user-type collection, union-branch membership, union detection, and multipart
cycle detection must use the same identity.

The model has four generated types, two sharing a design identifier. Only one of
that pair requires a server validator. It explores every collection order and
interleaves collection with a two-node walk. The walk is either acyclic or cyclic.

The three `design_*.cfg` configurations separately reproduce skipped types,
validator leakage, and a false cycle. Both `generated_*.cfg` configurations check
complete collection, exact validator selection, correct cycle detection, and
termination. Each repaired configuration passes with 64 distinct states and 178
generated states.

Run from this directory with a TLA+ tools jar:

```sh
java -cp /path/to/tla2tools.jar tlc2.TLC -config design_visits.cfg TypeIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config design_validators.cfg TypeIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config design_cycles.cfg TypeIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config generated_acyclic.cfg TypeIdentity.tla
java -cp /path/to/tla2tools.jar tlc2.TLC -config generated_cyclic.cfg TypeIdentity.tla
```

The repaired invariants extend to finite inputs when each physical type has a
stable, unique cache key. A seen set skips only an already visited physical type.
A validator key selects precisely the type whose eligibility was recorded.
A recursion guard contains exactly the keys on the active path; removing the key
on return permits later visits through sibling paths. Thus key reuse cannot skip
a distinct type, select its validator, or mistake it for an ancestor.

The model assumes that generated type hashing matches NameScope identity. It does
not prove hash implementation, canonical design cloning, layout selection,
validation semantics, or generated code compilation. Go seam tests exercise the
actual hashes, shared identifiers, traversal order, and recursion. Generated
module tests check request validation and success/error response behavior.

## Audit boundary

`collectUserTypes`, union-branch membership, `containsUnionType`, multipart path
checks, and `ServerRequestValidationTypes` operate on generated body variants.
They use generated type hashes. The existing type layout and union declaration
registries already distinguish these physical identities.

`makeHTTPTypeRecursive` retains design-identifier traversal: its caller first uses
`expr.DupAtt`, whose canonical design clone is itself keyed by design identifier.
The transport IR normalizer has the same clone boundary. Changing only their
visited sets would not change that identity contract; this change does not alter
canonical design cloning or OpenAPI schema identity.

Run the implementation checks from the repository root:

```sh
LOOM_DIR="$PWD" go test ./http/codegen -run 'TypeIdentity|BodyVariants|MultipartBodyVariant|GeneratedTypeTraversal' -count=1
```
