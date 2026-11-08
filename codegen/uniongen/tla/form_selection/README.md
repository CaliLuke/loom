# Selected native form branch presence

A discriminator independently selects a tagged branch. Repeated arrays
can encode both nil and nonnil empty values as zero value keys. This codec does
not preserve their distinction: the selected decoder materializes a nonnil empty
array, then ordinary constraints apply. No sentinel or alternate wire grammar is
introduced. Object required fields and other missing value grammars still fail.

`FormSelection.tla` covers absent/present tag, nil/empty/nonempty input and minimum
length zero/one. `keys-only.cfg` reproduces missing-field rejection in 19 states;
`selected.cfg` checks all 24 states and empty normalization. An initial malformed
assignment in the scratch model was corrected before these recorded runs.

`JSONFieldSelection.tla` separately distinguishes absent keys, present JSON null
and JSON values. Reflection-only default decoding rejects admitted nullable JSON
map branches. `json-default.cfg` reproduces the mismatch in 23 states;
`json-captured.cfg` checks 24 states using captured branch nullability.

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config keys-only.cfg -metadir /tmp/loom-form-keys FormSelection.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config selected.cfg -metadir /tmp/loom-form-selected FormSelection.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config json-default.cfg -metadir /tmp/loom-form-json-default JSONFieldSelection.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config json-captured.cfg -metadir /tmp/loom-form-json-captured JSONFieldSelection.tla
```

`expr.HTTPFormTaggedBranchEncoding` owns the captured codec and empty-presence
fact. `uniongen.ConfigureFormField` lowers it into the shared descriptor for
service/view/HTTP builders. `UnmarshalFormBody` selects the existing
the shared `internal/formgen` captured-field lowering for JSON/nullability,
value/item application and native fields. Flattened object members consume the
same captured descriptors; see `internal/formgen/tla/captured_fields/README.md`. Reusable JSON-only declarations do not acquire
form-admission requirements; actual form occurrences validate their capture.
`TestHTTPFormBranchEmptyPresence` checks descriptor inputs and
`TestDiscriminatorHintGeneratedWire` compiles actual client/server code, checks
wire bytes, decoded and returned slice state, null/empty JSON maps, minimum
length and hint neutrality. These bounded models abstract reflection, generated
Go types and error formatting; implementation tests cover those boundaries.
