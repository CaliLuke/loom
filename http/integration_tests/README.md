# HTTP Integration Tests

This module carries persistent generated HTTP fixtures that exercise transport behavior against the current local `loom` tree.

## Ticktock SSE Fixture

[`fixtures/ticktock`](/Users/luca/code/loom/http/integration_tests/fixtures/ticktock) is a checked-in generated HTTP SSE specimen. It exposes two simple SSE endpoints:

- `GET /tick`
- `GET /tock`
- `GET /guarded`

`/tick` and `/tock` intentionally wait before emitting the first event. That delay is part of the contract check: clients must still observe `200 OK` and `Content-Type: text/event-stream` immediately when the stream is accepted.

`/guarded` exists specifically to prove the opposite branch: if the endpoint rejects before streaming starts, the client must receive the real HTTP error response instead of a silently committed SSE stream.

## What The Tests Prove

[`tests/sse_fixture_test.go`](/Users/luca/code/loom/http/integration_tests/tests/sse_fixture_test.go) verifies:

- the generated HTTP SSE handler commits headers before the first event
- a real third-party client (`github.com/tmaxmax/go-sse`) can consume both `tick` and `tock` streams from the generated server

[`tests/sse_adversarial_test.go`](/Users/luca/code/loom/http/integration_tests/tests/sse_adversarial_test.go) verifies:

- pre-stream endpoint failures preserve the real HTTP status (`/guarded` returns `401`, not `200 text/event-stream`)
- the checked-in fixture can be copied to a temp directory, repinned to the current repo root, regenerated, and compiled from scratch

## Running

```bash
cd http/integration_tests
go test -count=1 ./...
```

From the repo root, `make integration-test` also runs this module.
