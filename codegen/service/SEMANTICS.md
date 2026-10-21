# Service value ownership during compilation

`generator.generationFiles` creates one `SemanticData` for each evaluated root
after preparation plugins finish. It captures effective method payload, result,
stream and error occurrences before backend analysis. Service code and transport
code use separate `ServicesData` instances over that owner. OpenAPI receives the
same owner through `WithServiceSemantics`.

The shared boundary contains immutable captured occurrences and value resolution
caches. It contains no Go names or selected example records. Each backend owns
its `ValueData` records; package-name reanalysis within one backend retains that
backend's records. A new compilation creates a new owner. There is no global
cache, and a method from another design is not admitted.

Example synthesis is policy-dependent. Each backend analysis has a fresh
synthesis scope, preserving custom randomizer call sequences even when builtin
seeded generators would produce equal values. Documentation uses the default
scope and may disable
synthesis while preserving authored examples. Cache keys include this scope so
an earlier backend cannot determine another backend's representative example.
Authored resolution still shares occurrence/source/role identity. Scope is a cache boundary, not a random seed; existing scoped generators own
seeds. Backend scope counters are local to the compilation and never appear in
generated output.

This is deliberately a method-value boundary. Go analysis still normalizes raw
object wrappers and allocates backend names. Component-only schemas, independently
authored document bodies and transport-local examples retain their own semantic
occurrences. Target plans still capture target graphs; declarations are not yet
interned across occurrences. Generated files and backend data still remain live
through the existing plugin and render stages. This change therefore neither
freezes the entire expression graph nor bounds total compilation memory.

Standalone `NewServicesData` and OpenAPI entrypoints create their own owners or
captures. Plugins retain the existing preparation and generated-file hooks;
there is no plugin API migration in this step.

See the [ownership model](tla/semantic_ownership/README.md) for the rejected
alternatives, checks and limits. Allocation and process-memory measurements must
be compared separately: fewer temporary allocations do not imply lower peak
resident memory. Compare generation from the same evaluated design in fresh
processes, preserve the baseline binary, alternate before/after runs, and verify
all emitted bytes and generated compilation before interpreting performance.
