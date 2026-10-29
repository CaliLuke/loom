# Ordered schema examples

`SchemaExamples.tla` models the projection of an already prepared authored group
onto a schema. `representative.cfg` reproduces #1002: retaining only the final
value loses the earlier value. `checked.cfg` retains the full ordered sequence,
including a null token and duplicate values. TLC 1.7.4 checks nine distinct states.

Run explicitly with the pinned TLA+ 1.7.4 tools jar and temporary output paths:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/schema-examples-old -config representative.cfg SchemaExamples.tla
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/schema-examples-new -config checked.cfg SchemaExamples.tla
```

The first command must fail `OrderedValuesPreserved`; the second must pass.
The additional `collapsed.cfg` control merges schema/media groups and fails on
the first visit; `append.cfg` appends the group again and fails on the second
visit. Run them with the same command and separate temporary metadirs. The
checked mode keeps both scopes separate and replaces output on each visit.
`Analyzer.applySchemaExample` consumes the prepared sequence; `RenderSchema`
emits plural values without the redundant representative. Singleton output
continues to use `example`, which denotes the same one-value sequence.

This finite model assumes valid, already projected values. It does not establish
Go copying, codec projection, provenance-marker extraction, filtering, component
identity or JSON/YAML encoding.
`schema_examples_test.go` checks the concrete projection; rendered and imported
contract tests cover serialization, regeneration and compilation. Proof tools
remain opt-in and are not CI dependencies.

Validation for #1002 uses base `73762ebc0` and Go 1.27.3. The bounded generated
comparison covers `OpenAPIReusableComponentsDSL` (multiple object and header
examples and reusable references), `StreamingPartialExamplesDSL` (unchanged
singletons, invalid partial objects and streaming controls), and
`MultipleSchemaExamplesDSL` (nullable order, duplicates and byte values). Each
base and changed application was generated and compiled. With identical CLI
basenames, every generated Go file matches byte for byte; only OpenAPI JSON/YAML
change for the two plural fixtures. The singleton streaming fixture is entirely
unchanged. Intended differences are replacement of representative schema
`example` values with complete `examples` arrays. Media Example Objects and
component names remain unchanged in the existing unscoped fixtures. The new
scoped fixture additionally excludes schema-only entries from media output,
retaining its separately authored green media example.

The import test extends `supported.json`, compiles the regenerated service,
checks ordered null/duplicate values, repeats generation in an independent
process, and strictly re-analyzes the schema after removing separately
unsupported document-identity/server fields. Direct tests additionally check
partial-example omission, detached snapshots, custom projection, scope-specific
representatives, and complete local annotation replacement. Review exposed
parameter-array loss, inline schema/media conflation, and repeated nullable
overlay appends; durable regressions cover all three. Imported source groups
retain their scope rather than deduplicating equal values. The OpenAPI package tests check parser admission,
JSON/YAML goldens and isolated-process byte stability for both output versions;
the focused rendered test supports the pinned Vacuum gate through
`LOOM_OPENAPI_CONTRACT=1`.
