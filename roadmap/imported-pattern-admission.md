# Imported pattern admission (#678)

Parent: `89920d3f08769b912785ecd90a08a709ab4faf50`.

1. [x] Reproduce unevaluable imported lookahead and cross-engine whitespace drift;
   preserve ordinary accepted patterns and fatal diagnostics under allow-lossy.
2. [x] Review the existing export pattern owner and its bounded proof. Define a
   shared portable grammar and direction-specific translation before editing.
3. [x] Obtain independent readiness review of the grammar, tests and acceptance rule.
4. [x] Move the existing Go-to-ECMAScript owner into a private internal package and
   add import admission there. Preserve export bytes. Import reports a fatal
   diagnostic at the authored pattern before rendering an invalid design.
5. [x] Compare accepted imports with actual Go and Node matching, evaluate generated
   designs, compile representative outputs and compare separate-process bytes.
   Extend the existing finite proof only for its stated witness domain.
6. [ ] Independent exact final diff review, atomic commit, push and issue closure.

The contract is preservation of acceptance, not merely Go compilation. OpenAPI
patterns use ECMAScript syntax without flags; Go patterns use Go regexp semantics.
The existing exporter already limits the supported bridge to Unicode scalar
strings, positive BMP character sets, ordinary grouping/alternation/repetition,
anchors and ASCII word boundaries. Import must share this owner instead of
copying source text or adding unrelated transport checks.

Proposed import grammar admits ordinary BMP literals, positive classes, groups
and noncapturing groups, alternatives, valid bounded/unbounded repetitions,
anchors, ASCII digit/word classes and boundaries. A small lexical admission pass
rejects extensions whose Go parse could silently change meaning: lookaround,
backreferences, flags, named captures, identity/octal/control escapes, wildcard
and complement classes, POSIX classes, non-BMP/surrogate characters, and malformed
or ambiguous repetition/class forms. The Go syntax parser then validates the
remaining grammar and the existing portable-tree check validates its domain.

Direction-specific escapes must be explicit. ECMAScript `\s` includes vertical
tab, NBSP and other Unicode whitespace; translate it to its exact positive BMP
class, including inside classes. Translate four-digit Unicode escapes to Go
literal/hex spelling without introducing unsupported braced hex syntax. Inside
a class, `\b` means backspace; outside it means a word boundary. Reject shorthand
range endpoints rather than guessing at engine-specific class range rules.
An initial unescaped class `]` must not inherit Go's literal interpretation:
reject empty/initial-close classes, preserving escaped closing brackets. Unicode
escapes produce escaped literal atoms, never raw regex punctuation; include
escaped backslashes before `u`, metacharacter escapes and literal class ranges.
Reject leading-zero repetition bounds, which Go can interpret as literals while
ECMAScript interprets them as counts. Quantified assertions need grammar checks.

This remains a bounded bridge, not a full ECMAScript implementation. Rejection
must explain the unsupported construct without discarding the authored constraint.
Common portable patterns retain their language; existing exporter fixtures must
remain byte-identical after moving the owner. No new runtime dependency or shell
JavaScript execution belongs in the importer. Node is a verification oracle only.

Sources: JSON Schema 2020-12 validation section 6.3.3,
https://tc39.es/ecma262/multipage/text-processing.html#sec-patterns,
and https://pkg.go.dev/regexp/syntax. Existing proof and engine checks live under
`http/codegen/openapi/internal/ir/lean/pattern_projection` and
`pattern_projection_test.go`; inspect their exact assumptions before extension.

## Acceptance evidence

- Parent direct admission tests fail for unsupported constructs and whitespace/
  Unicode/class-backspace mismatches. Parent generated ordinary fixture passes;
  whitespace fixture loses vertical tab acceptance; Unicode fixture fails DSL
  evaluation on an invalid Go escape. These are behavioral reds, distinct from
  preliminary harness or probe compilation errors.
- Focused race checks pass: shared grammar 2.888s, importer admission including
  normalized nested/nullable occurrences 2.282s, existing export owner 15.432s.
  Actual Node and Go agree for 353 admitted patterns over 52 representative
  Unicode scalar strings; ambiguous/unsupported syntax is rejected explicitly.
- Three generated applications compile and actual generated request/response
  validators pass accepted/rejected ordinary, whitespace and Unicode/backspace
  values with race checking (81.013s). JSON and YAML specs validate with libopenapi
  and pinned Vacuum rules: zero errors, 20 existing descriptive/example warnings.
- Independent processes produce 51 identical design/generated artifacts. Of 34
  parent artifacts, 27 are unchanged: all 17 ordinary artifacts and ten whitespace
  artifacts. Seven whitespace differences are intended: normalized design,
  manifest digest, JSON/YAML patterns, server/client validators and CLI validator.
  Unicode has 17 newly generatable artifacts; parent DSL evaluation fails.
- Existing export implementation moved without algorithm changes; only package,
  function visibility/name and caller references change. The bounded Lean model
  proves five witness-table theorems without axioms; it does not prove the parser
  or arbitrary regex equivalence. Focused lint passes without suppressions.

Local review evidence: `/tmp/loom-678-artifact-comparison.json`,
`/tmp/loom-678-final-focused.log`, `/tmp/loom-678-generated-parent.log`,
`/tmp/loom-678-generated-validators-final.log`, `/tmp/loom-678-vacuum.json`,
`/tmp/loom-678-pattern-proof.log` and `/tmp/loom-678-lint-final.log`.
