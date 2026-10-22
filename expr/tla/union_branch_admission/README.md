# Resolved union branch identity admission (#701)

This model checks the boundary between authored declarations, resolved branch
identities, design admission and generator capture. Each branch records its union
owner, authored name, a preliminary tag, and its effective tag. Eight small designs
cover repeated names, mapped suffixes, name duplicates with distinct tags,
conflicting inherited/explicit tags, occurrence overrides, sibling reuse, case
preservation and an ordinary distinct pair. Types and branch shapes are assumed
valid; Go tests cover actual DSL resolution and untagged shape admission.

The current strategy admits every list, leaving capture or generated compilation
to detect ambiguity. Name-only and tag-only admission each miss a real boundary.
Early admission uses preliminary tags before inherited metadata resolution and
misses the same effective-tag collision. Global uniqueness rejects valid reuse.
The chosen owner validates both resolved identities inside each individual union
before generators consume the graph. It does not rename or deduplicate branches.

## Run

Use the existing pinned TLA+ 1.7.4 jar (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
with Corretto 25. From this directory:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-union-admission-owned \
  -config owned.cfg UnionAdmission.tla
```

Repeat with each other configuration and a separate temporary metadir.

| Strategy | Exit | Generated / distinct | Result |
| --- | --- | --- | --- |
| current | 12 | 17 / 17 | Accepted duplicate reaches publication |
| early | 12 | 23 / 23 | Preliminary identity misses resolved tag conflict |
| names | 12 | 23 / 23 | Effective tags can conflict despite distinct names |
| tags | 12 | 20 / 20 | Logical names can conflict despite distinct tags |
| global | 12 | 19 / 19 | Independent owners cannot reuse legal identities |
| owned | 0 | 32 / 32 | All bounded designs satisfy the three invariants |

## Correspondence and limits

`AttributeName` supplies logical names; `UnionVariantTag` owns occurrence metadata,
named-type metadata and fallback-name precedence. `eval.RunDSL` runs DSL bodies
and preparation before expression validation, then finalization. `attributeValidator.validateChildTypes` calls `validateUnionBranchIdentities` for
tagged and untagged unions. Preparation resolves branch types before this check
evaluates inherited tags.
`valueOccurrenceBuilder.union` keeps its independent effective-tag integrity
check. `codegen/service.newValueData` currently panics when that check fails.
Distinct tags with duplicate logical names instead reach duplicated generated Go
members. The fix must reject both at expression admission, not recover panics or
rename public branches downstream.

`TestUnionBranchAdmission*` checks actual declarations, forward references,
metadata precedence, mapped suffixes, collection nesting, both union modes and
independent scopes. `TestGenerateRejectsDuplicateUnionBranches` checks stage and
source location, no panic and no generated directory. Bounded generation and
compilation of valid designs remain implementation acceptance, not a property of
this model. Existing cycle and promoted-name tests protect graph traversal and
allocation behavior. The model is finite and assumes each union has already been
reached by the evaluator; it does not prove arbitrary recursive graph traversal,
metadata resolution, Go naming, transport codecs or concurrency.
