# Mixed HTTP/SSE fixture parity repair — #841

CI at pushed39c33f48b fails five mixedtick tests before transport assertions:
their generated-tree comparison finds stale decoder error-variable placement and
a superseded local error-dispatch path. Production generator corrections already
exist; this repair must regenerate the owned fixture, preserving parity checks.

1. Reproduce a focused mixedtick test at pushed main and retain CI's exact diff.
   Read the fixture design, module and regeneration harness before changing it.
2. File the self-contained CI regression after independent editorial review.
   Regenerate with the current Loom CLI from the fixture module; never edit gen.
3. Compare all old/new generated artifacts, list intended differences and repeat
   generation in a fresh process for exact byte equality. Compile/vet the fixture.
   Run all five mixed HTTP/SSE tests under race: final responses, endpoint errors,
   envelope errors, request-size rejection and cross-origin rejection.
4. Independently review the exact fixture/plan diff, commit/push atomically and
   close the report. No new model or runtime semantics are introduced; existing
   transport behavior and generation parity remain the acceptance contract.

Issue [#841](https://github.com/CaliLuke/loom/issues/841) was published after
independent editorial review. Parent focused reproduction at pushed0d9671589
fails in7.951s at the exact tree comparison; CI shows the older server differences.

Regenerated with `go run -mod=mod github.com/CaliLuke/loom/cmd/loom gen
example.com/mixedtick/design -o .` from the fixture directory. All16 generated
artifacts were compared; exactly four change:

- client/cli.go receives the delivered detached CLI input declarations and
  presence-aware conversion from #838.
- server/encode_decode.go moves both decoder error locals into outer scope.
- server/server.go delegates the stream error through SendError.
- server/stream.go threads context into mappedErrorResponse.

The other12 files, including the generation manifest, are byte-identical.
A fresh-process repeat matches all16 files exactly. `go mod tidy` removes the
unused Jennifer dependency and its two sums; no dependency is added or updated.
The fixture builds/vets, and all five mixed transport tests pass under race.
The final race run was performed after regeneration and module tidy completed;
no fixture mutation overlapped that run. The existing copy-local-source and
full-tree parity checks remain unchanged. No new test/model or production code
is needed for this generated-artifact correction.

Comparison receipt: `/tmp/loom-mixed-fixture-comparison.json`; parent/current logs
are `/tmp/loom-mixed-fixture-parent-red.log` and `/tmp/loom-841-final-race.log`.
Seven-file fixture/module/plan diff awaits independent final review and one atomic
commit. Preserve existing stream contract models with their owning implementation.
