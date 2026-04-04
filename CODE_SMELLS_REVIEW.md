# Codebase Smell Review

This document summarizes the findings from a wide-ranging review of the codebase against the `AGENTS.md` guidelines and the `go-code-review` checklist.

*(Note: The build errors and explicit TODOs identified in earlier reviews have now been resolved! The codebase passes `go vet` and `golangci-lint` cleanly.)*

## 1. Ignored Errors (Violation of `AGENTS.md`)
The repository guidelines explicitly state: *"Never ignore errors or use `_ = call()`"*. However, there are numerous instances where errors are actively discarded:
*   `http/client.go:72`: `reqb, _ = io.ReadAll(req.Body)`
*   `http/integration_tests/harness/server.go`: Ignores process and teardown errors multiple times (e.g., `_ = server.Stop()`, `_ = s.cmd.Process.Kill()`, `_, _ = s.cmd.Process.Wait()`).
*   `grpc/middleware/xray/segment.go`: `ip, _, _ = net.SplitHostPort(...)`
*   `observability/otel/http_test.go`: `_, _ = w.Write([]byte("ok"))`
*   `http/codegen/testdata/payload_decode_functions.go`: Numerous instances of `c, _ = r.Cookie("c")` being generated.

## 2. Legacy Upstream Naming (Violation of `AGENTS.md`)
The guidelines mandate: *"Loom naming only: Do not introduce or keep legacy upstream-named aliases... Use `loom` naming exclusively."* Several files still contain references to the upstream `goa` project (over 100+ occurrences):
*   `http/integration_tests/fixtures/ticktock/clock.go:9`: Imports `"goa.design/clue/log"`.
*   `eval/error_internal_test.go`: Hardcoded paths like `/home/me/src/goa/eval/error.go`.
*   `dsl/api.go` and `dsl/http_file_server.go`: Multiple documentation examples still use `goa.design` URLs and emails instead of Loom equivalents.

## 3. Misuse of `panic()`
The Go code review checklist dictates that `panic` should only be used for truly exceptional, unrecoverable states, not normal error handling or validation. There are 88 `panic()` calls in the codebase. While many are marked with `// bug` (indicating an invariant breach), some are used as shortcuts for configuration or validation errors:
*   `http/mux.go`: `panic("too many wildcards")` — This should return an error during router initialization rather than crashing.
*   `expr/attribute.go`: `panic("Key " + keyName + " is not defined")` — DSL validation errors should be returned as `error` types so the DSL evaluator can format them nicely for the user.
*   `http/codegen/template_sources.go`: Overuses `panic(http.ErrAbortHandler)`. While this is an accepted pattern in standard `net/http` to silently abort, it is an architectural smell in a structured codegen/middleware framework.
