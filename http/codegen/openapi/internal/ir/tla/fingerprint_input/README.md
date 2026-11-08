# Checked fingerprint input analysis

`FingerprintInput` separates semantic input capture, representation admission,
policy-origin identity and annotation effects. The implementation belongs in
OpenAPI IR's canonical encoder and `fingerprint_input.go`. The encoder retains
one captured input, admits its whole documentation plan, and carries captured
positions through unchanged structural edges. New semantic alias overlays and
synthetic bytes wrappers receive fresh captures/admission. Actual annotations
retain independent plans; this does not change association ambiguity semantics.

The model visits four positions in every order. Position 3 represents a derived
semantic input with distinct meaning and origin. Ordinary positions use the
frozen root input. Capture work counts root/suffix node visits, excluding plan
construction, constraint solving and materialization. Representation admission
is an explicit precondition, not a proof of Go plan validation. No annotation
value effects are modeled; concrete codec-count tests are required.

Using official TLA+ v1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
and Corretto Java 25.0.4.1:

- Owned passes frozen input, semantic meaning, matching policy origins, mandatory
  admission, analysis budget and completion: 57 generated, 33 distinct, depth 5.
- Recapture violates AnalysisBudget: 2 generated/distinct, depth 2.
- StaleDerived violates SemanticInput: 4 generated/distinct, depth 2.
- CopiedPolicy violates PolicyOrigin: 3 generated/distinct, depth 2.
- SkipAdmission violates RepresentationAdmission in the initial state.

From this directory:

```sh
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$TLC_STATE" -config Owned.cfg FingerprintInput.tla
```

Repeat each negative configuration and require its named invariant failure.
Keep generated states, traces and logs outside the repository.

The model abstracts recursive traversal, default/enum constraints, visibility,
wire collisions, raw/checked hash encoding and opaque codecs. It assumes valid
finite capture inputs and correct construction of a derived input. Go tests
cover those correspondence obligations. Nil roots keep the existing hash without
capture; malformed non-finalized shapes retain their existing panic precondition.
An analyzer error prevents successful document publication even though the
fingerprint API still returns a full-length error-path hash for its callers.

The depth allocation regression fails recursive subtree capture: depths 16/64
allocate 3,233/40,318 objects. One input capture/admission allocates 489/1,718
under Go 1.27.3. The threshold detects growth without timing assertions. Actual
annotation shapes still require independent planning, and new semantic overlays
still require their own capture: no universal linear-time compiler claim follows.


## Prepared component defaults

`DefaultAcquisition` models the checked preparation/acquisition boundary.
Three positions consume two authored owners: positions 1 and 2 share owner 1,
while position 3 has owner 2. Requests revisit position 1. Builtin and opaque
modes use the same ownership relation. The two builtin owners model different
wire values; equal emitted wire bytes may share an implementation key. Opaque
owners remain distinct even when their host values happen to be equal.
Acquisition reads retained wire bytes or
an authored opaque owner; it never serializes a host address or calls a codec.
Preparation itself may call a codec, once per captured position. Optional example
preparation reuses the checked annotations.

Run with the pinned toolchain above, replacing the module with
`DefaultAcquisition.tla` and the configuration with `DefaultPrepared.cfg`.
Prepared passes six invariants (52 generated, 32 distinct states). Each negative
configuration was run and returned exit 12:

- `DefaultPosition.cfg` reproduces the rejected target/position identity and
  violates `ReusedDefault`: the same authored default receives two keys.
- `DefaultRaw.cfg` violates `CallbackFreeAcquisition`.
- `DefaultReprepare.cfg` violates `SinglePreparation`.
- `DefaultCollapse.cfg` violates `DistinctDefaults`.

`PreparedBeforeAcquire` and `RepeatedIdentity` are consistency checks, not
separately mutation-proven properties. The previous two-position model treated
every position as a distinct owner and could not detect the public component
splitting caused by position-based opaque keys; this model replaces that assumption.

Implementation correspondence: `representation.PrepareTargetValues` caches
checked projections per node; `JSONSchemaValue.DefaultOwner` retains authored
assignment/copy identity across captures; `Analyzer.componentDefaultIdentity`
uses emitted wire bytes or encounter ordinals keyed by that opaque owner.
`analyzeReferenceSchema` prepares a cached annotation-only owner for allOf,
discriminator mapping and bytes-base references before acquisition.
`TestCheckedDefaultOwner`, `TestComponentOpaqueDefaultOwnerAcrossTargets`,
`TestOpaqueDefaultDeclarationReuse`, `TestPlanFreeComponentDefaults` and
`TestReferenceAnnotationPreparationCached` bind the ownership assumptions to
actual code. `TestTemplatePublicationDefaultOwner`, `TestTemplateBodyDefaultOwner`
and `TestComposedTemplateDefaultOwner` cover the publication correspondence:
materializing a default transfers the selected authored owner with its value;
local reassignment has a fresh owner even for an equal value. Shared
`applyEffectiveDefault` owns that transfer for composition and templates.
Existing callback and phase-cache tests remain in force.

The model abstracts actual codecs, plan association, schema validation, graph
traversal, rollback and final rendered schema equivalence. It does not imply
one preparation across different captured targets or suppress legitimate custom
example callbacks. Opaque identities are framework-local comparison tokens,
not public names. Explicit Default reassignment owns a new token even when its
raw value equals the old value; controlled copies preserve the existing token.
