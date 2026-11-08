# Blocked occurrences under selection (#957)

`BlockedSelection.tla` is design evidence for the failure representation in
[`roadmap/issue-957-external-references-design.md`](../../../../roadmap/issue-957-external-references-design.md).
Each unresolved reference occurrence becomes a blocked component, with a blocker
diagnostic keyed by its bundled path. The model computes:

- the selected closure, as a deterministic worklist
- diagnostic retention
- the render decision
- per-operation closures for `--skip-unrenderable`

It compares them with a declarative least fixed point over every component
kind.

It also covers an unresolved Path Item, whose operations and tags are
unknowable. A hidden input records what the missing document would contain.
That hidden input is used only by the oracle.

## Decisions informed

- **Blockers are required.** A blocked placeholder without a recorded blocker
  renders. See `placeholder` (M2a).
- **Retain by bundled path.** Blockers are retained by bundled path, not by
  origin. See `originFilter` (M2b).
- **The closure needs examples, and the current filter over-retains.**
  Selection and partial closures must traverse example components (and,
  in the design, security schemes):
  - `fiveKinds` drops a reachable example blocker, so partial import writes
    an operation that reaches it.
  - `unfilteredExamples` is the pre-change behavior. It never filters example
    diagnostics, which over-retains a blocker that no selected operation reaches.
- **Unknown Path Items under tag selectors.** An unresolved Path Item is kept
  whenever a tag selector is active, because its tags cannot be known. A path
  selector decides by path key, precisely. See `assumeUnselected`.

## Commands and results

