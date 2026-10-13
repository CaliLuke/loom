# gRPC metadata absence/default ownership — #690

Source parent: `063198a2f4272d86e8c055714f7fab1ee9181359`.
Managed candidate: `/Users/luca/.codex/worktrees/explicit-presence-contract/loom`.
Managed comparison parent: `/Users/luca/.codex/worktrees/metadata-defaults-parent/loom`.

This is the first scoped M7 repair. The accepted contract is that defaults fill
absence, never an explicitly supplied zero or empty value. It does not resolve
JSON omission, protobuf collection presence (#684), tagged nullable unions
(#824), or binary metadata key/framing (#705).

## Sequence

1. [x] Reproduce #690 through generated request and client response decoders.
2. [x] Review the shared presence/default model, effective-default analysis,
   physical variable representation and all request/header/trailer/admission
   consumers. Obtain independent scoped readiness approval.
3. [x] Add parent controls for booleans, bytes, named strings, string/integer
   arrays, malformed input, required defaults and optional no-default fields.
4. [x] Replace repeated type-specific presence decisions with one metadata
   lookup/default application owner; retain the existing text and bearer codecs.
5. [x] Add mounted wire tests and the transport-local pointer-default edge;
   compile/vet generated modules and compare parent/candidate/isolated repeat.
6. [x] Complete final affected checks.
7. [ ] Obtain fresh exact-diff review.
8. [ ] Root integrates, obtains any integration delta review, commits one atomic
   #690 repair, pushes and closes only on matching delivery evidence.

## Owner and accepted examples

`extractMetadata` already resolves the effective occurrence's default for CLI
metadata. It now also supplies a typed optional default literal through the
shared `AttributeContext.DefaultValue` renderer, using the existing unaliased
metadata variable representation. The constructor still owns conversion into
named service fields. Required fields never receive a default literal.

`renderGRPCMetadataDecode` makes one lookup/presence decision. Supplied entries
use the existing scalar, repeated-value or bearer parser. An absent required
entry produces the same missing-field error; absent optional metadata receives
its selected default or stays absent. Response headers/trailers, ordinary
requests and early stream admission use this function. The former independent
string/slice/numeric/bearer presence branches are removed.

Accepted: missing optional `limit` with Default(3) becomes 3; supplied `"0"`
becomes 0; supplied `"false"` and `""` survive; malformed `"bad"` remains invalid.
Missing required metadata remains invalid even with a default. A metadata-local
Int32 default with a pointer service field creates a typed pointer only when
absent; supplied 0 remains0. Original type inference requires an explicit Int32
in that transport-local DSL declaration; the probe supplies it rather than
altering unrelated DSL inference.

Repeated metadata retains its existing text codec: zero entries means absent,
while one empty string entry is an explicit string element. This repair does
not make an empty array distinguishable from absence. That unresolved wire
representation remains a separate M7 obligation. Printable byte values are
covered on real gRPC wire; nonprintable framing remains #705. No compatibility
mode, public API addition, protobuf schema change or transport encoding change.

## Evidence and reproduction

Use Go 1.27.2 and the repository's pinned dependencies. External modules pin
`github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427`
(verified pushed parent) and explicitly replace it with the source under test.

```sh
LOOM_DIR="$PWD" go test ./grpc/codegen -run '^(TestMetadataDefaultsGenerated|TestMetadataDefaultPresenceOwner|TestMetadataDefaultLocalOccurrence)$' -count=1
```

The generated test builds/vets all emitted packages, then runs the direct and
mounted decoder harnesses with the race detector. It checks request admission,
response headers and trailers from generated and ordinary protobuf peers,
status/details, service invocation, nil no-default fields and payload state.

Parent evidence: `/tmp/loom-690-expanded-parent.log`, 4.609s, 14 expected failures
across absent defaults for integer/string/bool/bytes/named string/string-array/
integer-array in request and response. All explicit zero/empty/ordinary,
malformed and required/no-default controls pass. The original two-type #690
witness was also red before expansion. `/tmp/loom-690-direct-parent.log`
records the missing shared absence branch. Local pointer-default parent
reproduction is `/tmp/loom-690-local-parent-red.log`; its overlay removes only
candidate-field structural assertions, retaining the identical runtime test.

The approved broader M7 model preceded implementation. Its metadata slice is
retained under `grpc/codegen/tla/metadata_defaults`: current behavior fails
DefaultOnAbsence in 22 states, corrected presence passes 40. It does not prove
literal typing or actual codecs; compiled/wire tests establish correspondence.

The bounded comparison uses five designs: the expanded default matrix, a
transport-local pointer default, existing MessageWithMetadataDSL, existing
NamedMetadataDSL and existing StreamAdmissionAlternativesDSL. All parent and
candidate outputs compile/vet. Separate processes regenerate the candidate;
repeat output is identical. `/tmp/loom-690-comparison/hashes.json` records exact
per-artifact hashes; `/tmp/loom-690-comparison_test.go` and its parent/candidate
overlays preserve the command inputs until review. Build/vet is not repeated
for identical regenerated outputs.

Intended output differences are confined to decoder files: new absence-default
branches; equivalent supplied-first required checks; and the shared `vals`
lookup for repeated values. Encoding, protobuf schemas, service types and
admission-alternative artifacts remain unchanged. One checked-in request
metadata golden changes only repeated-value temporary variable naming.

Existing named defaults, mapped metadata, credential typing, metadata-name
collisions, stream admission alternatives/controls and request/response codec
goldens passed the focused selection in 10.184s. Scoped lint reports 0issues.
Final receipts below supersede checks invalidated by the last harness edits.
No full exported-design corpus or unrelated M7 red tests are an acceptance gate.

## Atomic delivery boundary

Include only the three production files, metadata default tests, one affected
golden, the scoped metadata model/configurations/README, this plan and the gRPC
guide plus framework skill/model-index updates. Broader untracked M7 investigation tests/models and
`roadmap/explicit-presence-readiness.md` remain outside this atomic diff. The framework skill and formal model index record the shared owner. Existing
consumer skill guidance already says defaults fill absent optional fields and
metadata inherits the service default; the public gRPC guide supplies the
metadata-specific examples. Root owns master tracker integration.

## Final acceptance receipt

- Five designs produce 55 artifacts. Parent/candidate/repeat hashes cover every
  generated artifact; candidate and repeat are byte-identical. Exactly six
  files differ from parent: default request/client decoders, local-default
  request decoder, named request/client decoders (equivalent required branch
  ordering), and normal metadata request decoder (temporary variable naming).
  The remaining 49 artifacts are unchanged. All five parent and candidate
  modules build/vet successfully; repeat generation reuses that compile proof.
- Mounted request/response/peer cases plus the full decoder value matrix pass
  with race detection in 5.217s (`/tmp/loom-690-final-mounted.log`). An added
  assertion initially expected an unsafe raw field name in the malformed-value
  status message. The existing public contract deliberately says "validation
  error"; the corrected assertion checks that safe message plus the exact
  invalid_field_type problem detail. Required omission still checks
  missing_field and the documented missing-field detail. This was a harness
  expectation correction, with no production change.
- The transport-local pointer default is independently red on parent in 4.074s
  (absence nil; explicit 0/ordinary pass), and compiles/vets/runs with candidate.
- Model sources and all production/generation inputs were unchanged by the
  final assertion/scaffold edits. The earlier comparison, auth/admission,
  mapped/named, collision and golden checks remain valid. Latest local seam
  and lint logs are `/tmp/loom-690-final-local.log` and
  `/tmp/loom-690-final-lint.log`.
- Shared framework skill and formal-model index updates are included. Existing
  consumer skill defaults-fill-absence wording remains consistent; the gRPC
  guide records metadata examples and regeneration. No consumer skill edit is
  needed.
- Final local seam/generated check passes5.958s and final scoped lint reports
  zero issues. A separate preserved owner-control run passes8.465s
  (`/tmp/loom-690-owner-controls.log`); this replaces the overwritten aggregate
  log as durable evidence of auth/admission, named/mapped, collision and golden
  controls. No production or generation input changed during these checks.

Freeze metadata: `/tmp/loom-690-final-manifest.json` and
`/tmp/loom-690-final.patch`. Temporary build/checker trees must remain until
independent review and root integration finish, then be removed after preserving
compact hashes and command receipts. Reusable installed toolchains stay intact.
