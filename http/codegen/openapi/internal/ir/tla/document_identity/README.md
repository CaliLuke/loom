# OpenAPI document identities

`DocumentIdentity.lean` models operation IDs, route slots, normalized path
templates, and explicit media component-name claims. It captures three
counterexamples to the old behavior: two operations share an operation ID,
different parameter names produce the same normalized path, and media
components with one explicit name but different values are resolved by
last-writer order. It also keeps the valid control of multiple methods on one
path and allows identical media claims to share a component.

Run the bounded model with the pinned Lean 4.34.1 toolchain from this
directory:

```sh
./check.sh
```

`lean-toolchain` pins the compiler. `required-theorems.txt` lists the claims
that `check.sh` verifies are present and axiom-free; `#print axioms` reports
the transitive axiom set for each claim. The audit currently reports that none
of the required theorems depends on an axiom.

The theorems establish the finite claims in the source, not extraction from
Loom expressions, OpenAPI path parsing, media-type semantic equivalence,
deterministic serialization, or Go refinement. `OperationClaim.template`
abstracts the normalized path identity as static segment identities and
parameter positions; `pathKey` distinguishes separately authored path keys and
is assumed to identify one immutable path template. Operation and component
names use finite numeric identities; `none` represents an omitted name.
The Lean examples include a static suffix. Direct Go tests cover
expression-to-path normalization, escaped literal braces, static suffixes,
exact route slots, custom methods, and file-server conflicts. Media claim
values abstract the complete projected Media Type Object; rendered validation
and reversed claim-order tests cover the production projection and allocation
boundary.

The model also covers one authored route expanded over two service base paths.
Both paths retain the same authored route index, so deriving both default
operation IDs from that index violates document uniqueness. The corrected
projection keeps the first ID stable and uses the emitted-path ordinal for
subsequent paths.

## Correspondence and acceptance

- `document_identity.go` owns duplicate operation/path claims and explicit
  component-name conflict checks.
- `v3/builder.go` supplies all final path operations, including file servers,
  before serialization; `v3/openapi32.go` supplies fully projected 3.2 media
  component claims before replacing inline values with references.
- Tests must prove same-path/different-method is valid, equivalent path
  templates are rejected regardless of method, duplicate operation IDs and
  duplicate exact route slots fail with deterministic diagnostics, identical
  explicit media claims share, and conflicting claims fail independent of map
  iteration order. Expanded service-base paths must receive distinct default
  operation IDs while preserving the first route's existing ID.
- Existing valid generated output should remain byte-identical. The intended
  changes are errors for previously accepted documents with duplicate path
  identities, duplicate route/operation identities, or conflicting explicit
  media component names.

The production checks remain direct shared-IR and rendered OpenAPI tests,
libopenapi parsing, Redocly linting, and process-isolated byte comparisons.
