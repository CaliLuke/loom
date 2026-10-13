# Local OpenAPI reference identity (#891)

1. [x] Correct invalid original slash/tilde component specimens; preserve report history.
2. [x] Reproduce legal percent-encoded ordinary-name failure via actual import CLI and cleanup seam; ordinary control passes.
3. [x] Reuse owned managed worktree after #889 delivery, preserving its exact sources/evidence and inverse patch before fast-forward.
4. [x] Add one internal OpenAPI local-reference parser and direct rejection/normalization tests before production. Replace importer, IR and v3 identity parsing; prevent discriminator double decoding.
5. [x] Verify existing reference graph/selection/alias controls plus actual encoded import/generate/build/spec validity and unchanged ordinary output. Independent-process repeat generation. No new graph model or full corpus.
6. [ ] Freeze final exact diff for independent review; parent owns commit.

## Owner and contract

A fragment-only URI reference resolves to exactly three JSON Pointer tokens:
components, the requested component category, and its key. URI percent decoding
happens exactly once before pointer validation/token decoding. Reject wrong
category, external/document references, extra pointer segments, empty names and
malformed encoding. Preserve JSON Pointer parser semantics independently of the
OpenAPI component-name admission rule; do not redefine legal property names or
bundle general admission repairs. Canonical references emit stable unencoded
unreserved component names. Existing jsontext.Pointer and net/url primitives own
syntax; an internal/openapiref package owns the shared composition used by both
internal/openapiimport and http/codegen/openapi/{internal/ir,v3}.

Existing importer localComponentReferenceName callers include reference plans,
rendering, parameter/example resolution and selection. IR schemaComponentName
feeds component reachability, annotation/baseline identity, async materialization,
reusable-component naming and discriminator metadata. V3 schemaNameFromRef feeds
alias resolution and pruning. No graph algorithm changes. Discriminator currently
passes parsed.Fragment to the old helper: switch it to raw URI input so the shared
owner alone decodes it. Avoid a decode wrapper layered over duplicate parsers.

## Bounded acceptance

Parser cases: plain and percent-encoded letters/underscore, encoded prefix,
malformed percent escapes, wrong category, external/file references, additional
segments, malformed tilde syntax and exactly-once decoding. Existing pointer
escape controls remain parser tests, not admission assertions for invalid keys.
Importer: actual Analyze/render and CLI ordinary versus encoded named schema,
referenced parameter/selection and examples using existing seams where relevant.
Exporter: direct reachability and alias chain equivalent refs; ordinary aliases
and unused-schema pruning unchanged. Reuse existing IR graph/discriminator tests.
One actual header-item import/generation specimen demonstrates wire contract and
valid emitted references; build/vet plus existing spec validation. Compare ordinary
parent/candidate and encoded candidate/repeat output, explicitly record parent
encoded import rejection. No transport implementation changes or fixture sweep.

## Readiness evidence

At parent 59ac9947c, actual CLI rejects %49temName with source-context diagnostic;
ordinary ItemName imports. Read-only cleanup overlay on 0df1a49bc loses ItemName
and Item_Name for encoded refs, Plain passes. The underlying owner source hashes
are recorded in /tmp/loom-891-independent-assessment.json. The #889 worktree was reused after delivery, exact-source archival, inverse patch
and clean fast-forward to `051da453a`; the reuse receipt is retained.

## Implementation and final evidence

The shared `internal/openapiref.Name` parser replaces importer-local parsing;
IR and v3 adapt its error result to their existing boolean lookup contract.
Discriminator metadata passes the raw reference instead of a predecoded fragment.
All consumers retain their graph and admission policies. Unsupported component
keys remain outside this correction; pointer-level escape controls do not claim
that those keys are valid OpenAPI Components names.

Tests first reproduced importer, IR and cleanup failures on `051da453a`.
Final focused race checks pass for parser, import/selection/example resolution,
IR/discriminator, v3 aliases/pruning and existing reference graph controls
(1.127, 1.434, 1.408 and 1.595 seconds). The actual generated module test passes
(10.499 seconds), including parsing both emitted specs with libopenapi. An initial
selection probe used unsupported parameter examples; moving that control to the
existing supported media-example role resolved its setup error without changing
admission. Configured lint reports zero issues.

Actual CLI comparison uses an archived `051da453a` parent and the candidate.
The parent accepts the ordinary reference and rejects its encoded equivalent
before writing a design. Both candidate imports succeed and render byte-identical
designs. All 16 generated artifacts are identical across parent ordinary,
candidate ordinary/encoded and independent repeat ordinary/encoded runs (80
actual artifact hashes). The three successful parent/candidate modules build,
vet and pass pinned Vacuum contract rules; repeat runs only regenerate. There
are no intended output differences for previously accepted ordinary input.

Exact source/input/output evidence is retained in
`/tmp/loom-891-final-manifest.json`, with comparison results in
`/tmp/loom-891-output-comparison.json`. No graph traversal changes, new formal
model, checked-in transport fixture changes or exhaustive corpus run are needed.
Final independent review and delivery remain pending.

## Review correction: complete consumer adoption

Independent review found that body-plus-query mapping and nullable field rendering
still interpreted raw reference spelling after the first parser change. The
corrected candidate also routes form-body classification, primitive/union planning,
allOf/nullable composition, URL schema admission and OpenAPI 3.2 SSE property
lookup through the same identity owner. The early external-document check now
recognizes fragment-only references before decoding; it no longer mistakes an
encoded pointer prefix for an external document. Existing reference graph
algorithms and location-specific admission rules remain unchanged.

The exact first candidate was archived and reversed before a clean fast-forward
to `43edbf9fe`. Reapplying it preserves #791's decoded-name URL cycle guard and
admission owner, replacing only that owner's parser call. Seven direct consumer
cases compare ordinary, encoded-name and encoded-prefix references byte for byte.
All seven fail against the archived parent and pass with the correction. The
reviewer's first-candidate body/nullable counterexamples remain in the evidence.
OpenAPI 3.2 property lookup also checks equivalent references and missing/malformed
controls. Focused race checks pass across all four affected packages (1.219,
7.470, 1.626 and 1.968 seconds); configured scoped lint reports zero issues.

The added CLI comparison covers mixed JSON body/query, nullable property, form
body and untagged union. Ordinary parent output is the unchanged-output control;
encoded parent input fails before emitting a design. Candidate ordinary/encoded
and independent repeat output match those accepted ordinary artifacts exactly
(320 actual artifact hashes, 12 build/vet/Vacuum modules and eight repeat modules).
The initial comparison inputs omitted operationId and therefore failed Vacuum's
existing operation-ID rule after build/vet passed. Corrected inputs include
operationId, and the initial raw results remain labeled as setup failures. This
correction changes no production admission or lint rule.
