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
