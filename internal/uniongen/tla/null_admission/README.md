# Tagged-union null admission

The JSON envelope requires a recognized discriminator and a present value.
Explicit null is a value exactly when the selected branch's evaluated contract
allows it. `expr.AllowsNull` owns that fact, including inherited Nullable and Any.
The model compares the current blanket rejection, blanket acceptance, and a
branch-semantic admission decision. It does not infer policy from Go type names.

Run with Corretto Java 25.0.4.1 and TLC 1.7.4, JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
mkdir -p /tmp/loom-union-null-java
java -Djava.io.tmpdir=/tmp/loom-union-null-java -XX:+UseParallelGC \
  -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-union-null-checked -config Checked.cfg NullAdmission.tla
```

Use separate Java temporary directories for simultaneous checker processes;
the checker extracts standard modules there. Run Old and AllowAll configs with
separate metadata paths. Each must fail Contract, not parsing or termination.
Old rejects known nullable null (23 generated/distinct states, exit 12).
AllowAll accepts known non-nullable null (20 states, exit 12).
Checked passes Contract, RequiredValue, KnownBranch and Completes (24 states).
Initial parallel runs shared an extraction directory and were discarded; the
reported results use sequential runs and a dedicated temporary directory.

`internal/uniongen.UnmarshalJSONBody` corresponds to admission. Service,
projected service and HTTP analyses populate its shared Field model. The proposed
Field fact comes from `expr.AllowsNull` at the branch occurrence. Presence remains
mandatory even for nullable/Any branches. Unknown and missing discriminators
retain the existing error path. Untagged matching and form decoding are unchanged.

Direct tests in service and HTTP codegen cover nullable, inherited nullable, Any,
and non-nullable branches. `generator.TestUnionNullableWireAdmission` reuses the
branch-carrier design to compile and decode service, HTTP client and server
values, asserting absent/null/concrete and malformed envelope behavior. It also
round-trips accepted inputs, including whitespace around null. The additional `TestUnionNullAdmissionTransports` covers JSON-RPC, inherited
Nullable and Any, with projected Any and ordinary-string controls. Explicit
Nullable projected declarations currently fail before codec emission in the
separate #829 naming defect; direct projected analysis still checks Nullable
metadata propagation. This model does not establish projected declaration
correctness.
The model abstracts JSON syntax, duplicate keys, codec errors and branch value
validation. A concrete value is assumed syntactically valid here; runtime tests
remain responsible for its actual decoding and semantic validation.
