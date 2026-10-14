# Raw text validation carrier (#939)

Source archive: `9e0b16e12398d7ab99f9f862d786cca4d8be0be8`.
Integration follows the nonoverlapping #937 repair and documentation checkpoint;
its exact intended parent is recorded in the frozen manifest.

The shared HTTP transport attribute builder selects raw String text for authored
TextUnmarshaler validation, but its old format-only projection retains custom Go
storage metadata. #926 then correctly follows that incorrect carrier metadata.
Fix the carrier projection once, consumed by HTTP and JSON-RPC request metadata,
response headers/cookies and HTTP CLI arguments. Do not weaken custom validation.

## Sequence and acceptance

1. [x] Reproduce the six unchanged golden failures and two raw-value placement
   failures. Preserve the exact parent log and CI witness.
2. [x] Add a direct projection regression for custom/native and named inherited
   constraints, metadata removal, format delegation and source nonmutation.
3. [x] Replace the format-only helper with one fresh native String validation
   attribute using effective non-format constraints. Render according to that
   carrier, preserving original parser, presence and destination representation.
4. [x] Run direct regressions and existing server/CLI goldens without changing
   expected output. Generate a bounded custom path/query/header specimen and a
   native control, compare parent/candidate and process repeats byte for byte,
   and run existing generated custom CLI/wire plus scalar presence controls.
5. [ ] Obtain independent exact review, integrate one atomic commit, push and
   close #939. Preserve reproduction/evidence before inactive scratch cleanup.

This is a deterministic carrier projection, with no lifecycle, ordering or new
protocol state; no new formal model is required. Existing scalar-presence and
custom-representation models remain unchanged because their policy is preserved.

Expected output changes remove only redundant semantic projection of raw native
strings from custom-text validation. Existing pre-#926 goldens remain the desired
contract. Native controls and custom service storage remain unchanged; parser
format ownership stays explicit and inherited non-format constraints survive.

## Bounded validation results

- The direct carrier regression fails on the parent for retained metadata and
  inherited physical type. The corrected carrier preserves enum conjunctions,
  pattern clauses and minimum/maximum lengths, delegates all format rules and
  leaves original declarations detached. Native, custom without format, direct
  format and inherited-format controls pass.
- Existing `TestDecode`, `TestClientCLIFiles` and
  `TestTextUnmarshalerDecodeValidationPlacement`, plus the direct regression,
  pass with races (7.110s). No golden expectation changes.
- Five generated specimens use existing fixtures: custom path, optional query,
  optional header, native query/default, and constrained response headers/cookies.
  Of 66 Go artifacts, only six custom request server/CLI files change; they remove
  the redundant `StringValue` projection while preserving predicates and guards.
  The other 60 are byte-identical, including service storage and response output.
  All 66 repeat exactly in independent generation processes.
- Selected `TestCustomTextClientGeneratedModules` path/query/header cases,
  `TestCustomTextClientStringer` query/cookie cases and
  `TestScalarBindingPresenceGenerated` pass with races (42.135s), compiling and
  exercising valid/invalid custom text, optional absence and defaults.
  `TestCustomResponseRepresentationWire` passes with races (4.698s).
- Package golangci-lint reports zero issues. Final exact independent review and
  delivery remain pending; no broad exported-design audit is part of this repair.

Use `GOEXPERIMENT=jsonv2` and `LOOM_DIR=/absolute/path/to/loom` for generated
local-source checks. The direct/package command is:

```sh
go test -race ./http/codegen -run '^(TestRawTextValidationCarrier|TestTextUnmarshalerDecodeValidationPlacement|TestDecode|TestClientCLIFiles)$' -count=1
```

Generated consumer checks use the named tests above, with the three stated
`TestCustomTextClientGeneratedModules` subtests. Existing scalar-presence and
custom-representation model sources are unchanged; this projection corrects their
physical-carrier input without changing their lifetime or propagation contract.
