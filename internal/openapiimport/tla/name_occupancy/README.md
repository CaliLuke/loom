# Imported name occupancy

The former `uniqueName` counted base occurrences without claiming generated suffix
names. `counter.cfg` models that behavior; `occupied.cfg` selects and claims the
first available candidate, matching the existing NameScope allocation invariant.
Both explore every order of two identical bases and one numbered base.

Run with the repository's pinned TLC 1.7.4 jar and Java toolchain:

```sh
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-688-counter-state -config counter.cfg NameOccupancy.tla
java -cp /Users/luca/.local/share/tlaplus/tla2tools-1.7.4.jar tlc2.TLC \
  -metadir /tmp/loom-688-occupied-state -config occupied.cfg NameOccupancy.tla
```

The counter version must fail `Distinct`; occupancy must pass. This finite model
does not establish Go spelling, case folding, arbitrary suffix ranges, component
identity, or correct consumer adoption. Direct allocator/consumer tests and
generated compilation check those implementation boundaries. Source references
and wire names remain separate from allocated Go names. No order-independent
spelling is promised: allocation follows each caller's deterministic traversal.

The counter model fails `Distinct` with 11 generated / 9 distinct states; the
occupied model passes with 18 generated / 12 distinct states.
Remove temporary checker state and raw logs after independent final review,
preserving reproduction inputs and concise results.
