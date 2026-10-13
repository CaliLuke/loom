# Error declaration identity before inheritance (#699)

`ErrorDeclarations.tla` models one API, one service and one method with ordered
authored error lists. Exact wire names identify declarations within each list.
API errors are reusable definitions, not automatically inherited method errors.
The effective method list starts with its authored errors, followed by service
errors whose exact names are not overridden; duplicate inherited service names
are collapsed by the existing construction. Type validation is assumed valid.

This ordering matters: checking only effective method errors can miss an unused
API duplicate, a collapsed service duplicate, or a duplicate hidden by a method
override. Requiring globally unique names rejects valid narrower declarations.
The chosen admission checks each original scope before any finalization. It does
not change error lookup, override selection, ordering or Go name allocation.

## Run

Use Corretto 25 and pinned TLA+ 1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory:

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-error-declarations-local \
  -config Local.cfg ErrorDeclarations.tla
```

Run `Current.cfg`, `Effective.cfg` and `Global.cfg` similarly with separate
temporary metadirs. Preserve concise results, not checker state or raw logs.

| Configuration | Exit | Generated / distinct | Result |
| --- | --- | --- | --- |
| Current | 12 | 13 / 13 | Local duplicates reach accepted validation |
| Effective | 12 | 13 / 13 | Post-merge admission loses original scope evidence |
| Global | 12 | 19 / 19 | Valid cross-scope name reuse is rejected |
| Local | 0 | 30 / 30 | All ten bounded designs satisfy the invariants |

## Implementation correspondence and limits

`dsl.Error` appends original declarations. `RootExpr.Errors`, `ServiceExpr.Errors`
and pre-finalization `MethodExpr.Errors` own their local lists. `eval.RunDSL`
validates all sets before finalizing any of them. `MethodExpr.effectiveErrors`
defines the method/service merge; `RootExpr.Error` and `ServiceExpr.Error` own
reusable definition lookup. `validateErrorNames` in `expr/service.go` establishes exact-name
uniqueness for all three list owners, without modifying any list. Existing
per-error and effective discriminator checks remain in place.

`TestErrorDeclarationNames*` exercises the actual evaluator, unused/shadowed
duplicates, direct authored lists, cross-scope primitive type selection, siblings,
case distinctions and names that normalize to the same Go spelling. The generated
`TestErrorDeclarationScopeReuseCompiles` covers compatible API/service/method
declarations consumed by HTTP, JSON-RPC and gRPC; the existing constructor collision
fixture protects #734's separate generated-name policy.
`cmd/loom.TestGenerateRejectsDuplicateErrorDeclarations` confirms API, service
and mapped method duplicates fail in `eval.RunDSL` before any generated output;
its parent evidence distinguishes accepted designs from a late generator failure.

The model explores ten short authored lists and phase ordering, not arbitrary DSL
types or concurrent registration. The selection invariant explicitly assumes the
existing merge function remains unchanged; it does not independently prove that
function. Runtime direct tests check the resulting declaration type and order.
Generated artifact comparisons and compilation check the downstream consumers.
See [the active plan](../../../roadmap/error-declaration-admission.md).
