# Distributed suite ownership

Run explicitly with Lean 4.35.0 from the repository root:

```sh
lean scripts/coveragecheck/lean/Suite.lean
```

The model informed the suite design before implementation: each package has
one lane owner; a result is accepted only when every required lane completed
successfully for the selected source; excluded consumers cannot inflate a
protected boundary. Race and coverage observe the same test execution.

Theorems:

- `every_package_has_owner`: each package belongs to its selected lane.
- `no_duplicate_owner`: two claimed owners of one package must be equal.
- `accepted_is_complete`: acceptance includes every required successful lane.
- `excluded_consumer_cannot_inflate`: excluded observations do not change a floor.

The implementation seams are `packageLane`, `runLane`, `mergeSuite` and the
selected-versus-remaining coverage split. `TestDistributedSuite` executes a
miniature repository and checks discovery, ownership, successful aggregation,
rejected source/config/inventory mismatches and a failed retry. The existing
suite regression checks the consumer boundary and test failures.

The proof assumes a fixed inventory and immutable source, successful results
truthfully represent executed tests, and atomic coverage reports observations
correctly. It does not establish Go compiler correctness, GitHub artifact
transport or infrastructure availability. No proof execution belongs in CI,
hooks, default tests or release eligibility.
