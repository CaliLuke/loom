# Prepared enum declaration and member provenance (#799)

`PreparedEnum.tla` compares representation choices before changing production.
It models two independently prepared consumers, optional wire renaming, the
inherited object enum `{n:1}|{n:2}`, a final minimum of 1 or 2, and defaults 1/2/3.
Mapping and wrapping are separate transitions; either consumer may mutate its
owned enum carrier afterward. All initial inputs and consumer orders are checked.
Terminal stuttering is explicit; these are bounded safety checks, not liveness.

Run sequentially from this directory with Java 25 and the shared TLA+ jar:

```sh
java -XX:+UseParallelGC -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config retained.cfg -metadir /tmp/loom-799-model/retained PreparedEnum.tla
```

The jar reports TLC 2.19; SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Substitute each configuration below; negative configurations must exit 12.

| Configuration | Property/result | Generated/distinct states |
| --- | --- | ---: |
| copy | Re-authoring inherited constants against refined fields rejects n=1 | 19/17 |
| lower | Predicate-only lowering loses enum declaration identity | 13/13 |
| first-local | Selecting a local Required carrier drops inherited membership | 13/13 |
| full-name | Comparing n against n:wire_n drops the logical enum member | 16/15 |
| borrow | Mutation of a consumer carrier must not change the source | 121/73 |
| default-at-declaration | Checking defaults only at the declaration admits n=1 under minimum 2 | 61/43 |
| retained | All six invariants pass | 492/192 |

There are twelve initial states and no state constraint. The retained variant
keeps declaration identity and logical member identity while the target chooses
wire spelling. Actual values still obey the complete enum/minimum conjunction;
default validation must not reuse the relaxed declaration-constant check.

## Correspondence and limits

- `NewMappedAttributeExpr` formerly selected the first non-nil inherited public
  `Validation`; `Remap` removed the named chain. The production constructor now
  retains the eliminated declaration edge and leaves authored local constraints
  local. The copy/lower/first-local configurations preserve the rejected designs.
  `TestPreparedEnumProvenanceConstructors` now passes all 27 constructor controls.
- `TestPreparedEnumProvenanceRepresentationCandidates` preserves the historical
  lowering-only counterexample. `TestPreparedMappingAllowsIncompleteDSLFields`
  verifies that retaining an edge does not eagerly evaluate unfinished DSL
  mappings. `TestPreparedEnumDefaultsAndOwnedCopies` checks final defaults and
  detached mutable representations through the production constructor.
- `valuePlanBuilder.object` matches logical member names only for captured
  ancestor enum clauses. `TestPreparedEnumProvenanceMappedProjection` verifies
  exact wire constants; `TestPreparedEnumLogicalMemberBindingGuards` preserves
  ambiguity rejection and unrelated ordinary-plan rejection.
- `TestPreparedEnumProvenanceGenerated` compiles real implicit, mapped and explicit
  request/response modules, accepts n=2 and rejects n=1/n=3 at both mounted
  request and client response validation boundaries. It checks declared enum
  annotations plus the final minimum in emitted schemas. `TestPreparedEnumOpenAPI`
  checks JSON/YAML artifacts with libopenapi and the pinned Vacuum contract gate.
- OpenAPI schema emission must not discard prepared enum annotations when the
  physical attribute has no local Validation. Its schema fingerprint uses the
  same checked annotation owner, while the raw structural encoder remains
  available for example seeds. `TestPreparedEnumSchemaIdentity` proves retained
  enums with the same physical shape remain distinguishable. This consumer
  correspondence is tested directly; the bounded model has no cache or hash.

The model abstracts away Go graph duplication, alias recursion, field-removal
semantics, named-type registry identity, value equality, target codecs, defaults
on missing fields and transport validation. Existing expression/value-plan tests
plus bounded generated tests must cover those boundaries. The model's borrowed
source and wrong-default variants are rejected candidates, not claims that both
faults currently exist. It does not authorize new public behavior or arbitrary
same-name mapping between unrelated attributes. Model sources/configurations
remain here; checker state and logs stay under /tmp until delivery cleanup.
