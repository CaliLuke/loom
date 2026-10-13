# Telemetry construction ownership — #763

All enabled providers must be constructed before New publishes process globals.
A failed construction must retain the existing globals and shut down acquired
providers in reverse order, preserving the primary error and cleanup errors.
Successful Runtime.Shutdown must consume the same shutdown owner. Existing
successful provider overrides and disabled sections retain their contract.

1. Reproduce trace→metric and trace/metric→log failures using real invalid
   exporter endpoints; check every process-global provider/propagator remains
   unchanged, restoring globals and shutting down escaped parent providers in
   the test even when the regression fails.
2. Model staged acquisition/publication/rollback, including disabled stages,
   failures at each stage and reverse cleanup. Current eager publication and
   missing rollback must fail; staged publication and cleanup must pass.
3. Propose one shutdown-stack function used by rollback and returned Shutdown;
   constructors register only successfully acquired resources. Delay all global
   setters until successful completion. Rollback uses an uncanceled context with
   a bounded cleanup timeout; report cleanup errors alongside the cause. No
   global restoration after publication or transport-specific cleanup paths.
4. Independently review readiness before production. Add direct shutdown-stack
   order/error/cancellation controls with the implementation seam, then rerun
   existing bootstrap/provider overrides and affected observability race suites.
5. Record callback termination assumptions, actual supported SDK behavior and
   public failed-construction behavior. Freeze exact diff for independent final
   review, atomic commit/push and ticket closure.

No new public configuration is proposed. An internal rollback timeout bounds a
failed constructor's cleanup; it does not promise recovery from non-cooperating
third-party shutdown functions. Successful Shutdown still uses its caller's
context. A successful publication calls the existing SDK global setters; this
is not an atomic multi-global snapshot or a new concurrent bootstrap API.

Readiness evidence: actual metric/log endpoint failures reproduce changed trace
globals and propagators; log failure also replaces metric globals. The pinned
trace exporter accepts the same endpoint on construction, so that attempted
negative is excluded with no claim about trace-constructor failure. Revised
parent regressions use independent assertions for all globals. Models distinguish
publication-only repair from complete rollback and detect forward cleanup.
Checked design passes169 states; parent/publication-only/wrong-order fail81/81/156.
Production was unchanged at independent readiness review.

Independent readiness review approved the exact manifest and repeated both
actual failed-bootstrap cases, successful controls and all four model outcomes.
Implementation now registers all acquired providers before publishing globals.
Rollback and returned Shutdown use the same reverse cleanup function; rollback
gets a five-second uncanceled context and retains construction/cleanup error
identities. No factory injection layer or new public configuration was added.

Final bounded acceptance: full runtime/logrusbridge/HTTP/gRPC race suites pass
1.564s/1.363s/1.343s/1.351s. Direct seam controls cover empty/one/three callbacks,
reverse order despite all errors, unchanged canceled caller context on normal
shutdown, and fresh rollback deadlines/context values with canceled or live
startup contexts. Independent final review remains required before commit.
