# Reusable transport reference planning

`plan_reusable.go` indexes all three component families before resolving
occurrences. It walks forward alias edges with a visited set, copies the terminal
value, and carries each edge's summary/description separately. `reusable_authored.go`
in the OpenAPI IR publishes canonical definitions and aliases before inferred
component allocation. `object_ref.go` preserves present empty annotations.

Run TLC explicitly (Java 27, TLC 1.7.4 release jar):

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC \
  -workers 1 -metadir /tmp/loom-reusable-copy -config copy.cfg ReusableReferences.tla
```

`copy` checks 486 states across three kinds, two names, nine possible two-node
reference graphs, and absent/empty/nonempty local annotations. `reject` reproduces
the former blanket admission boundary. `inline` loses identity, `mutate` changes
shared definitions, and `cycle` admits cyclic aliases. Run each negative control
with its matching configuration and a separate temporary metadir.

The model abstracts parsed content, naming collisions, schema generation, and
annotation applicability by target object kind. It does not prove arbitrary
OpenAPI parsing. The planner tests cover wrong kinds, missing targets and escaped
names. The Pet API round trip covers shared occurrences, generated compilation,
parsed contracts and process-isolated determinism; IR tests cover reserved names
and conflicts, and JSON/YAML tests cover empty annotation preservation. Model
checks are opt-in design tools, never ordinary build or CI gates.
