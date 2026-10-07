# Protobuf scalar presence

`ScalarPresence.tla` models one scalar with independent supplied, value,
required and defaulted inputs. Values are zero and nonzero; the default is a
third value. The legacy representation retains optional-field presence but
loses required-field presence. Its validator therefore accepts an omitted
required field. Explicit presence lets validation reject absence while
preserving authored zero values. Defaults fill absence only for optional fields.

The accepted contract follows `Required`: a required field with a default
still requires presence. Rejecting zero as a substitute for presence would
reject valid values. Applying defaults to required omissions would weaken the
authored guarantee. An opt-in mode would leave the defective representation as
the default. The implementation instead changes protobuf presence by default;
clients and servers must regenerate. See [the public contract and migration](../../../../docs/grpc-guide.md#scalar-presence-and-defaults).

## Run

Use Java with TLA+ tools 1.7.4. From the repository root:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -config grpc/codegen/tla/scalar_presence/Legacy.cfg -metadir /tmp/scalar-legacy grpc/codegen/tla/scalar_presence/ScalarPresence.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -config grpc/codegen/tla/scalar_presence/Presence.cfg -metadir /tmp/scalar-presence grpc/codegen/tla/scalar_presence/ScalarPresence.tla
```

The legacy configuration must fail `RequiredPresence` at initialization:
`supplied = FALSE`, `required = TRUE`. The presence configuration checks all
16 input combinations: 32 generated states, 16 distinct, depth 1, no errors.
This is a bounded representation/validation model, not a proof of protoc, Go
conversion, numeric ranges, byte-slice identity or network stream behavior. The model
assumes every scalar reaches this representation; root alias-chain Bytes
regressions check the actual traversal and compiled conversions.
The Go tests cover those implementation boundaries separately.

## Implementation and tests

- `grpc/codegen/protobuf.go`: root wrappers and nested scalar fields pass
  through the same physical scalar alias lowering, explicit field
  presence, and the protobuf Go context. Effective constraints and defaults
  remain owned by `expr`; native lowering does not choose new semantics.
- `grpc/codegen/protobuf_transform.go`: scalar message wrappers preserve
  pointer presence across conversion to/from service values.
- `codegen/validation_recurse.go`: collection elements and oneof scalar
  branches remain values; their enclosing representation owns presence.
- `codegen/validation.go`: empty lowered rules do not imply a callable
  validator. `TestValidationEligibilityWithEmptyRules` fails against the base
  implementation and prevents missing-validator references.
- `grpc/codegen/service_data_analysis.go` and `client_cli.go`: CLI decoding
  registers and invokes the same protobuf validator before service conversion,
  including when an empty flag leaves the message empty.
- `TestProtobufScalarPresence` fails on the old representation;
  `TestProtobufScalarAliasLowering` checks root and field normalization, retained
  constraints and no mutation
  of the service type.
- `codegen/generator.TestGRPCScalarPresence` extends the existing
  `DefaultFieldsDSL` specimen. It checks every scalar width, aliases and bytes,
  required/default combinations, root alias-chain Bytes, absence versus explicit zero through actual
  protobuf bytes, unary request/result decoding, service invocation, bidi input
  validation, client stream-result rejection, and CLI decoding.

## Bounded generated-output comparison

The acceptance comparison uses base commit
`0b7da36e` and the changed source, with Go 1.27.1, protoc 35.1,
protoc-gen-go v1.36.12 and protoc-gen-go-grpc v1.6.2. Generate each design in
an isolated module using `loom gen`, then `go mod tidy`, `go build ./...` and
`go vet ./...`. Select these fixtures rather than the full exported corpus:

| Input | Obligation |
| --- | --- |
| `grpc/codegen/testdata.DefaultFieldsDSL` | optional/default/required scalars; extended alias/byte/width and stream matrix |
| `PayloadWithAliasTypeDSL` | named scalar conversions |
| `PayloadWithMixedAttributesDSL` | required with and without defaults; unary and streaming |
| `StreamingValidationDSL` | root scalar wrappers, arrays/maps, stream envelopes and constraints |
| `MessagePrimitiveDSL` | scalar payload and result wrappers |
| `grpc/integration_tests/fixtures/quality/design` | nested required members, oneofs, errors, metadata, views and streams |

Compare every generated file byte for byte at the base and change, and compare
two independent process generations at the change. Intended differences are:

1. Required scalar declarations gain `optional`, pointer Go fields, presence
   descriptors and corresponding getters. Field numbers remain unchanged.
2. Converters address/dereference those fields; view pointers pass through;
   scalar wrappers use a typed temporary when encoding.
3. Required-field checks and nil-guarded value constraints appear in generated
   validators, along with newly needed nested validators. Decoders and stream
   receivers invoke them before conversion.
4. CLI builders validate decoded messages before conversion, and client types
   include the same request validators and their dependencies.
5. Physical alias lowering removes redundant native casts in validation without
   dropping constraints. Encoding preserves zero instead of default replacement.
6. `DefaultFieldsDSL` gains the scalar matrix and stream methods, their service
   and transport declarations, CLI flags, descriptors and design fingerprint.

Checked-in golden changes are generated from the same renderer. Existing
protobuf callers in test harnesses now supply pointers or use getters; their
behavioral assertions remain in place. No generated `gen/` files are edited.

Observed comparison: all six modules build and vet at both revisions. Of 91
files, 50 differ as described above; all 91 match exactly between independent
candidate generations. The affected gRPC codegen suite passed except for an
obsolete unused `proto` import in one test harness; removing that import and
rerunning `TestGeneratedRecursiveMapResultRoundTrip` passed. The shared codegen
suite and the generator's gRPC/protobuf test selection pass. Focused CLI, unary,
streaming and scalar-presence harnesses also pass with the pinned tools. Vet,
scoped golangci-lint and documentation checks pass.

The independent review found that root scalar wrappers initially bypassed native
lowering. An alias-chain Bytes root then compiled as `[]byte` in protoc but was
incorrectly treated as a pointer by conversion and validation. The direct root
normalization regression reproduces that failure. Root wrappers now traverse
the same lowering as nested fields; the extended scalar fixture compiles and
checks absent, present-empty, constrained and CLI values across that boundary.
The six-design comparison and independent-process determinism were repeated
following that repair, with the same file counts and no deterministic drift.
