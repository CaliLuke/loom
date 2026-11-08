# Text-union assignment error provenance

`BranchErrors.tla` follows one error from a text-union candidate's assignment
closure through the public `TextBranch.Decode` runtime and exact-one selection.
It is an opt-in design aid, not a build, test, hook, CI or release requirement.

It composes two existing contracts instead of enlarging either model:

- [`BranchMatch`](../../../internal/httpencoding/tla/parameter_codecs/README.md):
  every candidate's schema and decoder predicates run independently, and the
  first callback error fails the match with its cause (`FatalPropagated`).
- [`NestedFailure`](../../../pkg/tla/untagged_json/README.md): an ordinary
  contract mismatch rejects one candidate, while an unexpected failure is an
  error.

## Decisions

1. **The runtime is transparent.** `TextBranch.Decode` returns `Assign`'s result
   unchanged, as its documentation, `ComposeTextUnion` and `unionmatch.Mismatch`
   state: mismatches are `(false, nil)` and errors propagate with their causes.
   The runtime cannot tell a client mismatch from a configuration, codec or
   server-fault error in a hand-written branch (`UnknownErrorRetained`,
   `FaultFails`).
2. **The generated adapter reports its closure failures as mismatches.**
   Untagged admission rejects Go type replacements anywhere in a branch
   (`expr/untagged_validation.go`), and only a custom text replacement makes a
   parameter default fallible. A generated closure therefore holds only
   conversion and validation of supplied text, and returns `(false, nil)` on
   failure (`MismatchIsNotFatal`). It needs no fault classification.
3. **An error from any candidate fails the match.** `Select` keeps its
   independent schema and decoder predicates and fails fast on a callback
   error, so a hand-written branch's error, including a server fault, fails the
   request even when another candidate is the unique match
   (`errors-unselected-witness.cfg`). Skipping a decoder because its schema
   failed, or discarding an error from an unselected candidate, would change the
   established contract (`schema-gated`, `drop-unselected`). Generated branches
   apply absent-member defaults within their own candidate, and those defaults
   cannot fail.

## Configurations and results

Run 2026-11-06 with TLC 2.19 (`tla2tools-1.7.4.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`), Java
25.0.5, one worker, `-Xmx256m`. Every run completed in about a second, without
parse errors or timeouts.

| Configuration | Policy | Result | Generated / distinct |
|---|---|---|---|
| `errors-checked.cfg` | generated adapter returns `(false, nil)`, runtime transparent | exit 0: all five invariants pass | 592 / 448 |
| `errors-runtime-suppress.cfg` | R3: generated adapter returns its raw error, runtime keeps only faults | exit 12, `UnknownErrorRetained`: a hand-written `unknown` error is dropped | 380 / 372 |
| `errors-runtime-swallow.cfg` | runtime turns every `Assign` error into a mismatch | exit 12, `FaultFails`: a hand-written fault becomes a reject | 452 / 452 |
| `errors-raw-adapter.cfg` | generated adapter returns its raw error, runtime transparent | exit 12, `MismatchIsNotFatal`: a supplied-value mismatch blocks the unique match | 151 / 151 |
| `errors-schema-gated.cfg` | decode only schema matches | exit 12, `FaultFails` | 399 / 391 |
| `errors-drop-unselected.cfg` | discard errors outside the schema match | exit 12, `FaultFails` | 399 / 391 |
| `errors-unselected-witness.cfg` | checked | exit 12, `NoUnselectedFaultFailure`: a hand-written fault in candidate A, which is not the schema match, fails a request that generated candidate B uniquely matches | 207 / 207 |

Exit 150 (parse error) or 75 (TLC exception) means the model is broken, not that
a counterexample was found.

## Run

From this directory, with a task-owned metadir outside the repository:

```sh
for cfg in errors-*.cfg; do
  "$JAVA" -Xmx256m -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 \
    -config "$cfg" -metadir "$META/${cfg%.cfg}" BranchErrors.tla
done
```

## Correspondence

| Model element | Implementation owner and regression |
|---|---|
| `Runtime = "transparent"` | `http/param_branch.go`: `TextBranch.Decode`; `TestTextBranchDecodePropagatesAssignErrors` |
| `Adapter = "mismatch"`, generated `Kinds` | `http/codegen/untagged_text_match.go`: `renderTextUnionMatcher` returns `false, nil` from the closure; `expr/untagged_validation.go` admission; `TestTextUnionAssignClassification` |
| hand-written `unknown` and `fault` | public `TextBranch.Assign` callers |
| `Try`, `Commit` | `internal/unionmatch/select.go`: `Select`; `http/union_text.go`: `MatchUntaggedText` |
| unselected-candidate error | `TestTextUnionCandidateErrorFailsMatch`: a hand-written `TextBranch` candidate's fault or unknown error fails `MatchUntaggedText` with its cause while another candidate matches |

## Assumptions and limits

Two candidates, one assignment outcome each, and no schema-callback errors.
`kind` abstracts the closure: the model does not parse text, run codecs or
materialize values. It assumes admission keeps fallible defaults out of
generated untagged branches; if admission changes, the generated kinds and
decision 2 must be revisited. Nested unions are not modeled;
`ComposeTextUnion` already propagates evaluator and assignment errors. No
liveness, concurrency or Go correctness is claimed.
