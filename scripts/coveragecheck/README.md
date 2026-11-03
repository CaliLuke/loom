# Combined race and coverage suite

`make test-ci` executes every root-module package once with `-race`,
`-shuffle=on`, `-count=1` and atomic coverage. The configured consumers in
`coverage/baselines.json` determine protected boundary coverage; other packages
run separately and contribute only to the aggregate profile.

CI distributes the same package inventory with:

```sh
make test-ci-lane TEST_LANE=http COVERAGE_PARTS=/tmp/loom-coverage
make test-ci-merge COVERAGE_PARTS=/tmp/loom-coverage
```

Run all six lanes before merging: `http`, `grpc`, `generator`, `jsonrpc`,
`importer`, and `remaining`. `packageLane` owns the partition; remaining receives
new packages automatically. Discovery uses `go list -race` so race-only
packages are included. Keep whole-package tests together for shuffled-order
checks. `CI_SOURCE` defaults to the checked-out commit; CI checks out the same
immutable revision for every lane and the merger. Do not combine results from
different working trees or reuse results after modifying an uncommitted tree.

Each successful lane writes profiles and then a completion manifest. The
merger verifies every lane, exact source/configuration/package membership and
profile digests before checking floors and publishing `cover.out`. A failed
retry removes the old completion marker. Aggregation never runs tests again.
The artifact download is scoped to the current workflow run. Keep the workflow
matrix aligned with `suiteLanes`; missing lanes fail aggregation.

`make test-race` remains an independent local command; `ci-local` does not run
it after `test-ci`. `depend-test` installs the test tools, while `depend` also
installs the linter. Formal reasoning is [explicitly opt-in](lean/README.md).

## Consolidation evidence

The original CI ran the full root suite for coverage and again across four
race lanes. At source `ba329c3cf1093855c2ba48939c13993b3a0540ff`, GitHub run
[41523373041](https://github.com/CaliLuke/loom/actions/runs/41523373041) spent
31.8 minutes in the coverage step and 14–18 minutes per race lane. These times
overlap and must not be added as elapsed time. The new pipeline combines both
observers and separates HTTP, gRPC, generator, JSON-RPC and importer packages
onto individual runners. CI wall-time improvement requires a new runner sample;
the number of root test executions drops from two to one by construction.

`TestRequestBodyContractsGenerated` consolidates six temporary-module suites
into one module containing separate generated namespaces and harness packages.
All six designs and seven complete runtime harness constants are unchanged.
The selected local Go 1.27.3 macOS/arm64 run fell from 20.040s (including the
retired compile-only test) to 6.853s; this is a warm-cache sample, not a CI SLA.
The retained checks cover client/server round trips, absent/null/empty values,
invalid and malformed input, problem responses, service invocation, CLI
payloads, named/nested collections and WebSocket streams.

`TestLargeErrorSetHTTPClientIntegration` only generated and compiled a module;
it supplied no runtime tests. Its exported `LargeErrorSetHTTPClientDSL` remains
in the compile corpus, which additionally generates the example app and vets
the module. The exact retained corpus case passed during consolidation.

The former shell race-partition test used command stubs. `TestDistributedSuite`
replaces it with real miniature modules, once-only markers, race-only discovery,
coverage aggregation, source/config/inventory mismatch rejection, corrupted
profile rejection, protected-consumer floors and failed-retry invalidation.
