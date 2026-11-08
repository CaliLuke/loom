# Published default capture

This model separates local constraint authorship from a finalized occurrence's
published default annotation. It informed the immutable publication descriptor
retained during occurrence capture and passed through the target value plan to
JSON Schema annotation preparation.

## Contract and alternatives

Five positions represent a direct declaration, an inherited published default,
an equal-valued local override with a distinct owner, semantic inheritance with
no published annotation, and a target whose published value is 8 and whose owner differs from its
semantic source (the source-owner control isolates the owner mismatch). Capture retains publication presence, value and owner separately
from local authorship. Preparation can visit positions in any order; the original
live value can change after capture.

The selected design must preserve every captured publication, introduce no
annotation where none was published, preserve equal-valued distinct owners, and
leave local authorship unchanged. These are safety properties, not liveness
claims. `PublishedDefaults` subsumes `NoInventedDefault` and `EqualValueOwners`;
the two named invariants document those consequences rather than adding
independent proof obligations. An inherited annotation must not become a local
constraint override.

`Published.cfg` checks the selected retained-publication design. The other
configurations are deliberate negative controls:

- `LocalOnly.cfg` reproduces loss of the inherited published annotation.
- `SourceOwner.cfg` reproduces assigning the semantic source owner to a different
  target's default.
- `Promoted.cfg` records the decision that publication presence must not be
  copied into local authorship. The model does not represent constraint
  selection or conflicts; the inherited-conflict regression tests that
  consequence.
- `Live.cfg` shows why preparation must not reread mutable source values.

## Run explicitly

Use the maintained TLA+ tools 1.7.4 jar (TLC reports version 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Run from this directory with a Java runtime and `TLA2TOOLS_JAR` pointing to the
installed jar. Keep checker output outside the source tree:

```sh
model_output=$(mktemp -d)
java -Xmx256m -cp "$TLA2TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir "$model_output" -config Published.cfg DefaultPublication.tla
```

Repeat with each negative-control configuration to inspect its counterexample;
those runs deliberately exit with invariant failure. These commands are opt-in
design tools and are not build, test, CI or release requirements.

## Recorded result and implementation correspondence

Checked with Corretto 25.0.5 and the jar above. The selected design completed with
194 generated states, 65 distinct states and depth 8. Local-only, source-owner
and live-read policies violated `PublishedDefaults`; promotion violated
`LocalAuthorship`. No checker output belongs in this directory.

Occurrence capture in `expr/value_occurrence.go` owns the detached publication;
`expr/value_plan_build.go` retains the target binding;
`expr/value_schema_annotations.go` prepares its checked annotation. Local
constraint selection in `expr/effective_default.go` remains separate.
`TestSchemaAnnotationsPublishedDefaultCapture` covers inherited publication,
unpublished inheritance, equal local override and detached capture.
`TestSchemaAnnotationsTargetDefaultOwner` covers distinct target ownership. `TestSchemaAnnotationsPublishedDefaultPreservesInheritedConflict`
checks that publication does not mask a conflicting inherited default.
`TestComposedTemplateDefaultOwner` covers actual composed owners, while
`TestBodyTemplateInheritanceGeneratedHTTP` covers generated transport and schema
preparation for the reported body-template case.

The finite model assumes that captured values are valid for checked projection.
It does not implement Go graph capture, codecs, schema rendering or inheritance
conflict resolution. Those remain regression-test obligations. The
[checked fingerprint input and default acquisition models](../../../http/codegen/openapi/internal/ir/tla/fingerprint_input/README.md)
cover the downstream checked-acquisition boundary; this model does not authorize
renderer fallbacks or unchecked recapture.
