# Body template publication

Function-form HTTP bodies select fields from a source occurrence. Their callbacks
run once in the normal DSL phase; a callback may register a `CollectionOf` whose
DSL runs later. The expression composer publishes selected templates only after
those declarations and their inheritance are complete. Explicit local properties
remain separate from the provisional inherited values used by DSL validation.

`BodyTemplateOverlay.tla` models two consumers, selected fields, explicit type
and default choices, a callback-created declaration, and detached mutable data.
The `overlay` configuration checks eight invariants over 78 distinct states.
The bounded negative controls explain the selected design:

| Configuration | Counterexample |
| --- | --- |
| `current`, `eager` | An incomplete inherited field or collection is published. |
| `defer-callback` | The callback registers a declaration after DSL execution ends. |
| `overwrite`, `diff-overlay` | An explicit choice is lost, including a default equal to its provisional value. |
| `whole-source` | Unselected fields or source composition identity leak into the body. |
| `shared` | Mutating a consumer changes the source. |

Run explicitly from this directory with the independently installed TLA+ tools:

```sh
java -Xmx512m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-body-template-states -config overlay.cfg BodyTemplateOverlay.tla
```

The recorded design run used the tla2tools-1.7.4.jar release artifact, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`
(TLC 2.19), and Corretto 25.0.5.0.10. Run the seven negative configurations
separately to reproduce the listed invariant failures. Remove only the owned
checker-state directory after inspecting the result. No proof execution or tool
installation belongs in CI, normal builds, hooks, or release eligibility.

## Implementation correspondence

- `dsl/http_endpoint_body.go` retains original callback timing and evaluator
  context, creates `NewBodyTemplate`, and ends capture before framework writes.
- `expr/attribute_template.go` owns source paths and detached local properties.
  `Attribute`, `Elem`, and `Key` record nested occurrences; scalar semantic setters
  capture validated writes. `AddReference` and `AddBase` retain authored edges.
- `designComposer.publishTemplate` visits source dependencies before selecting
  effective occurrence constraints. The source graph is never flattened into a
  selected body. Explicit local properties replace that template property;
  independent constraints and named-type intrinsic contracts remain present.
- Pending expression copies preserve detached local state. Value snapshots
  discard authoring state. Completed private snapshots do not register new
  generated-result declarations. Metadata deletion history is detached by the
  shared value-copy owner. Publication transfers local deletion history through
  `DeleteMeta`, so a later refined-member composition cannot reintroduce a
  removed key. Re-adding a deleted key retains replacement semantics.

`TestBodyTemplateEffectiveInheritance` covers inline and named composition;
`TestBodyTemplateNestedRefinements` covers repeated object, array and map
callbacks and independent consumers. The property-selection tests cover empty
pattern/title, metadata order/removal, nullable examples and defaults.
`TestBodyTemplateMemberMetadataPublication` checks removal and remove/re-add
against a later local base, preserving ordinary append order and source/copy
isolation. This implementation correspondence is not a separate model policy.
`TestBodyCollectionDeclarationAdmission` and `TestBodyTemplateLateCollectionSource`
cover declarations created during callbacks and incomplete source collections.
`TestTemplatePublicationKeepsAuthorshipAndEndsCapture` checks same-valued explicit
defaults, detached local state and the capture boundary. Existing Extend cycle,
constraint, mapped-requiredness and rejection tests remain separate obligations.

The model proves neither arbitrary recursive Go copying nor setter completeness.
It ends at expression publication: wire admission, default application during
conversion, codecs, OpenAPI serialization and generated program behavior need
independent tests. In particular, preserving a Body-local default does not prove
that a later transport phase applies it. The generated HTTP template specimen
checks that boundary separately; the source-target default phase must not be
silently inferred from this model's green result.
