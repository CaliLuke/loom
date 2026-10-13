# CLI JSON decoding role — #838

Parent: `7b59759c58a237a925750aad7bc412ce7a41e004` (pushed main when this managed
worktree was created). This is an atomic prerequisite to #835. No #835 production
changes were transferred. Root independently approved the shared decode-role owner. Implementation and
bounded acceptance are complete; fresh final review is pending.

Issue: https://github.com/CaliLuke/loom/issues/838

The original readiness is frozen in `/tmp/loom-835-cli-producer-readiness-manifest.json`
SHA256 `a2eb7222ecd6ee7489225afb0b8b522bd3580528cd23eb546455cb6fb458d039`.
Its historical complete835 probe remains in the frozen presence-application-owner
worktree; a transfer copy is at `/tmp/loom-838-historical-readiness-inputs/`.
The scoped executable tests below now own the durable838 reproduction inputs. The smaller self-contained issue reproduction is runnable
as `TestCLIJSONRequiredPresence` in `codegen/generator`; it has a pushed module
pin and an explicit local source override. Original parent27255d025 generated
build/vet succeeded; required missing members failed for HTTP and JSON-RPC,
supplied false controls passed. Historical log `/tmp/loom-cli-body-issue-parent.log`.

## Contract and owner proposal

Inspection of the body argument, representation and conversion owners exposed a producer precondition missing from the
original readiness: CLI JSON flags decode into the client *encoding* body type.
`buildPayloadBodyArgs` chooses `GoType*WithDefaults`; `buildTransformCode` uses
`httpContext(request=true,server=false)`. That context materializes native
scalars before JSON decoding can preserve member absence. The client declaration
and nested-type registries also derive decoding/encoding from server placement.
Consequently a transform cannot distinguish `{}` from `{"flag":false}`.

New executable evidence: `TestDefaultApplicationCLIBody`, paired with the
existing three-transport design and an additional nested-object design, builds
and vets generated modules before checking both HTTP and JSON-RPC constructors.
At parent27255d025, explicit root/nested zeros and missing-required fail. Under
candidate835, explicit zeros pass, but root/nested missing members lose their
defaults; missing-required still fails. Controls preserve populated values,
missing outer objects, explicit empty bytes/arrays/maps, Nullable absence/null/
empty, and supplied required zeros. Logs:
`/tmp/loom-835-cli-body-parent.log` and `-candidate.log`.
This was the pre-implementation evidence; the final results below supersede it.
#835 remains a separate atomic default-admission repair.

### Proposed prerequisite owner and sequence

1. [x] Expose producer loss with actual generated CLI inputs, including nested
   missing members and required/default coexistence; preserve original835 red
   evidence and candidate independently.
2. [x] Model text decode → required admission → conversion. `http/codegen/tla/cli_json_input/CLIInput.tla`
   reproduces encoder absence loss, old zero substitution, and default-seeding
   hiding required absence. A decode carrier retains all five states and passes.
3. [x] Root independently reviewed and approved the expanded ownership proposal.
4. [x] Separate body representation role from package placement in the shared
   HTTP body representation/registry owner used by HTTP and JSON-RPC. An input
   decoded from CLI JSON needs the same presence/collection carrier and recursive
   validation obligations as any JSON request decoder, even in a client package.
   Allocate its detached declaration closure through existing NameScope/layout
   owners; distinguish it from client wire encoder types without forcing all
   clients to use decode layouts. Nested aliases, recursive named boundaries,
   unions and codec metadata must retain existing representation decisions.
5. [x] Make CLI argument type, validation and conversion derive from this one
   decoded-body plan. Validate requiredness before default materialization;
   convert admitted fields through835's shared default record. Remove the prior
   assumption that a CLI JSON argument is an encoding body. Public CLI syntax,
   scalar flag parsing and encoder wire representation stay unchanged.
6. [x] Extend bounded acceptance with decode-plan identity/detachment and
   declaration closure controls, generated CLI absence/zero/null/empty/required
   and nested tests, existing custom codec/union/optional body controls. Pair
   parent/candidate/repeat bytes and build/vet with actual CLI endpoint parsing
   and mounted service observations. New declarations/helpers and CLI load/
   conversion changes are intended; schema and encoder type/tag changes are not.
7. [ ] Fresh exact final review, then root decides atomic prerequisite delivery
   order and835 integration. No transport-local default seeding or zero checks.

This is an implementation prerequisite, not an unsettled public policy: the
accepted absence/default and requiredness contracts already determine behavior.
The model treats null as an admitted nullable input; rejecting null for ordinary
fields and malformed JSON remains a decoder/validator test obligation. Nested
values are repeated applications of the same occurrence contract; the finite
model does not prove recursive declaration or generated-code closure.


## Atomic boundary

Keep `TransformAttrs.DefaultForField` in835. This ticket owns CLI JSON decoding
representation and admission: one shared HTTP/JSON-RPC decoded-body plan drives
its declarations, nested physical layout, validation, argument type and conversion.
No seeding, native-zero checks, encoder-wide layout mutation or schema changes.
Keep the original835 candidate frozen until838delivers; then integrate835against
the delivered prerequisite and repeat only invalidated acceptance. Four unrelated
gRPC golden mismatches remain a separate test-alignment report.

## Transfer receipt

On7b59759c58a237a925750aad7bc412ce7a41e004, the self-contained issue regression
again builds/vets generated code then fails both missing-required cases while
supplied-false controls pass (2.953s). Log `/tmp/loom-838-main-red.log`.
At transfer no production files were modified. The readiness model is retained
exactly; #835 production remains in its frozen worktree.

