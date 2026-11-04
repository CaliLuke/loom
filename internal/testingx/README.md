# Generated contract batches

Use `ModuleBatch` when several generated designs share the same Loom source and
dependency set. Resolve the source through the owning transport's existing
resolver, allocate the batch in the parent test, and add cases sequentially.
Pass each case's `ModulePath + "/gen"` to the generator. `WriteFile` rebases
handwritten harness imports; it must never rewrite generated files. Call `Check`
after the case subtests return. A focused `go test -run Parent/Case` still runs
that case's generated contracts.

Each case retains separate packages and test binaries, isolating global flags,
service names and protobuf registrations. The parent owns temporary directories,
so subtest cleanup cannot remove inputs before validation. One module shares
`go mod tidy`, full `go vet ./...` and `go test -race -count=1 ./...`. The test
command disables its implicit vet because the explicit vet already ran. A
separate `go build` would repeat compilation. Runtime tests have a 30-second
per-binary timeout; compilation is outside that timeout.

An export directory preserves the original module imports and case paths for
artifact comparisons. Exported modules are validated separately. Existing
transport-specific generate-only switches still skip validation explicitly.
Do not batch cases requiring incompatible dependencies or authored external
module paths without preserving those contracts.

## Retained regression inventory

The consolidation from `6e64a3d885753cceda57ebe206369cddcd0bf525` preserves all
87 generated variants below while reducing module validation runs from 82 to 12.

| Test family | HTTP variants | JSON-RPC variants | gRPC variants | Contract retained |
| --- | ---: | ---: | ---: | --- |
| Selected body | 18 | 18 | — | Five directions, renamed fields, six additional carriers, optional views; client/server and CLI runtime checks where previously present |
| Anonymous validation | 4 | 4 | — | Nullable and enum combinations, valid and invalid wire payloads |
| Named nullable validation | 3 | 3 | — | Object, array and map validation |
| Nullable collection responses | 6 | — | — | Nil, empty, populated and invalid response values and problem details |
| Client member identity | 6 | — | — | Naming reservations plus mounted endpoint, group, CLI and streaming bindings |
| Server member identity | 2 | — | 8 | Naming and emitted declaration inventory, compilation, mounted dispatch and authentication |
| Custom text clients | 9 | — | — | Path, query, header and cookie codecs, validation, defaults and CLI conversion |
| Request body contracts | 6 | — | — | Existing six-case body batch and its unchanged harnesses |

The HTTP selected-field identity, HTTP client/server member identity and gRPC
member-declaration assertions now run in their corresponding generated cases.
Their four separate drivers and repeated DSL evaluations were removed; their
assertions were retained. Three selected-body compilation drivers per transport
became one matrix each. Distinct mapped-field fallback assertions and custom
Stringer external-type tests remain separate.

gRPC member cases render service and transport files, without empty
consumer-owned response-contract scaffolds. Those placeholder scenario maps
were not executed by the previous member tests. The dedicated scaffold tests
still render, populate and execute a real response contract.

## Validation evidence

Local uncached test executions with warm compiler caches on Go 1.27.3,
Darwin/arm64, and `LOOM_DIR` selecting this checkout:

| Same case inventory | Parent seconds | Batched seconds |
| --- | ---: | ---: |
| HTTP selected bodies, anonymous/named validation, collection responses and client/server members | 121.9 | 29.4 |
| HTTP custom text clients | 35.5 | 9.2 |
| JSON-RPC selected bodies and anonymous/named validation | 92.6 | 27.6 |
| gRPC server members | 27.5 | 7.3 |

These are development samples, with some concurrent checks, not CI latency
predictions or cold-build benchmarks. Timings exclude nested subtest durations
from totals. The existing six-case request-body batch is not counted as a new
speedup.

Parent test sources loaded through Go's file overlay and the revised tests
exported 19 representative cases: all HTTP/JSON-RPC anonymous variants, all HTTP
client/server members, and ordinary, authenticated and located-import gRPC
members. All 247 generated files matched byte for byte. Normal batches change
only their generated import namespaces; harness bodies and DSL inputs remain
unchanged. The standalone module headers use the common replaced Loom requirement.

`TestModuleBatchContracts` executes two real modules in both batch and export
modes, checks distinct namespace values, verifies execution after child cleanup,
and fails without race instrumentation. An isolated overlay that inverted its
value assertion made the generated test and parent suite fail for that assertion.
Focused race/shuffle validation covers every migrated matrix and callers of the
split render helpers; repository lint is the commit gate. No exhaustive design
corpus or proof execution is added to CI.
