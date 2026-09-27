# Request body analysis

`RequestBodyAnalysis.tla` models two endpoints sharing an authored body shape.
The integer for each endpoint counts suffix applications to a nested user type.
Analysis may run zero, one, or two times before finalization; the checker explores
both endpoint orders. Finalization must apply exactly one transport suffix.
The old implementation stores the derived body during analysis, so one analysis
followed by finalization violates `RenamedOnce`. Pure analysis preserves the
source until finalization and satisfies the invariant in this bounded model.

Run with a downloaded TLA+ tools jar:

```sh
cd expr/tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config RequestBodyAnalysis_before.cfg RequestBodyAnalysis.tla
java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config RequestBodyAnalysis_after.cfg RequestBodyAnalysis.tla
```

The before configuration must report an invariant violation; after must pass.
This is an abstract lifecycle model, not a proof of Go graph cloning. The direct
`TestHTTPRequestBodyAnalysisPreservesSource` regression exercises actual array,
map, and inline-object graphs, repeated derivation, source pointer identity, and
names that already end in `RequestBody`. DSL and generated-module tests check
union branches, shared constructor allocation, presence validation, and compile.
