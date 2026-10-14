# Built-in problem reconstruction contract (#911)

Parent: `ce26ecbe9`. Preserve the documented pruned built-in problem contract.
The public ErrorResult describes a wire document. Its required fields are not
required fields of the service error rebuilt by ProblemErrorFromBody; omitted
properties intentionally default to false. Custom errors, payloads and results
retain their authored required-field reconstruction obligations.

## Sequence

1. [x] Reproduce the existing OpenAPI and generated explicit-object admission
   failure; inspect docs/error-handling.md and the actual client conversion.
2. [x] Add direct admission controls for default/explicit built-in pruned bodies,
   HTTP/JSON-RPC consumers and custom errors with the same field names. Confirm
   current built-in failures and preserve custom rejection.
3. [x] Correct the shared expression error-reconstruction boundary to select the
   actual reconstructed value's obligations. Keep wire body/schema requiredness
   and ordinary generic reconstruction unchanged; no generator-local exceptions.
4. [x] Run direct expression tests, existing mounted selected-problem case and
   ProblemProperties OpenAPI generation/lint/determinism. Compare bounded valid
   parent/candidate output and compile; preserve all existing projection tests.
5. [ ] Independently review the exact diff, commit/push one repair, close #911,
   reconcile the master inventory and clean inactive scratch artifacts.

No new public policy or formal state machine is introduced. Existing
http/codegen/tla/problem_response_projection defines the already accepted
projection contract; this repair restores admission correspondence to it.

Current negatives: `/tmp/loom-problem-fixture-current.log` and CI job
115639900832 (`/tmp/loom-ci-227-http-compile.log`). The two empty-result designs
have ordinary required fields and remain a separate fixture adoption task.

## Acceptance evidence

- Direct HTTP/JSON-RPC default, explicit ErrorResult and ProblemResult cases
  fail before the repair; custom same-shaped errors still reject omissions.
  Focused reconstruction races pass (1.622s).
- Existing mounted explicit-detail-object client/server case passes (4.295s);
  existing ProblemProperties OpenAPI JSON/YAML validation, lint and process
  determinism pass (4.504s). No golden or generated fixture changes.
- Immutable parent `ce26ecbe9` and candidate containing only the production
  repair generated the existing full/custom/projected problem specimens. Full
  and custom output remain byte-identical (16 files each). The 11-method
  projection specimen changes from admission failure to compilable output;
  candidate and separate-process repeat match exactly for all 48 artifacts.
  Accepted parent and candidate modules build and vet successfully.
- Inputs: `http/codegen/tla/problem_response_projection/*_design.go.txt`.
  Commands and byte hashes: `/tmp/loom-911-cli/commands.json` and
  `/tmp/loom-911-cli/artifact-ledger.json`. CLI binaries use identical names;
  temporary modules pin pushed commit 92e232503 with explicit local source
  replacement. Toolchain: Go 1.27.2, GOEXPERIMENT=jsonv2.

The existing projection model remains unchanged: this correction aligns DSL
admission with its established adapter semantics. No new policy is introduced.
