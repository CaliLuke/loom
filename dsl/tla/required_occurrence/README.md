# Required occurrence ownership

`Required` conjoins requirements on the current occurrence with those inherited
from its declaration. It does not write into a referenced declaration or promote
local requirements into copies whose physical fields still have that declaration's
layout. Independent aliases and collection occurrences must not contaminate each
other. Required empty scalar values remain supplied values; absence is distinct.

The model explores both refinement orders for two independent occurrences and
both completion orders for deferred copies. Intrinsic `id` is inherited;
`left` and `right` are local requirements. Its three invariants preserve the
shared declaration, declaration storage and occurrence conjunction.

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| `write-through` | `SharedUnchanged` fails after the first refinement | 2 / 2 |
| `deferred` | `DeclarationStorageUnchanged` fails when local requirements are promoted into a completed copy | 4 / 4 |
| `occurrence` | All invariants pass, depth 5 | 14 / 9 |

The deferred counterexample also describes loss of declaration identity during
mapped projection: treating semantic requirements as declaration requirements
changes the existing physical field. The implementation retains immutable scalar
field facts across that projection instead of re-deriving storage from refinements.

Implementation correspondence:

- `dsl.Required` stores only on its current attribute. `completeKeptReferences`
  completes structural copies without replaying occurrence requirements.
- `materializeMethodType` deliberately creates a separately named method
  declaration after the single method customization callback completes. That
  owner explicitly materializes its authored requirements; views derive from
  this new declaration. This is distinct from an occurrence retaining the
  original declaration and is outside this model's identity bound; the
  [method customization model](../method_customization/README.md) checks it.
- `expr.NewMappedAttributeExpr` retains immutable `ObjectFields` for eliminated
  named edges. `NewObjectFields`, mapped queries and `ServiceFieldLayout` consume
  the retained declaration requiredness. Defaults and physical field attributes
  retain their normal ownership. The snapshot carries no expression pointers.
- `validateNamedOccurrence` resolves local predicates in the named declaration's
  layout while preserving the caller's pointer/value position. Required map-key
  membership belongs to the shared validation snapshot, so local additional-key
  requirements run even when intrinsic validation uses a named helper.
- `representation.ResolveField` emits Optional, pointer and omission decisions
  from `LayoutRequired`, matching shared transformations and validation.

`TestRequiredAliasesPreserveDeclaration` reproduces both alias-order mutations;
`TestRequiredAdditionalAliasPreservesLiteralKeys` covers exact undeclared keys.
Existing later-declared, recursive, collection and structure-copy tests retain
completeness and cycle checks while distinguishing occurrence requirements from
physical declarations. Direct mapped and validation tests check retained field
facts and named context restoration. `RequiredOccurrenceDSL` is a maintained
specimen for generated HTTP client/server compilation and runtime tests: Base,
two aliases, direct/array/map occurrences, recursive deferred occurrences,
zero-valued required fields, missing-member rejection and untouched Base uses.
Existing AdditionalComposite HTTP/JSON-RPC tests cover required additional keys.
The existing gRPC effective-constraints specimen compiles the alias and checks
generated handler rejection of absent fields and acceptance of empty values.
OpenAPI 3.1/3.2 tests check alias isolation and reference-sibling requirements.

The model abstracts concrete Go names, wrappers, codecs, JSON Schema syntax and
recursive graph copying. Tests establish those implementation correspondences;
the bounded model is not a proof over arbitrary designs.

Run explicitly with Java 25 and TLA+ tools 1.7.4 (TLC 2.19):

```sh
for policy in write-through deferred occurrence; do
  java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
    -workers 1 -metadir "/tmp/loom-required-$policy" \
    -config "dsl/tla/required_occurrence/$policy.cfg" \
    dsl/tla/required_occurrence/RequiredOccurrence.tla
done
```

The first two modes exit 12; occurrence exits 0. These are opt-in design checks,
never ordinary build, test, hook or CI requirements. Delete checker directories
after independent review, retaining this source and the summarized evidence.

Regenerate after adopting this correction. An alias's `Required("field")` no
longer makes that field required on its base or sibling aliases. Named references
keep the declaration's optional pointer/Optional storage and validate the local
presence constraint. Move a requirement to the base DSL only when it is intended
to apply to every use of that base; do not add compatibility write-through paths.

## Validation record

With Go 1.27.3, direct DSL/expr/shared-codegen/representation packages passed.
Generated HTTP occurrence and HTTP/JSON-RPC additional-member tests passed;
the gRPC effective-constraints module compiled, vetted and passed its handler
runtime checks. The pre-fix occurrence module compiles but its runtime tests
expose contaminated Base output (`left` and `right` appear as scalar zeros).
The corrected module preserves absence and rejects only the local missing keys.

A bounded comparison of RequiredOccurrenceDSL, AdditionalCompositeDSL and
ParameterStylesDSL contains 122 raw/rendered sections: 82 are byte-identical and
40 change. The parameter-style control is wholly unchanged. Intended differences
are optional declaration storage and matching conversions/guards, occurrence
validation instead of mutation of named intrinsic validators, and corresponding
OpenAPI required sets/reference siblings and generated examples. No files are
added or removed. Five independent-process generations are byte-identical.
The comparison and five-process determinism were refreshed against committed
additional-value projection corrections (`700709243`), including occurrence-local
required predicates and the empty-key preservation fix. Both sides use those
same corrections; the counts and intended differences above remain unchanged.
OpenAPI 3.1/3.2 JSON/YAML specimens parse with libopenapi and pass pinned
Vacuum 0.36.2 contract lint with zero errors. Focused golangci checks pass.
