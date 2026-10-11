# Go pattern projection boundary

Run `./check.sh` with the pinned Lean 4.34.1 toolchain. The manifest audits each
listed theorem's transitive axioms; all three pass without axioms.

The finite witness model reproduces the former absolute-anchor and whitespace
mismatches, then checks the proposed target acceptance table. Numeric witnesses
stand for `foo`, `Afooz`, `foo` followed by LF, space, tab, vertical tab and NBSP.
The theorem is about this modeled table, not a proof of either engine, the
renderer or arbitrary regular expressions. `pattern_projection_test.go` compares
actual Go and Node matching, including rejected and adjacent syntax.

The production owner is `pattern_projection.go`, called before schemas are
fingerprinted or rendered in `analyzer_validation.go`. Go Pattern semantics stay
unchanged. Export translates `\A` and `\z` to unflagged `^` and `$`, and `\s` to
Go's explicit ASCII whitespace set (including inside character classes).
`\a` becomes Unicode-escaped BEL. Ordinary BMP literals, positive BMP classes, ASCII `\d`
and `\w`, boundaries, grouping, alternatives and quantifiers retain their
acceptance semantics. The target renderer consumes the parsed Go tree: classes
use resolved rune ranges, literal braces stay escaped, and quantified assertions
are grouped. Canonical spelling can differ without changing acceptance.

The admitted domain is Unicode scalar strings and ECMA matching without flags.
Wildcard newline behavior, non-BMP character sets (including ordinary negated
classes and `\S`, `\D`, `\W`), flags, named captures, POSIX classes, Unicode
properties, quoted literals, octal and braced hex escapes fail export with a
diagnostic. No undocumented semantic approximation is permitted. This is a
bounded preservation policy, not a claim that OpenAPI's SHOULD-level dialect
recommendation makes every Go extension a specification violation.

`Analyzer.Err`, `BodyTypes.Err` and `BuildDocument` carry failures to `Files`;
legacy helpers without error returns fail instead of rendering partial schemas.
Rendered tests check both targets. The selected existing portable fixture bytes stay unchanged. Absolute anchors and ASCII whitespace are the only intended
semantic pattern corrections; unsupported projections are newly rejected.
