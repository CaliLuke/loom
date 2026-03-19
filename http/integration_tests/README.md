# HTTP Integration Tests

This module carries persistent generated HTTP fixtures that exercise transport behavior against the current local `goa-light` tree.

## Ticktock SSE Fixture

[`fixtures/ticktock`](/Users/luca/code/goa-light/http/integration_tests/fixtures/ticktock) is a checked-in generated HTTP SSE specimen. It exposes two simple SSE endpoints:

- `GET /tick`
- `GET /tock`

Each endpoint intentionally waits before emitting the first event. That delay is part of the contract check: clients must still observe `200 OK` and `Content-Type: text/event-stream` immediately when the stream is accepted.

## What The Tests Prove

[`tests/sse_fixture_test.go`](/Users/luca/code/goa-light/http/integration_tests/tests/sse_fixture_test.go) verifies:

- the generated HTTP SSE handler commits headers before the first event
- a real third-party client (`github.com/tmaxmax/go-sse`) can consume both `tick` and `tock` streams from the generated server

## Running

```bash
cd http/integration_tests
go test -count=1 ./...
```

From the repo root, `make integration-test` also runs this module.
