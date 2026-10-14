# Problem response representation and field projection

This model accompanied the #797 parent reproduction at
`66a9dc9090a06cd4136be1e5ebdaf26883ff90fa`. The implementation is now based on
`25179e87b8b5d4e9cf5bb233d1818600469d060d` and was delivered in
`14ab5a758c7ca94857145fa374c96be34c5df2eb`.

`ProblemProjection.tla` distinguishes the physical ServiceError fields, the
normalized public problem fields, and the selected body fields. Three public
fields abstract required string and optional guidance members. Every subset is
checked, including empty, scalar selection, pruned object and complete body.

- `legacy.cfg` fails Renderable after 10 states: a selected public problem member
  is read directly from the differently shaped ServiceError.
- `source-only.cfg` fails after 18 states: normalizing the server source is not
  sufficient when the client still reads complete-body fields from a projection.
- `checked.cfg` checks all 24 states: normalize the source and derive reads/writes
  from the same selected body membership. Nonbody members come from their mapped
  arguments, as already selected by the effective HTTP wire-binding owner.

Run with TLC 2.19 / Java 25, keeping checker output outside the repository:

```sh
java -XX:+UseParallelGC -cp "$HOME/.cache/loom-tools/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -deadlock -metadir /tmp/loom-problem-model \
  -config checked.cfg ProblemProjection.tla
```

The model does not prove Go compilation, local-name allocation, optional wrapper
conversion, error text safety, formatter behavior or OpenAPI media selection.
Those are direct/generated/runtime acceptance obligations. It intentionally does
not invent a raw-text leak in the current noncompiling scalar path.

Concrete correspondence:

- The parent client shortcut read complete-body Code/Detail/Instance/RetryHint
  from every nonempty body. Its server shortcut constructed every field after
  header/cookie mappings had pruned members; scalar projection read ServiceError.
- `problemRepresentation` describes the native public runtime document once.
  `responseBodySource` selects that physical source before server body projection.
  `partial_response` materializes `http.NewProblemResponse` before body/header/
  cookie reads. Custom formatters receive the whole error for every body shape.
- `errorBuilder.buildProblemClientResult` derives scalar/object/empty body and
  mapped-field inputs from the prepared bindings, uses generic transforms and
  field initialization, then invokes the existing ProblemErrorFromBody adapter.
  The complete-body handwritten constructors and decoder shortcut are removed.
- `TestSelectedProblemBodyConstruction` compiles sixteen generated modules and
  exercises mounted handlers/imported clients for all builtin shapes, optional
  guidance, safe-message overrides, automatic headers, explicit header/cookie
  mappings and formatter overrides. The custom selected-error control and existing
  custom media suite preserve ordinary errors. Type/title overrides are covered.
  Malformed and incomplete client bodies fail decoding/validation before the
  public representation is reconstructed.
- The three `*_design.go.txt` inputs preserve the bounded process-isolated
  comparison: selected/pruned/empty built-in errors in a service named `problem`,
  a full problem control, and a custom text error control. The first fails parent
  compilation and all candidates compile/vet. Public specs remain byte-identical.

RFC 9457 defines problem JSON as an object. Existing HTTP expression finalization
selects the media type from the actual body result type; scalar field selection
loses the problem result type and uses ordinary response encoding. Preserve that
boundary and test JSON/YAML output and response Content-Type; do not inherit
problem+json merely from the original error type.
See https://www.rfc-editor.org/rfc/rfc9457.html#section-3.