The toolchain is the same as in `../resolution_identity/README.md`: Java 25.0.5
(Corretto) and `tla2tools` 1.7.4. Run from this directory:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 4 -metadir "$(mktemp -d)" -config checked.cfg BlockedSelection.tla
```

Observed results (2026-11-05):

| Configuration | Expected | Observed |
|---|---|---|
| `checked.cfg` | All invariants hold, plus `Termination` | Pass: 288,192 generated, 263,616 distinct, depth 18, exit 0 |
| `placeholder.cfg` | `NoBlockedRendered` is violated | Fails (exit 12): strict mode renders `U1`/`U2` with no blocker |
| `origin-filter.cfg` | `BlockerKeptIffImported` is violated | Fails (exit 12): selection `tag:t1`, `o1 → E → U2`, and the reachable blocker is dropped |
| `five-kinds.cfg` | `PartialSound` is violated | Fails (exit 12): `o2 → E → U2`, and `o2` is written |
| `assume-unselected.cfg` | `PathItemSound` is violated | Fails (exit 12): selection `tag:t1`, and the hidden Path Item's operations carry `t1` |
| `unfiltered-examples.cfg` | `BlockerKeptIffImported` is violated | Fails (exit 12): selection `prefix:/c` selects no operation, yet the `U2` blocker is retained |

## Assumptions and limits

**Bounds**

- Schemas `A` and `B`, example `E`, and blocked components `U1` (a schema) and
  `U2` (an example).
- Restricted edge shapes, including the cycle `A`/`B`.
- Two operations, one security-key failure, and one Path Item key.
- Four selectors.

**Abstractions**

- Security-scheme components follow the same rule as examples and are not
  separately enumerated.
- `--list-tags` incompleteness is a Go obligation, not a modeled invariant.

**Out of scope**

The model does not prove:

- the existing `filterSelectionDiagnostics` / `filterOperationDiagnostics`
  string handling
- libopenapi parsing
- planner refusal of blocked keys

Implementation tests must cover each control as a concrete document. They must
cover strict, selected, partial, report and list-tags modes, and they must
assert the exact bundled `Path` and `Origin` of every retained blocker. The
pre-change seams were `pruneComponents` and `filterSelectionDiagnostics`
(`selection.go:211-437`) and `AnalyzePartial` (`partial.go:42-176`).

The model is an explicit design tool and is never a CI gate.

## Implementation correspondence

`resolve_bundle.go` records occurrence blockers; `selection_references.go`
computes the grammar-reference closure that supplements normalized components.
`resolve_provenance.go` retains selected blockers for planner admission, and
`partial.go` records unknown Path Items as skipped. `TestBlockedOccurrenceSelection`,
`TestBlockedSecurityKey` and `TestBlockedClosureKinds` cover these seams.
CLI external-reference tests cover report and lower-bound tag output.

The bounded assumptions and recorded checker results above remain unchanged.

## Authored operation addresses and effective security

`OperationSelection.tla` extends the selection boundary with one standard and
one Additional Operation location, two source-key spellings, inherited/local/
explicitly public security, and supplied/failed root and local targets. The old
method-derived address loses the resolver's exact key; the old raw-root filter
blocks a public override. These are separate current-policy counterexamples:
`operation-source-legacy.cfg` violates `NoLostSecurity` (129 distinct states),
`operation-root-legacy.cfg` violates `NoFalseBlock` (98 distinct states).
`operation-checked.cfg` checks both invariants over 144 distinct states (192
generated, depth 3). Legacy controls exit 12; checked exits 0 with the toolchain
above. Initial development runs with a missing Sequences import are syntax
failures, not counterexamples; only the corrected runs establish these results.

Run explicitly by substituting `OperationSelection.tla` and the chosen
`operation-*.cfg` in the TLC command above. This companion model abstracts URI
parsing and component reachability, already covered by the other models. It
checks one candidate at a time; tests cover actual multi-operation selection.

`Operation.sourcePath` is captured by `selectPathOperations` at discovery,
including `additionalOperations` and the exact authored key. Analysis, security
projection, dependency closure, promotion, naming, planner diagnostics and partial
filters consume that fact; the wire method cannot reconstruct it. Source-less
manually constructed Documents retain synthetic planner IDs as non-source
context. `filterSecurityFailures` applies the candidate's effective security to
both selected analysis and partial imports. `TestOperationSourceSecurity`,
`TestOperationSourceOverrides`, `TestPartialSecurityProjection` and
`TestAdditionalOperationIdentityDiagnostics` establish concrete correspondence.
`TestOperationSourceGenerated` compiles the emitted GET/PURGE/public design and
asserts actual regenerated security, scheme and inline-target schema contracts.

## Source-retained components keep their normalized body

`SourceRetention.tla` separates retained identity from the normalized body walk.
The pruner has two reference inventories:

- The normalized walk follows `Ref` edges, including edges that exist only after
  analysis, such as declarations promoted by `prepareSchemaObjects`.
- The source walk (`selection_references.go`) follows raw grammar references,
  including references that normalization drops. For example, a closed oneOf
  discards a redundant sibling `properties` entry whose `$ref` remains in
  `bundle.references`.

Before this correction, `closure` membership also guarded the normalized body
walk. A component that the source walk retained first therefore never had its
body walked. The source walk introduced this second writer.

Public input reproduces the defect (`TestSelectionSourceRetainedBodyVisited`).
Selecting the union operation retains Y raw-only, without `YAdditionalProperty`.
Strict import then fails with an unresolved schema reference, and partial import
skips a valid operation.

`TestComponentPrunerSourceRetainedKinds` is a synthetic seam. It retains the same
name in schemas, parameters, requestBodies, responses and headers through
separate raw prefixes, and requires each kind's own body to be walked.

The model has three components, an X/Y cycle and independently chosen normalized
and raw edges. Two operations run in either order. Within each walk, the
worklist is processed in a nondeterministic order. The oracle retains raw
closure from the operation raw roots, and every retained name contributes its
normalized body.

| Configuration | Observed with TLC 1.7.4 / Java 25.0.5 |
|---|---|
| `source-retention-name-guard` (previous pruner) | exit 12, `BodyComplete`: X retained raw-only, its normalized edge to Y never walked (210 093 distinct states) |
| `source-retention-guard-only` (body guard, but source retention still skips the body) | exit 12, `BodyComplete`, same trace |
| `source-retention-checked` (`body-once`) | pass: `BodyComplete`, `ExactClosure`, `RetainedBodiesWalked` and `Termination`; 1 016 788 distinct states, depth 25 |

These results were observed on the model before its explanatory comment was
reworded to name promotion only. The module's TLA+ definitions are unchanged, so
the results are reused under that comment-only equivalence. The file's bytes and
hash differ from the run input.

A pinned cross-operation trace adds a second case: o1 retains X raw-only, then o2
references X normally. That trace fails under `name-guard` and passes under
`guard-only`. Both halves of the correction are therefore needed.

Run with the command above, substituting `SourceRetention.tla` and a
`source-retention-*.cfg`.

`componentPruner.retainComponent` is the single owner. It records retained
identity in `closure` and walks each kind's normalized body once, guarded by
`bodies` and keyed by kind and name. Normalized `Ref` visits and
`visitSourcePrefix` both call it by name; source tokens are not re-encoded as
references. Example and security-scheme components have no body. The model
abstracts component kinds and the pruner's other edges. It does not establish
source-grammar completeness.

The inverse direction is resolved for #957, but it is an obligation on
consumers. Could a component retained only through a normalized edge have raw
grammar dependencies, and so blockers, that are never source-walked? Within
#957, no. Every normalized edge the analyzer builds has a raw witness inside a
prefix the source walk reaches:

- **Authored `$ref` occurrences** are recorded in `bundle.references`.
  `recordOccurrence` excludes only discriminator mappings and Path Item
  references. #957 rejects discriminators, and copies Path Item fields under
  operation prefixes.
- **Inherited-property references** take their raw witness from the base, which
  the child's raw `allOf` reference reaches.
- **Promoted declarations** live under their owner's raw prefix.

By induction from the operation prefixes, every retained authored component is
raw-reached, so its raw dependencies are walked. This rests on an independent
enumeration of the analyzer's schema `Ref` construction sites; it does not prove
the analyzer complete.

A consumer that adds normalized-only edges to authored components breaks that
induction. #1005's discriminator mappings, `defaultMapping` and weak ParentHint
children are such edges. That consumer must make the retention owner retain
those components' raw grammar dependencies too, and must not drop them or walk
them per edge.
