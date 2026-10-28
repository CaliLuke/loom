# Reference sibling conjunction and declaration storage

`analyze_reference_siblings.go` retains a local reference and normalizes admitted
sibling assertions against its carrier type. The occurrence is the conjunction
of the referenced contract and its siblings. It never replaces or mutates the
base. Object-property intersections use Extend for composed field storage and
retain contributors through `expr.AttributeExpr.CompositionBases` and
`ir.analyzeCompositionReferences`. Unsupported type/key-set intersections remain
explicit admission errors.

`expr.NewObjectFields` captures both semantic requiredness and declaration
requiredness. `codegen.AttributeContext.FieldPresence` and `expr.ServiceFieldLayout`
use the latter for physical storage. Requiring an optional field at a named
occurrence cannot turn an existing Optional field into a native scalar/pointer.
Generated validation must test its presence.

Run TLC 1.7.4 explicitly (Java 25 was used):

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-reference-siblings -config conjoin.cfg ReferenceSiblings.tla
```

The `conjoin` configuration checks 700 states over four representative values
and Boolean declaration/occurrence requiredness. The other configurations are
intentional counterexamples: `reject` violates SupportedInputAdmitted (the old
blanket rejection); `replace` violates ConjunctionPreserved; `mutate` violates
BaseUnchanged; `flatten` violates ReferencePreserved by dropping the named
contributor from the emitted graph; `nullable` violates ConjunctionPreserved by restoring null
excluded from the target domain; `storage` violates DeclaredStoragePreserved when an optional
physical field becomes required at an occurrence. Use a separate temporary
`-metadir` for each configuration. These are design checks, never CI gates.

The domain assumes nonempty intersections and abstracts away schema syntax,
recursive references, nullable carriers, forced transport pointers and custom
codecs. It does not prove arbitrary JSON Schema composition or the importer.
`reference_siblings_test.go` checks actual import, generation, compilation,
parsed contracts and independent-process bytes, including rejected intersections.
`codegen.TestReferenceRequirementPreservesDeclaredJSONStorage` reproduced the
incorrect carrier and checks both pointer contexts; the generated required-object
case reproduced a Go compile failure before the storage correction.
The nullable-enum regression reproduced acceptance of a forbidden null before
normalizing target membership. Object cases check inline and named fields,
recursive contributor identity, generated validation and schema re-import;
document-level server/dialect import limits are excluded explicitly from that
schema round trip.

The `source` configuration reproduces the numeric-token ownership defect:
reading the occurrence's enum instead of the inherited node's enum changes
`{null, valid}` into `{valid}`. `SourceOwnershipPreserved` checks this explicit
owner distinction; `conjoin` passes it. This abstracts token identity rather than
parsing. `numeric_source_identity.go` captures original tokens by source position
before semantic construction. `reference_source_ownership_test.go` checks inherited
nullable membership, inherited array item enums and bounds, JSON/YAML precision
and quoted kinds, and inline YAML merges. Transformed references inside general
allOf branches are rejected rather than losing their assertions.
