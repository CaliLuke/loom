# Value projection proof foundation

Ticket [#567](https://github.com/CaliLuke/loom/issues/567), milestone 1 of the
[value contract](../../../roadmap/value-contract-design.md). This directory
contains finite model vocabulary, independently specified judgments and proved
legacy counterexamples. **Candidate resolver/projection soundness, progress and
preservation are pending #569. Production correspondence is pending #570.**

## Run the gate

Lean **4.34.1** is pinned in `lean-toolchain`. It includes `lake` and
`leanchecker`. This is a development dependency only; no proof library or
generated-service dependency is added. With elan installed:

```sh
export PATH="$HOME/.elan/bin:$PATH"
elan toolchain install leanprover/lean4:v4.34.1
# From the Loom repository root:
make value-contract-proof
```

The Make target runs `scripts/check_value_contract_proof.sh`, then the gate tests
with `LOOM_VALUE_PROOF_TEST=1` so the real rejection controls cannot be skipped.
The script checks
the actual toolchain version, runs `lake build`, then runs the following from
this directory:

```sh
lake env lean -DwarningAsError=true ValueContract/AxiomAudit.lean
lake env leanchecker --fresh ValueContract.Legacy
```

The gate fails on missing tools, command failure, warnings, missing theorem
reports, an absent/empty audit result, or a forbidden transitive axiom. The
reviewed required-theorem manifest is `required-theorems.txt`. The audit uses
Lean's `collectAxioms` on each theorem, including imported dependencies; it does
not infer trust from source-text searches. Only `propext`, `Classical.choice`
and `Quot.sound` are allowed. Each theorem's actual dependency list is printed.
The fresh checker replays the witness module and all imported declarations in
an empty kernel environment. It supplements, rather than replaces, the axiom
audit: a kernel can accept a declaration that depends on a declared axiom.

The ordinary Go tier tests gate orchestration with controlled tool outputs.
The explicitly enabled external-tool tier compiles temporary copies containing
an admitted intermediate theorem or a custom axiom behind an intermediate
theorem. The same audit must reject the transitive dependency, and the full
gate must fail. Another control requires a nonexistent theorem. These are
intentional negative test inputs, never dependencies
of the accepted witnesses. Enabled mode fails when Lean is unavailable.

The gate is explicit in M1. Wiring proof/conformance into CI and `ci-local`
belongs to #570; no ordinary test silently claims to have run Lean.

## Model and witnesses

- `Model.lean` distinguishes raw and semantic values, roles/source identities,
  presence, nil containers, map entry lists, selected union identities, outcomes,
  recursive declarations and target plans. Observation, structural typing,
  scalar decoding and the initial representability fragment are independent of
  any candidate implementation.
- `Legacy.lean` proves that erasing a chosen branch loses information, that
  choosing the first equal-shaped branch changes identity, that literal byte
  text fails a base64-oriented decoder, and that resynthesis replaces both
  authored value and provenance. These are proofs of concrete abstract
  counterexamples, not a simulation proof of historical Go.
- The 22 required witnesses include both directions of the byte-length schema
  mismatch: valid two-byte `hi` has a four-character JSON encoding, and a
  one-byte value also has a four-character encoding. They also show that the
  pad-bit alias `aGl=` decodes to the allowed bytes `hi` while the canonical
  byte enum contains only `aGk=`. Schema enum rejection therefore cannot prove
  runtime branch disjointness. A canonical-spelling control admits `aGk=` in
  both independent predicates.
- Positive controls preserve a selected HTTP body's bytes while dropping its
  service header, keep that field loss representable, retain visible branches,
  distinguish null/absence, limit empty-value collapse to its field rule, retain
  duplicate map entries, and type a finite value of a recursive declaration.

The wire model has one string constructor. Bytes `hi` and String `aGk=` have
the **same wire string** `aGk=`; the collision witness proves both scalar
interpretations accept it. A private wire tag must never establish untagged
branch uniqueness. `ByteCodec` exposes lexical encode/decode functions, and
`ByteCodec.RoundTrip` names a possible theorem premise. The tiny `specimenCodec`
only models the literal witness strings; it is not a complete base64 codec and
does not satisfy a claimed universal round-trip law.
The separate `boundaryCodec` adds exactly the one-byte `aA==` and alias `aGl=`
cases needed for the new counterexamples; it has the same explicit limitation.
Neither finite specimen proves the full base64 grammar or the proposed length
projection. Actual codec acceptance is a tested boundary.

Judgments with a depth argument quantify existentially over all natural depths
at their public boundary. They do not fix a maximum value or recursion depth.
No universal candidate theorem has been proved from this vocabulary yet.

## Required expansion before #569 can pass

The full domain is not represented by the initial wire judgments. `WireTyped`
and `Decode` cover scalar, nullable, array and selected-body cases only.
Object/map/union wire decoding, unique untagged matching, protobuf mapping,
numeric/enum constraints, required/optional fields and full presence equivalence
must be specified and proved in #569. `HasType` currently checks complete
structural trees; it does not type absent optional object fields.
Complete-first authored matching, zero-complete partial fallback and nested
ambiguity ranking need the independently reviewed #575 clarification before
candidate proofs. Emitted-schema validity and actual decoder branch preservation
are separate obligations: matching a schema alone cannot discharge the latter.
Representation ownership must also enter schema/cache/reference plans; the same
named type cannot share an incompatible JSON and raw/location schema. The #574
length/grammar policy and #570 carriers remain pending production work.

`Observe` already expresses object visibility and selected bodies. Its initial
omission examples are not a complete specification of Go JSON/protobuf omission.
The candidate progress theorem must include legitimate field/visibility loss;
it may not exclude those values to make preservation trivial. Raw cyclic/opaque
inputs and all failure outcomes have constructors, but there is no resolver
claiming their outcomes yet.

The [correspondence ledger](correspondence.md) records the current claims and
their limits. #569 must extend the required-theorem manifest and fresh-check
`ValueContract.Proofs`; #570 must connect the proved functions to the executable
reference and production Go tests. Actual lexical codecs, target-plan derivation,
generated compilation and schema/decoder behavior remain separate obligations.
The known named-Bytes pointer-conversion compile regression #576 is a failure
of that generated-Go boundary, outside this semantic model. Passing this gate
does not establish generated-code correctness or close that regression.

Newly discovered weaknesses require updating the relevant counterexample or
model and this ledger before relying on the affected claim. Re-run the audited
gate and obtain independent re-review of the changed proof scope. A new witness
can establish a defect without establishing its candidate repair.
