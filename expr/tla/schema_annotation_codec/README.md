# Required annotations across a codec boundary

This bounded model distinguishes source resolution from target codec ownership.
The old rule rejects a valid portable authored value at an external target;
the blanket fallback accepts a known builtin failure beside an opaque subtree.
The candidate retains valid declared values as **unsupported**, never as emitted
custom JSON, after independently rejecting known builtin failures.

From this directory run (TLC 1.7.4 jar, TLC2 2.19):

```sh
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -config Old.cfg Annotation.tla
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -config Blanket.cfg Annotation.tla
java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -config Candidate.cfg Annotation.tla
```

Old must fail `PreservePortable`; Blanket must fail `RejectKnownFailure`;
Candidate must pass all four invariants. Use an external `-metadir` for checker
output. `known` summarizes only builtin subtrees outside opaque codec boundaries;
it does not claim knowledge of custom output. `retained` summarizes the captured
ownership and declared-value query. Source-invalid values cannot be retained.
The state space includes inconsistent summaries defensively and rejects them.

Implementation correspondence: `expr/value_schema_annotations.go` owns admission;
`value_declared.go` owns retained source eligibility; `value_schema_annotation_known.go`
checks builtin siblings through existing observation/construction/schema owners.
Direct controls in `value_schema_annotation_codec_test.go` exercise root codecs,
nested custom siblings and invalid source. Existing annotation numeric-boundary
controls and `lean/value_projection/ValueContract/NumericExtraction.lean` establish
the selected exact-decimal counterexample, outside this boolean abstraction.
This model does not prove graph extraction, codec execution, or generated code.
