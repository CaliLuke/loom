# Typed collection default literals (#692)

Status: implementation and bounded acceptance complete; frozen for independent final review.
Parent: `58916bc96d8057874242c6fc330f44bc1c684274`.
Worktree: `/Users/luca/.codex/worktrees/typed-collection-defaults/loom`.

1. Reproduce compatible authored numeric slices rendering with their source Go
   container type; preserve typed map and arbitrary-JSON array controls.
2. Inspect the shared default literal owner and every caller. Require recursive
   literals to use the target attribute representation, including nested arrays,
   maps and aliases, with no transport-specific substitution.
3. Record acceptance for normal, empty, nil (if admitted), named and nested
   collections. Preserve scalar literal meaning and custom representation policy.
   Sized range admission (#685) and presence/default substitution (M7) are separate.
4. Obtain independent readiness assessment before implementation. Reuse existing
   transform/generation fixtures for bounded parent/candidate byte comparison,
   separate-process determinism and generated build/vet across affected consumers.
5. Review the exact final diff independently, commit and push atomically, then
   reconcile only #692 witnesses established by those checks.

Initial owner: `codegen/go_transform_defaults.go`. Maps recursively use target
key/element references, while arrays use that path only for arbitrary JSON
values; other arrays fall through to `%#v` of the authored literal. The existing
array helper already recursively formats items, but constructs type syntax
manually. Keep NameScope authoritative for target references when correcting it.
The existing `typedDefaultLiteral` pass-through and silent fallback should be
assessed within this owner; avoid layering another transform-specific formatter.

## Evidence and candidate

Eight direct failures reproduce wrong target types through both the literal
owner and the real `GoTransform` object-default path (Float32, Int64, nested
UInt32, empty Int32). Six controls pass for already typed arrays, typed maps and
arbitrary-JSON arrays. Both HTTP and JSON-RPC generated modules fail at the
reported incompatible slice assignments after successful DSL evaluation and
generation. Logs: `/tmp/loom-typed-defaults-692-red.log` and
`/tmp/loom-typed-defaults-692-generated-red.log`.

The second independent formatting path is in `codegen/go_transform.go`:
`transformObjectArrayDefaultValueCode` repeats the raw-JSON exception, emits
`%#v` for ordinary arrays, and maintains a separate type-switch alias formatter.
A literal-helper-only fix would leave this main transport path defective.

Candidate: make the shared recursive default literal owner render every admitted
array/map using the target NameScope/context reference and recursively rendered
values; have ordinary object defaults and explicit-presence defaults consume
that same owner. Remove the superseded array/map assignment wrappers and alias
item formatters, and collapse the pass-through `typedDefaultLiteral`. Preserve
admission, scalar values and substitution timing. Investigate nil collection
admission before deciding the literal edge case; do not silently turn nil into
empty. Typed collection construction itself needs no new state model; existing
value-contract/presence models remain the authority for when defaults apply.
Bounded tests must compare those branches rather than claim new M7 semantics.

## Admission and representation controls

`dsl.Default` rejects untyped and typed nil top-level defaults before setting the
attribute (`isNilDSLValue`). Nested nil collections are represented explicitly
by the existing value resolver and must remain typed nil literals. Empty arrays
remain nonnil empty arrays, including arrays of named scalars. Neither case
changes when default substitution occurs. Added direct literal/transform cases
cover alias elements, empty aliases, nested maps and nested nil arrays.

The single owner must retain custom representation handling, raw JSON values,
map-key representation and deterministic map ordering. Go type syntax must come
from the target context/NameScope. Scalar formatting is outside this repair;
use accepted scalar constants inside the correctly typed collection rather than
converting values through float64 or inventing numeric admission. Existing
explicit-presence and bytes-default suites are substitution-timing controls.

Independent assessment by `/root/basic_cli_defaults` accepted the shared owner,
with bounded obligations now adopted: preserve original named outer and located
element identity, both Optional/Nullable callers and native application gates,
nested nil maps, and Any-map key representation. Root authorizes implementation
under those obligations; final exact-diff review remains mandatory.

## Final implementation and acceptance

The shared `defaultValueLiteral` now recursively renders arrays and maps from the
original target attribute and its NameScope reference. Named outer containers,
located element types and Any-map key representation retain their target
identity. Nested nil collections render typed nil; empty arrays render nonnil
empty literals. Ordinary assignments and the existing Optional/Nullable paths
consume this owner. The old array/map assignment helpers, alias item formatter,
raw-JSON discovery walk and pass-through wrapper were removed. Unexpected
noncollection inputs fail with an attributed generator error rather than falling
back to a differently typed authored literal. Scalar formatting and default
application timing remain unchanged.

Independent readiness assessment: `/root/basic_cli_defaults`, no architectural
blocker; all representation/presence obligations were incorporated.

Go 1.27.2 acceptance at parent `58916bc96d8057874242c6fc330f44bc1c684274`:

- Forty direct literal/transform cases cover all numeric kinds, empty and nested
  arrays, named outer arrays/maps, named/located elements, nested nil arrays/maps,
  raw JSON values/keys and typed-map preservation. Thirty-two context cases
  preserve requiredness, pointer, UseDefault and JSON-presence gates.
- `TestTypedCollectionDefaultRuntime` compiles and race-tests native, Optional
  and Nullable conversion code: absent values default, supplied empty/populated
  values remain supplied, and nullable null remains null. Generated module vet
  passes. Existing named-collection, bytes-default and metadata controls pass.
- Complete `go test ./codegen -count=1` passes (17.333s). Existing typed-nil and
  union-default DSL rejection controls are reused, with no admission changes.
- Focused HTTP/JSON-RPC default, raw-JSON, scalar-map, collection-CLI, bytes-form
  and type tests pass (13.858s/8.184s). Final extended numeric/named/located
  generated modules, including CLI, build and vet (1.175s/1.378s).
- A JSON-RPC test-helper attempt omitted located-type import preparation for a
  newly constructed CLI analysis. The actual CLI generation pipeline already
  performs that preparation; a minimal real `loom gen` probe confirmed it.
  The durable regression now initializes its CLI service imports consistently.
  This was a test harness defect, not a new framework ticket.

### Bounded output comparison

The parent is an immutable `git archive` under
`/tmp/loom-typed-defaults-692-parent`; only the new reusable test fixture was
copied in. External modules pin pushed
`v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e` and explicitly replace it with
parent or candidate source. The quality fixture uses the remotely verified
parent version `v1.10.0-alpha.9.0.20261012202605-58916bc96d80` instead.

Real CLI generation (`go run github.com/CaliLuke/loom/cmd/loom gen <module>/design`)
compares the HTTP and JSON-RPC typed fixture plus existing CollectionBodyDefaults
and AnyDefaults designs. Parent typed designs fail build/vet on incompatible
slice assignments; all six candidate designs build/vet. The checked-in quality
fixture is regenerated in temporary copies and passes test/vet at both sources.
The three checked-in generated streaming fixtures contain no affected defaults;
no generated fixture edits are needed.

Together, seven specimens contain 109 generated artifacts. Candidate output is
byte-identical in separate processes. Exactly six files change: client/cli.go,
client/types.go and server/types.go for the typed fixture in each transport.
Every difference changes default collection expressions to their target type
(Float32, Int64, nested UInt32, named Counts, empty Count array and located
Located array). The 60 existing collection/raw-JSON control artifacts and 17
quality artifacts remain parent-identical. There are no golden changes.

Evidence before cleanup: `/tmp/loom-typed-defaults-692-comparison/summary.json`,
`/tmp/loom-typed-defaults-692-preservation-summary.json`,
`/tmp/loom-typed-defaults-692-quality.log`, and focused `*-692-*.log` records.
The six design functions are retained in testdata; the comparison invocation,
pins, source selection and intended differences above are sufficient to repeat
the bounded audit without preserving temporary generated trees.

No claim is made for #685 range admission, #693 object default construction or
M7 presence policy. This commit addresses #692's collection type mismatch through
one owner. Final exact-diff review, atomic push, issue closure and scratch cleanup
remain root-owned.
