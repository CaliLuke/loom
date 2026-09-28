# Byte defaults: issue #577

Default transformation must use the physical storage type and the target's
presence policy. Native and named byte slices use a nil check; a supplied
nonnil empty slice is a value and does not receive a default. Field metadata
takes precedence over the semantic kind. Nullable defaults use the declared
wrapper's SetValue method so a named slice does not become Nullable[[]byte]
through generic inference. They apply only when the target uses defaults and
does not store primitive values as pointers, matching Optional/native policy.
Explicit null, empty and nonempty supplied values remain unchanged.

## Reproduce the direct and generated checks

Run from the repository root:

    LOOM_DIR="$PWD" go test ./codegen ./http/codegen ./jsonrpc/codegen \
      -run '^(TestBytesDefault|TestBytesValidationGenerated|TestNamedCollectionDefaultTransform|TestNullableNamedCollectionDefaults|TestTransformAnyDefault)' -count=1

The direct tests compile independently emitted source/target field declarations
across 448 layout cases and 64 presence/context cases. They cover native Bytes,
named Bytes, alias chains, raw JSON, comparable String/custom metadata, required
fields, source/target defaults, pointer contexts, Optional and Nullable values.
Generated HTTP and JSON-RPC checks compile, vet and exercise the actual runtime;
form requests distinguish omitted fields from supplied empty bytes. Required
source-pointer cases obey the source validator's nonnil precondition.

## Findings and limits

Before the repair, the compiled cases reproduce:

- Equality against a zero slice, which Go rejects.
- Nullable[[]byte] assigned to Nullable[Blob] or Nullable[BlobAlias].
- Missing native/named byte defaults in form requests when source defaults are
  disabled, despite the target requiring default application.
- Nullable defaults applied in contexts whose target disables defaults or
  stores primitive values as pointers.

The shared owners are go_transform.go and go_transform_presence.go. The repair
changes default guards/assignment and one regenerated transform golden. It does
not establish a universal proof about generated Go. Compiler, runtime and
cross-revision artifact checks remain separate from the semantic Lean model.

Focused checks, full `make lint`, full `make test` and
`make generated-code-quality` passed. The revision comparison passed all 12
probes, each generated twice in independent processes at the parent and candidate
(48 records). The four byte-default probes fail compilation at parent
`8714f0d2008e565c1e41c71a2b14388cd359a3c1` and compile/vet successfully with the
repair. All other probes pass at both revisions.

The [comparison manifest](bytes_defaults_manifest.json) pins all 18 intended
artifact changes: three generated files each for HTTP and JSON-RPC Any defaults,
named HTTP/RPC bytes, unpreserved HTTP bytes, and form defaults. Each change is a
nil-source default guard or a declared-wrapper default assignment. All remaining
artifacts, including schemas, protobuf output, collection-default controls,
no-default controls and both checked-in SSE fixtures, are byte-identical.
Repeated generation produces identical bytes within each revision.

Reproduce the comparison with the repository's installed Go/protobuf tools:

    LOOM_VALUE_BASE=8714f0d2008e565c1e41c71a2b14388cd359a3c1 \
    LOOM_VALUE_CANDIDATE="$PWD" \
    LOOM_VALUE_MANIFEST="$PWD/codegen/bytes_defaults_manifest.json" \
    LOOM_VALUE_RESULTS=/tmp/loom577-verification \
      go test ./internal/valuecontract -run '^TestCompareRevisions$' -count=1 -timeout=90m

The results directory must not already exist. The reviewed candidate snapshot
was `3e23e095b088b053012d8038eedf5dfce280117ec48cb75e63cee6058b0684d9`;
concurrent proof-only work does not change the frozen generator/test inputs.

## Architectural follow-up

A separate pinned-parent reduction shows named nullable fields losing their
occurrence's nullability in response views when the same named type is also used
by a non-null sibling. It reproduces with or without a default. That is an
occurrence/shape propagation acceptance case for #570's shared representation
plan, not an additional prerequisite patch. This ticket does not change view
shape ownership. See the [live plan](../roadmap/value-contract-plan.md).

Keep source regressions, the final comparison manifest and concise evidence.
After final validation/review, remove task-owned temporary outputs under the
AGENTS.md process, preserving the inputs still needed by #570.
