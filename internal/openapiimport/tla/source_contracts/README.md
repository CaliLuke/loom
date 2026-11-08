# Source dialect admission before transplant

`SourceDialect.tla` models the document context lost when an external target is
copied into an entry document. Five target classes cover direct and nested
schemas plus schema-free examples and security schemes. Every source-dialect
and selected-closure combination is enumerated.

- `discard.cfg` reproduces the bug: a selected schema from a custom-dialect
  document is transplanted without its blocker. `ContractRetained` fails.
- `document.cfg` rejects a broader alternative: blanket document rejection
  blocks an example whose literal data happens to contain schema-like keys.
  `ContractRetained` fails.
- `target.cfg` preserves the dialect and applies it only to schema-bearing
  targets. All 60 generated / 40 distinct states pass, at depth 2.

The document owner in `resolve.go` retains the dialect; the grammar walker in
`resolve_grammar.go` identifies schema positions, including boolean schemas;
`admitTarget` in `resolve_tree.go` rejects unsupported schema-bearing targets
before bundling. Existing occurrence blockers and selection closure then retain
only the relevant diagnostic. `TestResolverSourceDialect` covers each target
kind, and `TestExternalDialectSelection` checks strict, selected and unrelated
operation imports through the public importer and renderer.

The model assumes correct grammar classification and an already-correct
selection closure. It does not prove arbitrary YAML parsing or graph traversal;
those remain Go test obligations. `TestDescriptionDuplicateKeys` also checks
source syntax admission before map canonicalization, with legal YAML merge
controls. Numeric-token preservation remains covered by the maintained bundle
and numeric-source tests.

Run explicitly from the repository root with TLA+ tools 1.7.4 (TLC 2.19,
revision 5a47802; JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-source-dialect-target \
  -config internal/openapiimport/tla/source_contracts/target.cfg \
  internal/openapiimport/tla/source_contracts/SourceDialect.tla
```

Substitute `discard.cfg` and `document.cfg` to reproduce their expected
invariant failures (exit 12). Models are opt-in design tools, never test, build,
commit, CI or release gates. Remove temporary checker output after independent
review. These results were obtained with Go 1.27.3 regression inputs and the
pinned TLA+ toolchain; scratch logs are in
`/tmp/loom-fresh-audit/reference-source-fix`.

## Schema target roles

`SchemaRoles.tla` explores the shared typed-target policy required by ordinary
references and URI discriminator mappings. Its finite grammar graph contains
component, nested alternative/property, boolean and inline-response schemas,
plus operation/container/literal-data controls. Traversal records grammar roles;
it does not classify a target by its mapping shape or component-root prefix.
`role-component.cfg` reproduces the old valid-inline rejection (351 distinct
states); `role-mapping.cfg` rejects the alternative that admits arbitrary mapping
nodes (354 distinct states). Both violate `ExactAdmission` and exit 12.
`role-grammar.cfg` passes: 684 generated, 360 distinct states, depth 10, exit 0.
Run the command above with `SchemaRoles.tla` and one of these configs.

This explicitly reconciles the former component-root-only #957 table with #1005's
accepted URI mapping to inline schemas. Mapping-only bypasses and arbitrary
object admission were rejected. The model assumes the finite grammar labels; it
does not establish every OpenAPI grammar position or source-parser behavior.
`descriptionSchemaPositions` derives and caches positions through the existing
`descriptionGrammarWalker`; `admitOpenAPITarget` consumes that index for schemas.
Other object kinds retain their existing restrictions. Version, source loading,
custom-dialect and resource-boundary checks still apply after role admission.
`TestResolverSchemaTargetRoles` covers 48 local/external and ordinary/mapping
combinations, including escaped property names and retained bundled provenance.
Property dictionaries, required arrays, operations and example/default data stay
rejected. Existing dialect/resource/no-network tests remain controls.
