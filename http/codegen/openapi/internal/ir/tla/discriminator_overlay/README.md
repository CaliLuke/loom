# Named occurrence discriminator ownership

An annotated named occurrence is represented by a reference (or an explicit
allOf reference) plus local annotations and constraints. Its discriminator still
requires the complete branch graph and original named declaration. Moving the
hint to the wrapper caused an out-of-bounds union branch lookup and a false
ParentHint named-object rejection. Dropping the hint would instead lose authored
overrides; modifying the declaration would contaminate another occurrence.

`ComponentHintOwnership.tla` abstracts union/parent, ref/allOf, annotated/plain,
and inherited/overridden policy. Run explicitly with TLA+ tools 1.7.4:

```sh
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config overlay.cfg ComponentHintOwnership
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config drop.cfg ComponentHintOwnership
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config mutate.cfg ComponentHintOwnership
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 -config partition.cfg ComponentHintOwnership
```

The first three policies intentionally violate complete ownership, hint retention,
and source isolation respectively (exit 12). The partition policy checks all
four invariants over 32 distinct states (exit 0). These runs preceded the owner
correction. No model execution belongs to ordinary tests or delivery gates.

`analyzeUserTypeOverlay` partitions authored metadata before analyzing the named
component. `userTypeOverlayMetadata` retains identity and discriminator metadata
on that component and gives the annotation pass its own detached metadata.
Existing `CopyObjectOpenness` independently selects closure variants. Existing
component analysis and fingerprinting retain branch/mapping and named scope;
there is no wrapper guard, discarded hint, or reconstructed union graph.

`TestDiscriminatorNamedOccurrenceOverlay` covers both wrapper forms and both
hint kinds, occurrence override precedence, extensions, aliases/default mapping,
required constraints and source/component isolation. The inherited-hint test
checks annotation-first publication and reuse of the same base component.
The model abstracts policy values and ownership, not Go pointer graphs,
serialization, recursion, canonical-name conflict rules, or runtime matching.
Those remain implementation-test obligations; full IR tests and rendered native
consumer controls cover the surrounding graph and output behavior.
