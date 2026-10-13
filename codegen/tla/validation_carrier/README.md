# Validation carrier agreement (#803)

The shared validator has two independent representation facts: its current
object may be an extracted concrete value, while the fields inside it retain
pointer layout. The named occurrence renderer already distinguishes these facts;
anonymous occurrences and object enum predicates previously did not.

`Carrier.tla` models named/anonymous occurrences, concrete unwrapping and forced
field layout. Native anonymous controls use forced pointer layout, matching the
HTTP request body; named native controls cover either field layout. It does not
model every primitive/collection storage rule or infer layout from source syntax.

With Java25 and TLA tools1.7.4 (TLC2.19), JAR SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`:

```sh
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config legacy.cfg -metadir /tmp/loom-carrier-legacy Carrier.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config clear-layout.cfg -metadir /tmp/loom-carrier-clear Carrier.tla
java -XX:+UseParallelGC -cp "$TLA_JAR" tlc2.TLC -workers 1 -config scoped.cfg -metadir /tmp/loom-carrier-scoped Carrier.tla
```

Legacy fails `TypedGuard`: an extracted anonymous struct is compared with nil.
Clearing field layout fails `FieldLayout`: native child fields no longer retain
their declared representation. Both negatives find 13 distinct states. Scoped
carrier state passes all14 states, preserving native guards and field layout.
These bounded representation checks do not establish Go typing or wire behavior.

Implementation correspondence:

- `presenceValueContext` marks unwrapped objects with `validationValue` while
  retaining pointer and registered named field layout.
- `validationTargetIsPointer` combines that occurrence fact with the existing
  required/default/forced-pointer predicate. Anonymous and alias traversal and
  predicate render data use that decision. Named calls continue to use their
  physical declaration owner and the same occurrence fact.
- `renderObjectValidation` resets `validationValue` at each child; unwrapping a
  child sets a fresh occurrence fact. No descendant inherits the parent's value
  carrier merely because it inherits field layout.
- `TestAnonymousValidationCarrier` covers Optional/Nullable and object enums;
  generated HTTP/JSON-RPC tests compile the independently emitted declarations
  and exercise named, anonymous, array and map descendants, absent/null values,
  valid input and rejected predicates.

Keep this model alongside the recursive-validation model, whose named graph
coverage deliberately abstracts physical Go carrier agreement. Primitive aliases,
byte slices and named Nullable declarations remain independent regression controls.
Do not commit checker states or raw output; remove task-owned output after review.

## Alias spine extraction (#892)

`AliasPresence.tla` refines the root-versus-child distinction for inherited
nullability. It enumerates alias depths zero through three. The `alias-shallow`
configuration reproduces clearing only the occurrence and first named level;
`alias-whole-graph` rejects clearing structural child nullability. `alias-spine`
clears every root alias flag while retaining the child and original source.
Run each configuration with the command above, substituting `AliasPresence.tla`
and its `alias-*.cfg` filename and a separate temporary state directory.
Shallow clearing fails `RootConcrete` in 7 states; whole-graph clearing fails
`ChildPreserved` in 5 states. Spine clearing passes all 8 states. Source
preservation is assumed by the transition and checked by the Go tests.

The model abstracts the alias spine as indexed flags. It does not prove Go graph
copying, type identity, recursion termination or generated compilation. Direct
`TestConcretePresenceAliases` and `TestConcretePresenceRecursiveChild` check the
actual shared helper and its source/child boundaries. The two existing nullable
reference importer tests exercise its generated HTTP conversion consumer. The
helper follows named edges only; structural child edges keep their original
targets, so consuming the parent carrier does not consume a child's carrier.
