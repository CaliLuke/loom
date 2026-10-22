# Compilation value ownership

This bounded model compares ownership of service value occurrences and selected
examples across service code, transports and documentation. It informed
`SemanticData`, backend-local `ValueData` records, and `ExamplePolicy.SynthesisScope`.
It does not prove memory bounds or the correctness of Go graph capture.

## Contract and counterexamples

One generation captures each method/role occurrence once. Backends retain the
same semantic owner but select examples independently. Each backend has its own sampling policy, including the custom randomizer
stream position. Documentation may also suppress synthesis. Sharing an occurrence does not authorize sharing a mutable selection
record or a cache entry across different sampling policies.

- `Separate.cfg` reproduces the previous separate-owner architecture and violates
  `OneSemanticOwner` (7 generated/distinct states before the counterexample).
- `Mutable.cfg` shares mutable selection records and violates `LocalSelection`
  (58 generated, 32 distinct states).
- `Scope.cfg` isolates records but shares the synthesis cache across policies. It
  also violates `LocalSelection` (58 generated, 32 distinct states). This models
  the first implementation's changed OpenAPI parameter examples and the
  review finding about skipped custom randomizer draws in transport analysis.
- `Shared.cfg` shares semantics, isolates records and keys the synthesis cache by
  sampling policy. Both invariants and `Completes` pass: 217 generated states,
  81 distinct states, depth 9.

Four bounded consumers stand for service, transport, documentation and suppressed
documentation. Capture and selection are atomic. Weak fairness of `Next` models
completion of this finite sequential pipeline. Terminal deadlock checking is
disabled because completion has no outgoing action; the temporal property still
requires every consumer to select. There is no modeled external failure.

## Run

Use Java and `tla2tools.jar` from the official TLA+ v1.7.4 release (TLC 2.19),
SHA-256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, with the jar and checker state outside the repository:

```sh
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$TLC_STATE" -config Shared.cfg SemanticOwnership.tla
```

Repeat with `Separate.cfg`, `Mutable.cfg`, and `Scope.cfg`; each must fail its
named invariant, not parsing or resource checks. Do not retain checker logs,
traces or state in the repository.

## Implementation correspondence and limits

- `codegen/generator/compilation.go` creates owners after plugin preparation.
- `semantic_data.go` captures occurrences and returns detached backend records.
- `value_data.go` retains backend selection and Go naming is in `ServicesData`.
- `expr/value_synthesis.go` keys retained synthesis by occurrence and scope;
  authored resolution remains occurrence/source/role based.
- `semantic_data_test.go` checks shared identity, independent records/scopes,
  snapshots before backend mutation, and foreign-method rejection.
- `expr/value_synthesis_scope_test.go` checks concrete sampling isolation and
  suppression. `http/codegen/openapi/v3/semantic_data_test.go` compares complete
  JSON/YAML for parameters, selected bodies, recursion, views, streams and errors.

The model omits graph topology, actual generators, naming, authored examples,
custom codecs, allocation sizes, plugin mutation and filesystem output. Existing
value-contract models and Go tests cover separate obligations; their guarantees
are not implied by this small ownership model. Component-only and independent
transport/documentation occurrences are outside this method-value owner.
