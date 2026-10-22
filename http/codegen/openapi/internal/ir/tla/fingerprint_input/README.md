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
