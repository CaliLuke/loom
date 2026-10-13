# Metadata absence defaults — #690

This is the metadata slice of the M7 readiness presence model. It models
selection before parsing, not Go type layout or the complete transport.
`Current.cfg` reproduces missing optional defaults. `Presence.cfg` fills absence
only, keeps required omission invalid, preserves supplied zero/empty/ordinary
values and retains malformed-input errors. Each input stands for a metadata
lookup: zero entries means absent; one or more entries means supplied. Empty
arrays remain indistinguishable from absence under the existing repeated text
codec; this model does not claim otherwise. Binary-key framing (#705), JSON
omission, protobuf collection presence and authorization are outside this model.

`service_data_helpers.go:extractMetadata` resolves the effective default and
renders it in the decoder's physical variable type. `codec_metadata.go` makes
one presence decision, then calls the existing scalar, collection or bearer
parser. Request decoding, response headers/trailers and early stream admission
all consume that owner. Generated mounted tests check actual gRPC framing,
error status and service invocation; direct tests check representation and
required/no-default behavior.

Run from this directory with Java 25 and TLA+ tools 1.7.4. `TLA_JAR` points to the
installed JAR, SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -cp "$TLA_JAR" tlc2.TLC -config Current.cfg -metadir /tmp/loom-690-current MetadataDefaults.tla
java -cp "$TLA_JAR" tlc2.TLC -config Presence.cfg -metadir /tmp/loom-690-presence MetadataDefaults.tla
```

The current variant fails DefaultOnAbsence in 22 generated/distinct states; the
presence variant passes all four invariants in 40 generated/distinct states. Model results cover the finite input classification only;
parsing, default literal typing and service conversion are test obligations.
Keep the model/configurations and summarized results, not checker output.
