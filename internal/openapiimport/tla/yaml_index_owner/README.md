# Effective YAML index traversal

This bounded model supports #922's dependency-index correction. Two authored
paths refer to one canonical YAML source node. Each occurrence contributes a
`Bound` leaf and contains a cyclic alias edge. `AnchorKind` selects mapping or
merge-sequence ancestry; both need the same active-path ownership rule.

Source identity is the canonical node before `ResolveMergeNodes`, never the
pointer to a newly allocated shallow merge view. `PathSeen` passes ancestry to
children and releases it when that path ends. `GlobalSeen` incorrectly treats
an earlier sibling visit as a cycle. `Raw` represents index traversal ignoring
alias/effective mapping content. `MapOnly` represents merge expansion guarding
mappings while recursively following sequences without recording their identity.

The model checks occurrence coverage, exclusion of cyclic re-entry and eventual
completion of this finite graph. It assumes immutable source reads and correct
merge expansion; those are implementation obligations, not properties proved by
this model. It does not model OpenAPI reference resolution, source locations,
JSON Pointer grammar, concurrent indexing, or semantic validity of cyclic YAML.
Valid recursive `$ref` graphs are distinct from YAML graph traversal cycles.

## Results and design decision

TLC 1.7.4 with Java 25, one worker:

| Configuration | Expected result | Generated / distinct states |
| --- | --- | --- |
| raw | OccurrenceCoverage fails (exit 12) | 3 / 3 |
| global | OccurrenceCoverage fails (exit 12) | 5 / 5 |
| unguarded | NoCycleReentry fails (exit 12) | 5 / 5 |
| sequence-current | NoCycleReentry fails (exit 12) | 5 / 5 |
| owned | Invariants and completion pass (exit 0) | 7 / 7 |
| sequence-owned | Invariants and completion pass (exit 0) | 7 / 7 |

Use path-local ancestry over canonical nodes for all traversed container kinds.
A global visited set loses valid occurrences, and guarding only mappings leaves
self-referential merge sequences unbounded. An early draft omitted parentheses
around a primed Boolean assignment and failed TLC state construction; that run
was corrected and is not evidence.

From this directory, point `TLA_JAR` at the installed pinned jar (SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
model_tmp=$(mktemp -d)
java -Djava.io.tmpdir="$model_tmp" -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$model_tmp/states" -config owned.cfg IndexTraversal.tla
```

Repeat each configuration with its own temporary directory. Preserve summarized
results, then remove checker state and raw output after independent review.

## Implementation correspondence and acceptance

- `utils.ResolveMergeNodes` owns immutable effective mappings; expansion ancestry
  must include sequence nodes as well as mappings before following aliases.
- Index fragment navigation and component collection enumeration must read the
  same effective view. Do not repair the same semantics at each component kind.
- Ref/schema-ID walking resolves an occurrence before deriving its scope/keys,
  retains the existing example-payload exclusions, and uses canonical ancestry.
- Go tests cover mapping/sequence precedence, direct aliases, repeated anchor
  paths, empty and cyclic input, unchanged source nodes and metadata, missing
  refs, encoded pointers and actual public model/index races.
- Loom import tests compare merged and explicit source through DSL/OpenAPI output
  and compilation; the model does not establish those concrete correspondences.

## Concrete correspondence

The dependency regression tests are `TestBuildV3ModelEffectiveComponentIndex`,
`TestEffectiveReferenceOccurrences`, `TestEffectiveFragmentTraversal`,
`TestEffectiveTraversalCycles`, `TestEffectiveComponentCollections`,
`TestEffectiveInlineSchemaCollections`, `TestEffectiveSchemaIDScope`, and
`TestResolvedMergeNodeSequenceCycleTerminates`. Existing merge metadata tests
retain original key/value pointers and serialized authored source. Existing
index tests retain missing-reference rejection and valid recursive references.

Run the bounded affected dependency packages from its checkout:

```sh
GOEXPERIMENT=jsonv2 go test -race ./index ./utils ./datamodel/low/v3 . -count=1
```

All four packages pass (about three minutes for the index suite). No exhaustive
framework design corpus is part of this gate. `TestImportedYAMLComponentIndex`
checks the five container shapes at the Loom importer boundary; the sixth
schema-value merge is a dependency-only control because Loom's independent raw
schema-admission defect is tracked in #924. The model establishes traversal
ownership, not admission of every schema feature.

The importer/generator check uses identical extension values in explicit and
merged fixtures, preserving extension metadata in the comparison. With the
separate #925 deterministic extension serializer, all 17 design/generated files
match exactly across explicit, merged and independent-process repeated imports.
Generated code compiles; generated JSON and YAML pass libopenapi and pinned
Vacuum contract validation. This check does not normalize emitted bytes or omit
extension fields. The accepted dependency source diff was independently reviewed
against 819f09426ceaa19ea8b04c6901e70e3c9b29344b; the root module pin records the
published source used by consumers.

The published dependency correction is commit
`7886f0170cc4ff64f4be95a18a2576767989aaa2`, module version
`v0.41.7-0.20261014145029-7886f0170cc4`. An independent consumer fetched it as
an ordinary dependency without replacements and passed the six public model
shape tests under race detection. The shared owner remains in that dependency;
Loom adds the importer regression and dependency pin, with no source flattening.

## Upstream release recheck

On 2026-10-27, upstream v0.41.8 still rejected the four aliased/merged component
container shapes in `TestImportedYAMLComponentIndex`; the explicit control passed.
The standalone public-API reproduction is
[libopenapi #672](https://github.com/pb33f/libopenapi/issues/672).
[PR #673](https://github.com/pb33f/libopenapi/pull/673) ports the correction to v0.41.8.
The immutable model-building repair is released, but the effective index repair
remains necessary. The original publication above records the earlier source;
the root `go.mod` owns the current exact dependency pin.

All six model configurations were rerun with TLC 1.7.4 and Java 25.0.5.
The raw/global controls still fail occurrence coverage, unguarded/mapping-only
sequence controls still fail cycle exclusion, and both path-owned controls pass
with seven distinct states. The model and its assumptions are unchanged.

The v0.41.8-based correction was published as
`github.com/CaliLuke/libopenapi v0.41.7-0.20261027210921-2be7e43876e4`.
The pseudo-version prefix follows the fork's existing tag ancestry; `FORK.md`
records the v0.41.8 source and upstream correction. Focused importer alias/merge,
exact-number, encoded-reference and rejected-input checks pass, including the
YAML checks under race detection. Ordinary and nullable imported designs compile;
all 34 captured design/generated artifacts match the previous pin byte for byte.
The nullable contract also passes independent-process regeneration. OpenAPI v3
package tests and the released Vacuum/libopenapi contract suite pass.
