# Logging serialization ownership — #750

Status: readiness approved by root; implementation and bounded acceptance complete at pushed parent `7b59759c5`. Final independent review pending. No commit or publication.

## Contract and shared cause

`FormatJSON` promises JSON entries, and `FormatText` promises quoted/escaped logfmt values. The hand-written JSON string/number branches bypass the stdlib serializer, while text fallback and terminal values bypass text escaping. A valid sibling field must survive an unsupported value. Preserve field order, duplicate field occurrences, numeric precision, duration/gRPC-code strings, existing ordinary formatting and the final newline.

## Sequence

1. [x] Retain direct parent failures for all ASCII controls, JSON keys/invalid UTF-8, scalar/nested nonfinite floats and typed-slice/fallback newline injection, with existing normal output controls.
2. [x] Establish the shared serialization boundary before implementation. JSON names and string values use the standard library's JSON quoting; value dispatch uses stdlib encoders, retaining the existing per-value diagnostic-string fallback on complex encoding failure. Text and terminal values share one escaping owner after presentation; no per-control escape patching. Keep redaction and logr sink changes separate.
3. [x] Record byte-for-byte normal controls and a bounded formatter benchmark at the parent. Resolve any actual semantic uncertainty with root before production. No model is proposed for this synchronous serializer change: no ownership, ordering or protocol state machine changes; stdlib parsing/round-trip tests establish the relevant boundary.
4. [x] Obtain independent readiness review. Then implement the shared owner and remove superseded hand-written escaping, documenting nonfinite/unsupported value diagnostics and invalid UTF-8 handling.
5. [x] Run focused/affected tests and race checks, ordinary exact-output controls, scoped lint and bounded parent/candidate benchmarks. Preserve ordered fields and per-value failure containment. No generator output changes or full corpus gate.
6. [ ] Freeze exact final diff for independent review; root owns integration/atomic commit. Clean temporary evidence after delivery, preserving hashes/results.

## Proposed details and limits for readiness review

The existing fallback converts complex JSON encoding failures to an escaped `<invalid: ...>` string; retain that behavior. For scalar floats, use Go 1.27's `jsontext.Float`/`Float32` tokens, whose JSON encoding represents NaN/+Inf/-Inf as the strings "NaN"/"Infinity"/"-Infinity". This standard owner also applies recursively through the existing []any presentation path. The initial AppendFloat proposal failed runtime verification: the local Go1.27.2 implementation emits raw nonfinite values despite its comment. The corrected token constructors are verified below; no custom nonfinite classifier is introduced. Keep scalar integer/bool fast paths from the standard library. Do not reimplement float special cases or character escapes. Invalid UTF-8 in names/string values should be replaced with U+FFFD via the stdlib's explicit permissive option, while every ASCII control is escaped. Preserve the existing special duration and gRPC-code presentation and scalar/array distinctions; do not accidentally switch []byte to a new display encoding or alter typed fallback semantics.

Text fallback presentation remains `fmt`-based, but its result must pass the same escaping boundary as ordinary strings. Terminal rendering should share that value boundary while preserving framework-generated ANSI decoration. JSON object field order and repeated field names must survive; encoding the entire entry as a Go map would discard both and is excluded. Text/terminal field names have no documented restriction that would justify raw unescaped writes. They therefore use the same quoting owner as presented values, with a retained parent failure for unsafe names.

The readiness tests validate bytes/framing and parsed JSON rather than asserting an unstable full stdlib diagnostic. They are not a universal guarantee for malicious or nonterminating application marshalers/Stringers.

## Readiness evidence

- `/tmp/loom-750-parent-red.log`: direct failures for ASCII controls, unsafe JSON names and UTF-8, scalar/[]any nonfinite values, typed-slice/map/Stringer text and terminal newlines. Typed float-slice fallback already emits a valid per-value diagnostic and is a retained control.
- `/tmp/loom-750-parent-controls.log`: existing exact text/JSON/terminal/timestamp output and added ordered/duplicate-field/uint64-max controls pass, 0.464s.
- `/tmp/loom-750-parent-bench.log`: six representative formatter/input combinations, three samples each, 22.141s. Benchmark source is retained beside regressions for unchanged parent/candidate measurements.
- The stdlib Go1.27.2 `encoding/json/jsontext/value.go` documents AppendFloat's JSON-string nonfinite behavior; `encoding/json/jsontext` exposes AllowInvalidUTF8 for valid replacement-based serialization. This local toolchain source informed the proposal.

Acceptance must preserve all existing normal output goldens and sibling fields around encoding errors, pass all new framing/JSON round trips, and record benchmark changes rather than claim unchanged cost without measurement. No output map conversion may erase field order/duplicates. Runtime formatting changes require no generated fixture corpus.

## Implemented ownership and intended differences

