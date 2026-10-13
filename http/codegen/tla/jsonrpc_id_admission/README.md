# JSON-RPC envelope ID admission

Ordering model for #790. It explores envelope/params source selection, required
presence, optional defaults, validation and delivery ordering. Values abstract
valid String content, invalid content, absence and explicit null. Null admits
only when Nullable, and cannot default. Numeric normalization, actual
codecs, selected body shape and client framing are Go
implementation obligations; this model does not decide their public semantics.

`current` validates params then overwrites the service ID. `late` additionally
validates the envelope in the handler but retains earlier params rejection.
Both must fail. `owned` chooses envelope before defaults/validation and ignores
params as an ID source. It must satisfy exact admission, validation of the
actual delivered value and canonical source preservation. Required presence
precedes defaults, matching existing request validation before conversion.

Correspondence: `expr.resolveJSONRPCID` excludes ID from inferred and streaming
params; mapped indices use `AttributeName`, while the binding retains the actual
service key. `jsonrpcIDBodyConflict` checks selected object members rather than
Reference definitions. `http/codegen.buildJSONRPCID` uses the shared physical
field, validation and conversion owners; its decoder checks required/null
admission before constructor materialization. JSON-RPC handlers consume that
payload without injection. `envelope_id_readiness_test.go`
and its unary/streaming harnesses check admission, actual service values and
ordinary params validation. CLI tests retain nullable null and reject invalid
alias values before field assignment. Outbound body projection and protocol ID echo are separate
bounded acceptance checks.

Use TLA+ tools 1.7.4 (TLC 2.19), same toolchain as sibling response projection:

    java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -metadir /tmp/loom-790-model-owned -config owned.cfg EnvelopeID.tla

Run each configuration with its own temporary metadir. Preserve only summarized
results and source/configuration hashes; delete checker output after review.

Current and late configurations fail ExactAdmission; owned passes 513 distinct
states (577 generated), depth 5. Current native injection skips null, modeled
explicitly; accepted Nullable semantics are a candidate obligation rather than
a claim that every current Nullable carrier compiles.
