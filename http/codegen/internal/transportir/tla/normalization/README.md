# HTTP normalization input analysis

`NormalizationPass` compares one immutable input capture with recursive
recapture of each remaining subtree. The production owner is
`http/codegen/internal/transportir/normalizeHTTPAttribute`; lowering carries
`expr.ValueOccurrence` children while mutating a detached output graph.

The bounded model uses a chain of eight named layers, each contributing one
predicate. Capture retains every layer's effective predicate set. Lowering
writes output from that immutable snapshot. Work counts captured node visits,
not bytes, GC cycles or total constraint-solving operations. The old design
revisits each remaining suffix; the chosen design captures the chain once.
Both designs preserve meaning in this model: recapture is a resource defect,
not a modeled semantic bug.

With TLA+ v1.7.4 (TLC 2.19), jar SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`, and
Corretto Java 25.0.4.1:

- `Frozen.cfg` preserves frozen input, lowered meaning, one input analysis and
  completion: 11 generated states, 10 distinct, depth 10.
- `Recaptured.cfg` violates `OneInputAnalysis` at the second lowering step:
  4 generated/distinct states, depth 4. Eight initial node visits plus seven
  repeated visits make 15, exceeding the eight-node input budget.

From this directory:

```sh
java -Xmx512m -cp "$TLA_JAR" tlc2.TLC -workers 1 \
  -metadir "$TLC_STATE" -config Frozen.cfg NormalizationPass.tla
```

Repeat with Recaptured.cfg, requiring the named resource invariant failure.
Keep checker states, traces and raw logs outside the repository.

The model assumes valid finite input and exact captured constraints. It does not
prove Go graph pairing, shared declarations, recursive type-ID stopping, clone
behavior, default/enum materialization or alias provenance. Direct normalization
tests cover those implementation boundaries; representative generated comparisons
cover observable bytes. The nested-depth regression fails the original pass:
16/64 levels allocate 2,216/27,912 objects. With one capture they allocate 382/1,406
under Go 1.27.3. Its tolerant growth threshold detects repeated subtree capture
without a wall-clock assertion. The benchmark extends the depth sample to 256.
These numbers do not bound arbitrary constraint solving or whole compilation.
