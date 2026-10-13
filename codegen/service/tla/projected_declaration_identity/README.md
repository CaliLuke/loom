# Projected declaration identity and nullable occurrence storage

Issue #829 conflates projected declaration names with occurrence type expressions.
`GoTypeName` correctly returns `loom.Nullable[T]` for nullable storage, but that
expression is not a legal declaration or helper identifier. A first encountered
nullable occurrence also affects the projected named validator's parameter,
although other occurrences call the same intrinsic declaration.

The bounded model separates two allocated names (`ChildView` and `ChildView2`),
one ordinary and one intrinsically nullable type, and plain/nullable occurrences
of each. Every collection order is explored. The allocator's collision-free
identity and the evaluated declaration graph are assumptions, not proved here.
Type shape, packages, recursive projection and actual Go syntax remain runtime
and direct-test obligations.

- `old` derives name and parameter from the first occurrence. It violates
  DeclarationIdentity (3 generated / 3 distinct states).
- `names-only` repairs the identifier but retains the first occurrence's
  parameter. It violates ParameterAgreement (3 / 3).
- `erase-presence` globally clears presence to obtain concrete names. It violates
  OccurrencePreservation (3 / 3); intrinsic parameter preservation also matters.
- `checked` derives names from the allocated concrete identity and parameters
  from intrinsic nullability while keeping occurrence storage. All three
  invariants and termination pass (34 / 16).

Use Corretto Java 25.0.4.1 and TLC 1.7.4 (JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`):

```sh
mkdir -p /tmp/loom-829-java
java -Djava.io.tmpdir=/tmp/loom-829-java -XX:+UseParallelGC \
  -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC -workers 1 \
  -config checked.cfg -metadir /tmp/loom-829-checked ProjectedDeclarationIdentity.tla
```

Run each negative configuration separately with distinct metadata paths. Each
must exit 12 on its documented invariant; checked must exit 0. Syntax errors do
not count as negative evidence. Keep checker state and raw logs outside the repo.

Correspondence: `collectProjectedTypes` deduplicates projected named declarations;
`buildProjectedType` and `collectViewUnionTypes` own their emitted identities.
The ordinary service `collectUnionTypes` uses the same allocated concrete named
identity: nullable references cannot become declaration identifiers in either
package. `TestNullableUnionDeclarationIdentity` checks both collectors in both
occurrence orders; this adds implementation correspondence without extending
the model state space or its allocation assumptions.
`GoValueTypeName` already provides allocated concrete names. Shared
`NamedValidationFunction` owns validator identity and physical parameters when
called with the intrinsic named declaration, preserving naming metadata. Local
Nullable flags remain on field/branch occurrences and must not be cleared from
the authored or projected graph. Definitions and calls must use that same owner.

`TestProjectedNullableDeclarationIdentity` checks primitive, object, union, array
and map declarations; intrinsic/occurrence nullability; both first-visit orders;
legal allocated names; validator name/reference agreement; and unchanged authored
presence. The extended `TestUnionNullAdmissionTransports` reuses the #824 specimen
and removes its projected Nullable exclusion, adding nullable object/nested union
branches alongside inherited primitive, Any and non-nullable controls. It fails
on the unmodified parent before rendering completes. Existing union-declaration,
validation-identity and selected-view models remain separate contracts.

`collectHTTPUnionTypes` consumes the same allocated concrete named identity,
retaining client/server membership. `TestHTTPNullableUnionDeclarationIdentity`
checks both collection orders and transport object modes. Selected-view and
collection element calls use the intrinsic `NamedValidationFunction.Name`;
projection initializer fragments use `GoValueTypeName`.
`TestNullableSelectedViewHelperNames` checks these call sites with occurrence
Nullable on an ordinary ResultType, a publicly authorable contract.

On parent `bf7ae2246` (after #832), expanded generated specimens fail while
rendering illegal nullable declarations. Current output compiles/vets and runs
null/concrete/missing/wrong-type controls, default/tiny/unknown view validation,
collection views and nested selected views. `TestNullableObjectUnionTransformCarrier`
restores intrinsic nullable nested unions and keeps all 15 carrier contexts.
Four bounded designs emit 83 artifacts, identical across separate current
processes. The ordinary required-union/recursive controls keep 18 parent bytes
unchanged; parent versions of the two expanded fixtures keep another 57 artifacts
unchanged. Exact output and compile/runtime evidence complements the model.

Limit: this model establishes declaration/call identity, not all nullable
selected-view carrier operations. A separate parent reproduction with ordinary
Child ResultType and both plain and Nullable Child fields generates an invalid
`result.Z != nil` comparison against `loom.Nullable[ChildView]`. The identifier
repair does not correct that guard or unwrap its argument. This separate carrier
failure is tracked as [#834](https://github.com/CaliLuke/loom/issues/834), outside
#829; generated acceptance uses representable nullable
union branches and ordinary nested ResultType fields.
