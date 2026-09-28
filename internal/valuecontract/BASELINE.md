# Initial comparison evidence

The #566 baseline comparison passed all 24 probe characterizations on
2026-09-28. This establishes reproducible existing behavior; it does not fix the
seven characterized failing cases or prove validity for every design.

## Source and environment

- Published baseline: `f5b786b39675e2b5c1466f04f3301b7b337779b6`.
- Candidate HEAD: `ff7873dfff1ad3803c421e1e64978906ee1139ce`, with the
  uncommitted harness captured in an immutable source snapshot.
- Candidate/common content ID:
  `6aad24571cb868e1826a0d640a526b47c092c8febef14caebfa0b28397c70bc9`.
- Selected manifest SHA-256:
  `0894722678aef1c7431fb6325e02629cab447c4857a6d85f84f7c91647e5e0ce`.
- Go `go1.27.1 darwin/arm64`; protoc `35.1`; protoc-gen-go `v1.36.12`;
  protoc-gen-go-grpc `1.6.2`.

Run the [README command](README.md) with this baseline and a unique results
directory. The accepted local run used
`LOOM_VALUE_RESULTS=/tmp/loom-value-contract/baseline-566-4`,
`GOCACHE=/tmp/loom-tickets-build-cache`, the tool directory
`/tmp/loom397-protoc35/runtime/bin` on PATH, and the candidate checkout as both
`LOOM_VALUE_CANDIDATE` and `LOOM_DIR`. Its command additionally used `-v`.
The full log is `/tmp/loom-value-contract-566-4.log`; it exited zero in 616.73
seconds. These local paths retain detailed evidence, not required repository
inputs. Re-running produces fresh exact bytes and reports.

## Observed results

All 96 records are present: 24 probes, two revisions, two isolated runs each.
The 24 cross-revision difference reports are empty. Repeated artifact bytes,
decoder observations and actual common-input hashes match. There are 2,128
retained artifact entries across the runs. Final source-snapshot verification
passed. No output normalization or intended-difference allowance was used.

| Probes | Observed outcome on both revisions, both runs |
| --- | --- |
| `bytes-text` | Decoder rejects its advertised example; #565 |
| `nested-union` | Decoder rejects its advertised example; #566 reproduction |
| `grpc-type-union`, `grpc-type-plain`, `grpc-payload-union`, `grpc-payload-plain` | Decoder replaces the authored value; #434 |
| `map-collision` | Generation rejects duplicate JSON member `1`; #456 |
| `bytes-binary`, `bytes-empty`, `grpc-integer-limits` | Generation, example stubs, build, vet and decoder pass |
| `presence`, `effective-occurrence`, `custom-codec`, `suppression` | Generation, example stubs, build and vet pass |
| `mapped-names`, `type-identity`, `http-defaults`, `jsonrpc-defaults`, `documentation-bodies`, `views`, `protojson`, `recursive` | Common specimen generation, example stubs, build and vet pass |
| `http-ticktock`, `jsonrpc-ticktock` | Full handwritten fixture regeneration, build and vet pass; existing application bootstrap retained |

Phase totals are 92 successful generation/build/vet/tidy runs each, 84 successful
example-stub runs, 12 successful decoder runs, 24 expected decoder failures and
four expected generation failures. Build, vet, setup and infrastructure failures
were zero. Expected failing subprocesses retain their actual nonzero statuses
and diagnostics; only the outer characterization passes.

## Controls and superseded diagnostics

The final uncached helper/corpus checks passed:
`go test ./internal/valuecontract ./internal/testdatacompile -count=1 -v`.
Controls reject changed artifacts, stale allowances, wrong phase/baseline,
handwritten-input mutation, torn/mutated snapshots and revision-local specimen
leakage. A failing/passing regression also proves package-path rewriting preserves
authored Example/Enum/Default strings while rebinding imports and package metadata.
Independent review confirmed all three source/input/rewrite findings resolved
before the accepted run. The exact final docs and evidence receive a separate
review before commit.

Earlier runs remain diagnostic evidence, not acceptance. Run 1 exposed a harness
test-package mismatch; run 2 exposed invalid draft Nullable DSL and absent nested
fixture modules in the downloaded root module. Run 3 reached all 24 probes but
failed JSON-RPC fixture compilation because `loom example` added `http.go` beside
the handwritten `jsonrpc.go`, duplicating `handleHTTPServer` and `errorHandler`.
The corrected full-fixture workflow runs `gen`, `tidy`, `build` and `vet`, while
fresh probe modules still run `example`. Run 4 verifies that correction on both
full fixture apps under both revisions twice. Logs and result directories use
the same local naming with suffixes `-1`, `-2` and `-3`.
