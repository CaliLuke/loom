# Declaration metadata ownership (#1063)

Metadata belongs to the attribute that authors it. Validate a named declaration's
own metadata once even if unused; validate each occurrence's own metadata without
revalidating its referenced declaration. Do not deduplicate all errors by type:
two occurrences can author two independent invalid policies.

The bounded model enumerates zero, one or two occurrences, valid/invalid
declaration metadata and every subset of invalid local occurrences. `occurrence`
models the old named-openness walk: unused metadata is missed and repeated uses
repeat declaration diagnostics. `global` is a rejected deduplication strategy
that suppresses local errors after a declaration error. `checked` separates the
owners. Expected results: occurrence fails DeclarationOnce, global fails
LocalPreserved, checked satisfies both invariants. TLC reproduced those failures
(16 and 20 states respectively); checked passed with 28 generated/distinct states.

Run explicitly with TLA+ tools 1.7.4 (TLC 2.19), as used by the adjacent
object_openness model. Proof tools are not a build, CI or release gate:

```sh
for cfg in occurrence global checked; do
  java -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
    -metadir "/tmp/loom-metadata-$cfg" -config "$cfg.cfg" DeclarationMetadata.tla
done
```

`RootExpr.validateTypeDeclarations` and `AttributeExpr.validateMetadata` implement
the declaration pass. Attribute validation invokes the same metadata owner on
each occurrence. Result declarations retain their own explicit-view validation;
references select a view on their referenced result type. `validateObjectOpenness` no longer checks the referenced
declaration's metadata. Collection presence checks remain occurrence validation;
the declaration pass does not perform runtime codec admission or mark the runtime
validator's seen set. Tests `TestDeclarationMetadataValidatedOnce` and
`TestOccurrenceMetadataRemainsIndependent` check real DSL ownership and diagnostic
counts across types, aliases, result types and several metadata families.

The model assumes each declaration and occurrence is enumerated once. It does
not model DSL cloning, metadata normalization, Go shape analysis or validation
inside unused type fields. Go tests cover evaluation and normalization; only
the declaration's own metadata is added to the root pass.

OpenAPI's `resultTypeView` consumes selection through the existing projected-result
marker. It must not fall back to declaration view metadata after projection; an
already-projected declaration can retain that metadata. Direct IR regressions and
generation of default/non-default declaration views check termination and retained
fields. This generator recursion is outside the metadata-count model's scope.
