# Required contracts at schema boundaries

Issue #1069 reproduced on f6c17e1fb: multipart plan snapshots published the
local `Required(id)` of a composed payload after removing its path member;
a detached nested Row snapshot published `extra` without inherited `id`.
The existing expr requiredness query retains inherited requirements and typed
member omissions. Selecting its physical object members and retaining typed
additional keys produces the correct schema requirements. OpenAPI rendering and fingerprinting must both
consume that owner. Named reference occurrences still emit their local
requirements: their referenced declaration owns inherited assertions.

`RequiredProjection.tla` separates local requirements, constraint parents,
physical members and reference-local overlays. It includes ordinary empty
requirements, typed additional required keys and typed path-member removal. It assumes expr has already
admitted the design and resolved member identity; it does not prove arbitrary
recursive graphs, codecs, name mapping or Go implementation correspondence.

Run explicitly with TLA+ 1.7.4, one worker and a fresh temporary state directory:

```sh
java -Xmx256m -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -config effective.cfg -metadir /tmp/<fresh> RequiredProjection.tla
```

`local.cfg` reproduces the upload violation (exit 12, 2 states).
`all.cfg` rejects unfiltered inheritance (exit 12, 2 states).
`effective.cfg` passes all seven cases (exit 0, 8 states).
The counterexamples informed `schemaRequiredNames`, used by
`applySchemaValidation` and `canonicalSchemaEncoder.writeValidation`.

`TestSchemaRequiredNamesUsesEffectiveObjectContract` checks detached composition,
captured multipart roots, identity agreement and local overlays.
`TestInheritedRequiredProjection` checks rendered 3.1/3.2 JSON and YAML using
libopenapi, valid/invalid object values and both nested row references; enable
`LOOM_OPENAPI_CONTRACT=1` for the pinned Vacuum lint. The maintained
`InheritedRequiredProjectionDSL` also participates in generated compilation.
These implementation tests cover the correspondence the finite model assumes.
Proof tools remain explicit design aids, outside CI and normal test gates.

The first implementation recaptured full effective value graphs per hash node;
existing allocation-scaling and typed-projection regressions rejected it. The
final query uses `AllRequired` plus immediate membership, without descendant
value capture. `TestCheckedFingerprintSnapshotAllocationScaling`,
`TestNestedEnumFingerprintScaling`, and `TestAdditionalAttributesTypedPathBoundMembers`
cover those boundaries.

Independent review exposed inherited requiredness after usage pruning.
`usage-lost.cfg` rejects restoring a removed readOnly/writeOnly field on a typed
object while retaining the distinct literal key `id:literal`. Pruning records
declared keys separately from forbidden wire names; requiredness consumes the
former and schema properties consume the latter. `TestInheritedRequiredUsageProjection`
covers both directions, JSON-renamed fields, valid/invalid wire values, and the
retained literal additional requirement. Wire-name normalization itself remains
outside the model.
