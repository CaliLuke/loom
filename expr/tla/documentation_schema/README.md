# Documentation schema admission (#1034)

A documentation-only body describes JSON Schema, not a generated codec.
The model considers one valid schema whose untagged shape has no supported
runtime codec, visited once in each context and in either order.

`Old.cfg` models the original shared admission policy and rejects the valid
schema. `Shared.cfg` relaxes schema admission but shares a visited cache, letting
a schema-first visit accidentally admit the runtime codec. `Checked.cfg` keeps
context-local visits and admits the schema while rejecting the codec.

Run explicitly with the reusable TLA+ 1.7.4 jar (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`,
and Java 25, from this directory:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -config Checked.cfg -metadir /tmp/loom-documentation-schema DocumentationSchema.tla
```

Repeat with separate temporary directories for `Old.cfg` and `Shared.cfg`.
The old policy fails `SchemaAccepted`; shared-cache admission fails
`RuntimeRejected`. The corrected model checks 5 generated / 4 distinct states.
These commands are design tools and are never build, test, hook, or CI gates.

`attributeValidator` in `expr/attribute_validate.go` owns recursive admission;
`validateSchema` allocates its own visit map. `TestSchemaValidation*` checks both
visit orders through object, array and map nesting and retains required-field
and branch-identity rejection. `TestDocumentationUnionSchemas` checks real
request/response DSL evaluation and isolation from a shared typed result.
`resolveBodyAttribute` copies documentation type graphs before modifiers run;
response finalization preserves those schema graphs without runtime wrappers.
The raw request specimen and its rendered/compiled tests check scalar/object,
array/object, nested tagged unions and shared named types.

The model assumes schema validity and runtime unsupportedness; it does not
prove the Go traversal, copying, examples, naming, rendering, or generated code.
Those obligations belong to the implementation tests. Overlapping `oneOf`
branches retain JSON Schema's exactly-one meaning; documentation does not add a
runtime decoder or promise that every possible value matches a branch.
