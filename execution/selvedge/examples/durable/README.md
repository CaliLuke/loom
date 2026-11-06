# Durable object metadata

The application shares `contract.Inspect` and `contract.Read` between two
binaries. The producer admits and inspects jobs; the worker runs an HTTP HEAD
activity and returns object metadata. Neither binary needs generated code.

From the repository root, admit a job before starting a worker:

```sh
go run ./execution/selvedge/examples/durable/producer -database metadata.db -id object-1 -url https://example.com/object
go run ./execution/selvedge/examples/durable/producer -database metadata.db -action lookup -id object-1
```

The job is Running, with no local execution. In another terminal:

```sh
go run ./execution/selvedge/examples/durable/worker -database metadata.db
```

The worker prints `ready` after startup. Repeat lookup to observe Completed and
the metadata. Retry submission with the same ID and input to receive
`Created: false`; a different input under that ID reports a conflict. Ctrl-C
asks the worker to drain and shut down. Activity attempts can repeat after a
crash, so application side effects must tolerate retries.

`-database` also accepts a PostgreSQL URL. Both examples use native automatic
migration. Deployments can instead run migrations separately and configure
`WithAutoMigrate(false)` in application bootstrap. The worker's `-lease` and
`-poll` options configure native execution budgets and fallback polling; neither
changes the typed contract or ID semantics.

## Verification

`TestWorkflowTwoProcess` builds these binaries, submits with the worker absent,
then kills a worker after its HTTP activity has reached a controlled local
server. A new worker completes the job from the same SQLite database. The test
checks stable admission, pending status, metadata and worker restart without
adding test-only flags to the examples.

Run the explicit PostgreSQL acceptance using:

```sh
make test-workflow-postgres
```

The helper provisions only its own disposable loopback container, pinned to
PostgreSQL 18.6 and its verified official manifest, runs the race-enabled
admission and process-restart tests, and removes its container/storage. Missing
Docker or a skipped/unavailable database is a failure in this explicit suite.
The ordinary unit suite skips PostgreSQL when no test DSN is supplied.