## Final implementation and acceptance

`service_data_json_input.go` now owns the detached decoding attribute graph,
physical context, input declaration closure, validation and conversion. Existing
NameScope, body layout, named validation, union declaration and transform owners
remain authoritative. Recursive/named/custom-codec members use the same graph.
Both existing `--body` and bare collection `--p` inputs consume it; a CLI-only
initialization record leaves server initialization unchanged. The old client
encoding-to-payload helper path is removed for these JSON inputs. One shared
Optional-based runtime decoder rejects root null without a separate token walk.
The public flag syntax and ordinary encoding declaration/schema identities stay
unchanged. No #835 production code is included.

The existing CLI golden test now selects the named `cli-build-payload` section;
its old fixed index would have captured a new declaration instead of the builder.
Eleven goldens intentionally change carrier types, decoder calls, presence-aware
conversion/defaults and validation; the bare-array parser delegates to its builder.

- Parent missing-required: `/tmp/loom-838-main-red.log`, generated build/vet pass
  then both HTTP/JSON-RPC missing-member assertions fail; supplied false passes.
- Parent/newly exposed bare collection: `/tmp/loom-838-bare-collection-red.log`
  records acceptance of `null` and `[null]`; final tests reject both and preserve
  `[]` and `[""]` using the unchanged `--p` parser syntax.
- Runtime/direct/golden/Basic and optional-body controls:
  `/tmp/loom-838-final-direct-local.log` passes (HTTP/codegen10.085s).
  `/tmp/loom-838-final-owner.log` passes (.688s), including detached name collision
  and bare collection/server initializer separation.
- Actual generated CLI matrix: `/tmp/loom-838-final-generated.log` passes11.139s,
  including build/vet/race for the body harness. Covers absent/default/required,
  explicit zero/empty, nested absent/present, recursive named values, custom UUID
  codec, ordinary null rejection/Nullable admission, malformed/truncated/empty/
  whitespace JSON, untagged branches with and without predicates, and mounted
  HTTP/JSON-RPC status, service count and decoded payload. Invalid CLI input
  produces no endpoint/payload and no request/service invocation.
- Existing tagged nested union HTTP/JSON-RPC/gRPC round trips:
  `/tmp/loom-838-final-union.log` passes3.180s.
- Five bounded designs (`body`, `unions`, `untagged`, `field-name`, `query-control`)
  produce169artifacts/run. `/tmp/loom-838-final-comparison/hashes.json` records
  exact parent/candidate/independent-process repeat bytes; candidate and repeat
  match. `differences.json` and `output.patch` enumerate17intended changes:
  new CLI decode declarations/validation/conversion, superseded CLI helper removal
  from encode_decode, and bare collection parser delegation. All schema, encoder
  declarations/tags and the full query-control design are unchanged.
  Five parent modules build/vet8.195s; five candidate modules build/vet5.505s:
  `/tmp/loom-838-final-build-{parent,candidate}.log`. Repeat modules are identical
  and are not redundantly built. Generation logs use the corresponding prefix.
- Joint prerequisite closure: `/tmp/loom-838-with-835-overlay.json` overlays only
  the four frozen835 production owners onto838 (those parent files are unchanged
  between the bases). Actual generated body/build/vet/race matrix passes5.192s
  in `/tmp/loom-838-with-835.log`. No835 code is copied into this atomic diff.
- TLC reruns `/tmp/loom-838-model-*.log` reproduce four intended invariant failures
  and the60-state successful decode model. See the owning README for assumptions.
- Scoped golangci-lint2.14.0: `/tmp/loom-838-final-lint.log`, zero issues.
  Go1.27.2 with GOEXPERIMENT=jsonv2; temp modules pin pushed063198a2f427 and explicitly
  replace with their actual parent/candidate source. An initial control invocation
  without LOOM_DIR selected the remote pin and failed on the new helper; it was
  corrected and the local-source gates above supersede that invalid invocation.

Reproduce the bounded gates with `GOEXPERIMENT=jsonv2`, `LOOM_DIR` set to this
worktree and `go test -count=1`: `./http/cli ./http/codegen -run
'Test(Unmarshal|JSONBodyInputRole|CLICollectionInput|OptionalBodyCLI|ClientCLI|Basic)'`;
`./codegen/generator -run '^Test(CLIJSONBodyContracts|CLIJSONRequiredPresence|CLIJSONUntaggedInput|UnionInUnionRoundTrips)$'`.
For isolated output use `LOOM_CLI_JSON_OUTPUT=/absolute/output` with
`TestCLIJSONArtifacts`, then `TestCLIJSONArtifactBuilds`. Repeat generation in a
separate process and compare every `*/gen/**/*` file byte for byte.

Remaining boundaries: #835 fixes native supplied-value default admission in shared
transforms/ordinary encoders. #839 aligns separate stale gRPC collection goldens.
JSON omission, ordinary protobuf collection presence and #740 remain outside838.

## Final review correction

Independent review found no correctness issues and verified all507artifact hashes.
The LOW declaration-order finding is fixed mechanically: test helper types and
harness constants precede functions, private methods follow private functions,
and the custom JSON decoder method has GoDoc. No generated design or production
behavior changed. Focused decoder/owner/generated body+required+untagged tests
pass in `/tmp/loom-838-review-fixes.log`; existing output/build/model evidence is
unchanged. Targeted confirmation of these edits is pending.
