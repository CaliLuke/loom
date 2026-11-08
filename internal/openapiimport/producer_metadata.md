# Producer metadata import

OpenAPI 3.1 and 3.2 documents may explicitly name their normative default dialect,
`https://spec.openapis.org/oas/3.1/dialect/base`. Entry analysis and referenced
schema-bearing target admission use the same decision. Other dialects remain
unsupported; this does not admit custom schema resources or vocabularies.

Header examples use the existing wire-example normalization and `media` scope.
The header attribute retains its schema-scoped samples separately. The source
Header mapping owns presence, including explicit null and YAML merge selection;
a nested schema example is not an authored Header example. The dependency's
current recursive lookup behavior is recorded at [libopenapi #695](https://github.com/pb33f/libopenapi/issues/695). No inferred
value equality or second annotation store is used. Existing typed compatibility,
reusable Example metadata and unsupported-fact diagnostics remain authoritative.

The detached schemaWithExamples copy combines only normalized scoped entries at
the header occurrence. It leaves a referenced declaration's own examples intact.
Compiled design checks cover scoped sample ownership; actual exporter first and
second rounds remain separate acceptance obligations. Plain servers and inline
primitive composition are not enabled by this metadata increment.

An explicit null example remains present in the normalized scoped samples. A
nullable simple header still fails the existing parameter codec admission
(`nullable values require a JSON codec`); this change does not enable a new
header codec or claim that such a null can be sent on the wire.
