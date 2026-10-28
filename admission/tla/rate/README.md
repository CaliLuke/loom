# Atomic rate policy and quota conservation

`RatePolicy.tla` compares the accepted live-update rule with two rejected
candidate designs before implementation. This is a new integration, not a claim
of a defect in the maintained token-bucket dependency.

The model represents one local budget with integer refill ticks and three
rate/burst pairs. Consume, refill and update actions interleave for five steps.
The checked update retains available allowance, clamped to the new burst. It
never refills solely because settings changed, including unchanged settings.
`Conservation` requires available plus spent tokens to stay within initial
allowance plus actual refill. `CoherentPolicy` rejects a partially published
rate/burst pair; `Bounded` checks the effective allowance.

Correspondence and limits:

- `../../rate.go` serializes Allow, SetPolicy and Policy under one mutex. The
  maintained `golang.org/x/time/rate` v0.16.0 owns token accounting. SetLimitAt
  settles old-rate refill before SetBurstAt at the same timestamp. The library
  clamps its stored allowance lazily on the next observation; this model's
  `tokens` is the observable allowance, not the library's private field.
- Direct tests cover fractional refill, rate/burst increases and decreases,
  unchanged and invalid updates, cancellation, bounded retry delays, coherent
  versions and simultaneous updates/admissions. The model does not prove Go
  mutex refinement, floating-point arithmetic, time conversion or counter
  overflow. It abstracts versions as policy identity and excludes invalid
  policies; tests cover those obligations.
- Grouping and storage belong to the application. Independent group budgets
  are tested through the runtime and generated example. The model does not
  establish a fleet-wide quota or a cache eviction policy.
- `RateAdmission` checks are synchronous and require a backend honoring context
  cancellation. Direct and generated tests check deadline failure, safe error
  handling, preserved contexts and protected access ordering. Those properties
  are outside this quota model. No goroutine abandonment or retry is modeled.

Use the pinned repository TLC distribution (`tla2tools-1.7.4.jar`, TLC2 2.19,
revision 5a47802) with Java 25.0.5, from this directory:

```sh
java -XX:+UseParallelGC -cp "$HOME/.local/share/tlaplus/tla2tools-1.7.4.jar" \
  tlc2.TLC -workers 1 -metadir /tmp/loom-rate-model \
  -config checked.cfg RatePolicy.tla
```

The checked model passes 122 distinct states (349 generated). `reset.cfg`
replenishes the bucket on update and violates conservation after 4 distinct
states. `split.cfg` publishes rate before burst and violates coherent policy
after 6 distinct states. These controls informed preserving allowance and
locking the complete policy transition.

`CheckDeadline.tla` models an independently delayed context timer callback. The
initial implementation checked only `ctx.Err()` after a backend returned. An
independent review probe reproduced late admission in 9 of 10 calls with one Go
scheduler processor and a backend returning after its deadline. The
`deadline-err-only.cfg` model reproduces that defect in 8 distinct states:
deadline expiry, a successful backend answer, then admission before the timer
signals. `deadline-checked.cfg` passes 9 distinct states (12 generated) by checking
clock expiry as well as the published signal. Run it with the command above,
replacing the module with `CheckDeadline.tla` and selecting the named config.

The shared private `rateContextError` owns that predicate for the rate bucket
and admission boundary. `rate_deadline_test.go` checks a fixed elapsed deadline
with pending timer publication deterministically, plus an actual late backend
answer without waiting on Done. The latter must reject with either timer
ordering. Both regressions failed before the fix. The model ends at the decision;
it does not claim that time cannot advance afterward or forcibly stop a callback.

Run Go tests with `go test -race ./admission ./http/middleware`. The generated
boundary is `TestRateAdmissionExample` in
`codegen/generator/admission_integration_test.go`; select local Loom source
while developing an unpushed change. It generates the exact public example and
runs it under the race detector. Formal tools remain opt-in, outside all normal
build, test, hook, CI and release gates. Delete task-owned checker output after
validation and independent review; keep these sources and findings.
