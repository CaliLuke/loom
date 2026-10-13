# CLI JSON input presence

`CLIInput.tla` models text decoding, required admission and default materialization.
`cli-encoder` fails DefaultAbsent at42states; `cli-old-zero` fails MissingRequired
at43; `cli-old-zero-supplied` fails PreserveSupplied at46; default seeding fails
MissingRequired at44. `cli-decode` retains member absence through validation and
passes all three invariants at60states. These are finite occurrence-state checks,
not proofs of recursive declaration closure or custom codec correctness. Null is
modeled only for nullable inputs admitted by their decoder.

Correspondence: `service_data_json_input.go` owns a detached JSON decoding plan
for structured `--body` and bare collection `--p` flags. `bodyCodecContext` chooses
its physical role independently of client placement. Existing representation
layout, named validation and union declaration owners drive the plan together;
`client_cli.go` consumes its argument, declarations and conversion. Requiredness
runs before conversion/default materialization. `http/cli.UnmarshalNonNullableJSON`
uses Optional's root null admission while retaining JSONOptions/custom codecs.
Encoding declarations remain untouched. Native source default admission remains
separate #835 work, not a property established for ordinary client encoders here.

Regression: `http/codegen/service_data_json_input_test.go` checks detached graphs,
name collisions and independent bare-collection initialization; the generated
`cli_json_body_test.go`, `cli_json_required_issue_test.go` and `cli_json_union_test.go`
check nested/custom/recursive/nullable/union input and mounted CLI execution.
Malformed JSON and non-nullable null rejection are implementation tests outside
this occurrence-state model's admitted-input abstraction.

Use Corretto25.0.4.1 and official TLA+tools1.7.4 (TLC2.19 revision5a47802).
Set `TLA_JAR` to the installed JAR with SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -config cli-decode.cfg \
  -metadir /tmp/loom-cli-input-model-decode CLIInput.tla
```

Repeat with each config. `cli-decode` exits0; the other configs exit12 with the
named invariant (parse/evaluation errors are not successful counterexamples).
Final checked logs `/tmp/loom-838-model-*.log` reproduce all four expected
counterexamples and the 60-state passing decode model. Model source bytes remain
identical to the approved prerequisite readiness.
