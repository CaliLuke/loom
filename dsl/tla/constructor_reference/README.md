# Pending constructor occurrence completion

Function-form `OneOf` permits pending named branches just as the compact
constructor does. A branch refinement needs the referenced declaration to be
registered and completed before it can call `Required`, localize the type, or
publish a transport copy. Deferring only until expression preparation is too
late: HTTP DSL body copies occur earlier.

`ConstructorReference.tla` exposes both ordering defects. `eager.cfg` refines an
unknown target and fails `KnownBeforeRefinement`. `deferred.cfg` allows a copy
before completion and fails `CompleteBeforePublication` in four states.
`completed.cfg` completes before publication and checks five states, including
its temporal obligation. An earlier model without publication was insufficient
and was replaced by this model.

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config eager.cfg -metadir /tmp/loom-constructor-eager ConstructorReference.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config deferred.cfg -metadir /tmp/loom-constructor-deferred ConstructorReference.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config completed.cfg -metadir /tmp/loom-constructor-completed ConstructorReference.tla
```

`AttributeExpr.ResolveTypeReference` is one idempotent lifecycle: assign the
registered target, consume the deferred callback, then execute the refinement
against the actual occurrence. `typeDSL` uses it before completion/publication,
including functionless named Types; ordinary preparation uses the same owner as
a fallback. Existing running/done and kept-reference ownership handles cycles.
No separate callback queue or constructor-specific resolver is introduced.

`TestFunctionUnionKeepsPendingBranches` exercises tagged and untagged recursion,
HTTP Body copies and OpenAPIBody. `TestFunctionUnionRefinementOwnership` checks
copied constructors, mutually recursive object graphs, exactly-once callbacks,
repeated completion, occurrence overrides, source isolation and hint neutrality.
`TestFunctionUnionReportsMissingBranch` retains the missing-name diagnostic.
Importer generated round trips compile and decode recursive payloads. The model
has one occurrence and no mutable Go graph; these tests establish correspondence
for copies, multiple occurrences, recursion and error reporting. All model runs
are explicit design tools, never ordinary test or CI requirements.

## Completion scope and existing structural ownership

An intermediate implementation eagerly evaluated every named dependency while
completing a Type. In `Y -> X -> localized Y -> X`, that changes completion order:
the enclosing X has already completed when Y's local snapshot is published, so
ordinary copying makes a second X instead of retaining its structural back-edge.
The maintained `TestLocalizedStructuralBackReference` exposed the regression.

`CompletionScope.tla` records that observed snapshot effect as an abstraction of
the existing kept-reference implementation. `scope-broad.cfg` reproduces lost
owner identity (seven generated/six distinct states); `scope-owned.cfg` permits
only owned pending completion before the owner finishes and preserves identity.
This model does not prove the Go copier: the adjacent localization model and
exact-pointer regression establish that correspondence.

```sh
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config scope-broad.cfg -metadir /tmp/loom-completion-broad CompletionScope.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 -config scope-owned.cfg -metadir /tmp/loom-completion-owned CompletionScope.tla
```

`walkOwnedTypeAttributes` visits inline occurrences in source order and stops at
named declaration boundaries. It is shared with localization's owned-root
inventory. The existing named-declaration scheduler and kept-reference
publication contracts remain unchanged. Broad early traversal was unnecessary;
changing snapshot semantics to accommodate it was rejected.
