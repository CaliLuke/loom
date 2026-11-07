# Named method customization ownership

`Payload`, `Result`, `StreamingPayload` and `StreamingResult` accept a named
type with a customization function. The method occurrence owns a detached copy
of that type. The function runs exactly once on the occurrence. Only after it
completes does the occurrence decide whether the copy needs its own declaration:
an authored `Required` or `Extend` renames the copy, adds the requirements to it
and moves the `Extend` bases onto its innermost declaration. Examples,
validations and other metadata stay on the occurrence and keep the declaration
identity. Shared named types never change.

The model abstracts one occurrence. The callback has one authored effect
(`sample`, `required` or `extend`) and may install a different shared type
directly on the occurrence. Public DSL cannot do this in a named method
callback: `Attribute` and `AdditionalAttributes` initialize only an untyped
attribute, and `Elem`/`Key` require collections. Go callbacks can still assign
the type through `eval.Current()`. Each run adds one sample and repeats its
authored effect. Requirements and samples are abstracted as sets and counts.

| Configuration | Policy | Result | Generated / distinct states |
| --- | --- | --- | --- |
| `two-owner` | Helper runs and materializes, then `methodDSL` runs again and moves bases | `OnceBeforeMaterialization` fails at the second run, depth 4 | 19 / 19 |
| `two-owner-isolation` | Same | `SourceIsolated` fails: replaced shared type receives the bases, depth 5 | 30 / 30 |
| `captured` | One run, but materializes the copy captured before the callback | `NoStaleMaterialization` fails, depth 3 | 16 / 16 |
| `once` | One run, then materialize the completed occurrence's own copy | All five invariants pass, depth 3 | 24 / 18 |

The `two-owner` policy reproduces 404d `methodUserTypeAttribute` followed by
`methodDSL`: duplicated samples, Meta values, Extend bases and diagnostics, and
a base move onto whatever type the occurrence holds after the second run.
`captured` is the rejected single-run candidate that keeps the helper's
pre-callback pointer.

Implementation correspondence:

- `methodDSL` copies a customized named type once (`expr.Dup`), sets the
  description argument, and runs the function in the one `eval.Execute` shared
  with function-form and direct DataType arguments. Plain `Payload(Type)` keeps
  the shared reference.
- `materializeMethodType` runs after `applyUnionMetaFromAttribute`. It acts only
  when `att.Type` is still the copy and not `Empty`. Requirements go to the copy,
  the copy is renamed `<Type>_<method>_<Position>`, and alias links are renamed
  before the bases move. The obsolete helper and its always-zero `numreqs`
  comparison are deleted: the occurrence is newly allocated, so any authored
  `Required` creates the method declaration, as before.

`dsl/method_customization_test.go` checks the four public functions with named
scalar and object types. It covers callback count, the exact local sample, the
retained contributor sample and bound (1/min 0 against 8/min 5), Required plus
Extend declaration ownership with base and sibling isolation, and result-type
default views. It also covers recursive identity, union metadata, a requirement
already declared by the base, a replaced shared type, single diagnostics, and
direct primitive and function-form controls. The existing
`expr/method_extended_type_test.go` matrix and the generated
`TestExtendedMethodTypesGenerated`, `TestCustomizedResultViewsGeneratedModule`
and `TestGeneratedRequiredOccurrences` modules check renamed declarations, alias
chains and views end to end.

The model does not cover Go graph copying, composition after DSL evaluation,
view finalization or code generation; those remain test obligations.

Run explicitly with Java 25 and TLA+ tools 1.7.4:

```sh
for cfg in two-owner two-owner-isolation captured once; do
  java -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" tlc2.TLC \
    -workers 1 -metadir "/tmp/loom-method-customization-$cfg" \
    -config "dsl/tla/method_customization/$cfg.cfg" \
    dsl/tla/method_customization/MethodCustomization.tla
done
```

The first three exit 12; `once` exits 0. These are opt-in design checks,
never build, test, hook or CI requirements. Delete checker directories after
review.
