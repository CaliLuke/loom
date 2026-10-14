# Replica observation test admission (#935)

1. [x] Reproduce the CI assumption failure while retaining the real replica owner: delay a committed mutation's reply until periodic reconciliation publishes its revision.
2. [x] Model the invalid assumption that every post-write observation requires a snapshot. Compare deterministic pending and already-observed fixtures against the actual contract.
3. [x] Exercise pending observation cancellation and Close directly at the shared observation owner, with a separate real writer and a deliberately stale observer. Retain committed-write, joined-operation and borrowed-client assertions. Check already-published observations separately; keep public concurrent SetAndWait coverage.
4. [x] Run focused races and native Redis controls; preserve the reproduction and model evidence.
5. [ ] Obtain independent exact review, then deliver one atomic test-only repair.

Initial parent: `23acfe91f6e2b82858edf515641c397ce214ddab`. CI run 38596570734 at `5870a8e90` failed `observation_owner_test.go:37`. The controlled parent reproduces a successful SetAndWait with no tagged HGETALL after 1.04 seconds. Runtime behavior is correct; production files are unchanged.

Acceptance: pending observations must enter the read before cancellation or Close, return context cancellation, publish no partial state, preserve the committed write and leave the borrowed client usable. An already-published revision must finish successfully without requiring another snapshot. The private pending-observation fixture deliberately excludes background publication; the public already-observed and concurrent-request tests retain real replica-owner coverage.

Evidence on Go 1.27.2 with jsonv2: focused race tests repeated ten times pass (12.019s); native Redis 8.10.2 race controls repeated ten times pass (17.186s); full rmap race package passes (9.292s); package golangci-lint reports zero issues. No generated behavior changes, so no generator sweep is required. TLC current behavior fails at 11/11 states; controlled fixtures pass at 6/6, including completion. See `pulse/rmap/tla/observation_test_admission/README.md` for reproduction and model scope.