`jsonEntry.MarshalJSONTo` emits ordered names and values through the standard
streaming encoder. `json.Marshal` owns encoder reuse; there is no framework pool,
manual JSON punctuation, string-escape table or whole-entry map conversion.
Complex values are encoded independently first, so failure becomes a diagnostic
string without corrupting the enclosing encoder or losing siblings. Text and
terminal formatting share presentation/escaping; their integer fast paths and
`[]any` bracket presentation remain unchanged.

Intended output differences are: escaped ASCII controls and unsafe names;
replacement of invalid UTF-8 in names/strings; JSON-string nonfinite scalar floats;
standard finite-float spelling (including changed exponent notation); quoting of
text complex fallback values and terminal values that need escaping, plus unsafe
text/terminal names. Framework ANSI decoration remains. Existing ordinary exact
text/JSON/terminal/timestamp goldens, duration/code strings, duplicate ordered
fields and uint64-max precision pass unchanged. Nil/bytes/typed collections retain
their existing JSON presentation. Complex codec errors retain diagnostic fields.

## Final acceptance evidence

- Full affected race suites: `go test -race ./clue/log ./clue/debug`, pass
  1.594s/1.610s, `/tmp/loom-750-final-race.log`.
- Scoped pinned golangci-lint for both packages: zero issues,
  `/tmp/loom-750-lint.log`. `git diff --check` clean.
- Direct extra parent overlay retains unsafe text-name failures and scalar
  nonfinite failures: `/tmp/loom-750-additional-parent-red.log`; candidate full
  log package passes `/tmp/loom-750-extra-green.log` (0.449s). Controls include
  typed nonfinite diagnostic, custom codec error containing controls, nil/bytes,
  finite extremes and ordered duplicate fields.
- Runtime standard-library discrepancy probe is `/tmp/loom-750-appendfloat-probe.go`
  with output `/tmp/loom-750-appendfloat-probe.log`. No upstream report was posted.
- Final bounded benchmark: same six cases, three samples each,
  `/tmp/loom-750-entry-bench.log` (22.297s), summarized with baseline in
  `/tmp/loom-750-benchmark-comparison.json`. Timings are local measurements,
  not throughput guarantees. JSON allocation count is unchanged and bytes fall;
  safe text/terminal presentation costs additional allocations and CPU.

| Case | Parent median ns | Candidate median ns | Parent → candidate bytes | Parent → candidate allocations |
| --- | ---: | ---: | ---: | ---: |
| text/ordinary | 192.5 | 318.3 | 300 → 368 | 4 → 9 |
| text/structured | 241.5 | 666.5 | 348 → 488 | 7 → 16 |
| JSON/ordinary | 184.0 | 248.7 | 312 → 168 | 4 → 4 |
| JSON/structured | 233.6 | 377.3 | 360 → 232 | 6 → 6 |
| terminal/ordinary | 198.2 | 267.4 | 264 → 336 | 2 → 7 |
| terminal/structured | 398.8 | 499.9 | 296 → 448 | 4 → 14 |

No generator output changes, transport fixture generation or formal state model
is needed for this synchronous serialization boundary. Existing log/debug suites
cover consumers; this does not claim application codec/Stringer termination,
redaction, logr sink correctness or buffer-lineage correctness.

## Independent review repair

Root found MEDIUM: the first candidate treated an encoder nesting-depth error as
an entry-wide error and panicked, losing otherwise valid siblings. The parent
accepted the same finite 10001-deep `[]any` input. The regression now covers
9998/9999/10000/10001 nested arrays, both the direct `[]any` presentation and a
typed-slice fallback; it checks successful values below the entry-inclusive limit
and diagnostic containment above it, with both surrounding siblings retained.
A recursive `[]any` control naturally terminates at the same encoder limit; no
custom cycle detector or independent nesting policy was introduced.

`jsontext.Encoder.WriteToken`/`WriteValue` document that syntactic rejection leaves
state unchanged. The owning writer now replaces a rejected array start or raw
value with a diagnostic scalar at that position and continues the enclosing
entry. This uses the actual remaining encoder depth, including the outer object,
without hard-coded production limits. Existing complex codec failures use the
same diagnostic owner. No output panic occurs for these value errors.

Red: `/tmp/loom-750-depth-candidate-red.log` (first candidate); green:
`/tmp/loom-750-depth-green.log` (0.491s). The reviewer's exact overlay also passes
in `/tmp/loom-750-root-depth-fixed.log`. Revised race/lint/benchmark results are
recorded in the revised manifest; original evidence above remains historical.

Revised full log/debug race pass 1.745s/1.626s; pinned scoped lint zero issues;
reviewer's exact overlay pass 0.469s. Revised benchmarks pass 21.711s, with
allocation counts/bytes unchanged by the containment repair; all three samples
and medians are retained in `/tmp/loom-750-revised-benchmark-comparison.json`.
The ordinary output goldens still pass. Final re-review pending.
