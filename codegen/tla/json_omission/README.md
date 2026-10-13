# JSON field omission

This bounded model explores omission after required admission and before/alongside
semantic JSON projection. It establishes the automatic omission invariant within its stated inputs, not a
proof of the carrier classifier or arbitrary application codecs. Authored field exclusion and externally owned full-tag
behavior are outside the preservation invariant; only framework-owned omission
is modeled. Custom carrier states expose why blindly adopting a Go tag does not
establish this contract. The custom codec boundary retains its existing runtime
policy and is explicitly unsupported for builtin semantic projection.

- `current`: JSON `omitempty` drops supplied empty values. `explicit` represents
  existing Optional/Nullable fields whose IsZero already means absence.
- `all-zero`: `omitzero` repairs ordinary nilable carriers, but native zero/empty
  and a nonnil custom value with an IsZero method are still lost.
- `tags-only`: runtime retains values while semantic projection still omits
  emptiness; this must fail agreement.
- `carrier`: desired omission uses actual physical absence, with runtime and
  projection consuming the same fact. Custom codecs retain current omission and
  opaque projection; full authored tags remain outside this automatic-policy model.

Current implementation correspondence: `internal/encodingmeta.ResolveJSONField`, `expr.ServiceFieldLayout`,
`http/codegen/internal/representation.ResolveField`/`FieldTags`, and
`emittedHTTPFieldPolicy`; semantic projection is
`expr/value_projection_build.go`. Direct serialization evidence is in
`internal/encodingmeta/json_presence_{readiness,candidate}_test.go`. The mounted
client/server witness is `codegen/generator/json_omission_readiness_test.go`.

The model does not cover decoding, defaults (#835), required validation, codec
errors, recursive field layout, authored skip tags, protobuf, or text codecs.
Required missing inputs are rejected before emission, not silently repaired.

Run with TLA+ tools 1.7.4 (JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
export TLA_JAR=/absolute/path/to/installed/tla2tools-1.7.4.jar
cd codegen/tla/json_omission
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config current.cfg JSONOmission
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config all-zero.cfg JSONOmission
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config tags-only.cfg JSONOmission
java -cp "$TLA_JAR" tlc2.TLC -workers 1 -config carrier.cfg JSONOmission
```

Checked with Corretto25.0.4.1 and the pinned JAR: current fails PreserveSupplied
after32 states; all-zero fails PreserveSupplied after30; tags-only fails
ProjectionAgreement after32; carrier exhausts58 states with all invariants.
Fresh logs at parent `0f1fc3dc3` are `/tmp/loom-m7-model-{current,all-zero,tags-only,carrier}.log`.
Each run uses separate temporary Java and checker directories. Direct Go controls additionally
prove the custom IsZero counterexample (the first all-zero trace is native).
Keep transient checker directories outside source and remove after review.


The implementation derives the carrier from actual field declaration
selection, not DSL optionality. Explicit full tags with both omission predicates
combine the two observations; these authored contracts are outside this model. Sixteen direct service/HTTP layout controls in
`http/codegen/json_physical_layout_readiness_test.go` show that materialized defaults
can be native in an encoder and Optional in its decoder. This model takes that
carrier as input; it does not prove classifier correspondence. Requiredness here
is admission-before-serialization and is not a model of incomplete examples.
The separate inherited-tag cases and authored native-omitzero projection witnesses
are direct tests; their public explicit-tag policies are outside this model.
