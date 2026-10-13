# Parent-aware trace sampling — #748

At the parent revision, the trace-provider owner applied the root ratio to every
span, breaking propagated sampling decisions. The implementation now uses the
SDK ParentBased policy around the existing root ratio; local and remote valid parents retain their sampled flag.
Keep existing ratio normalization and new-root behavior. No dependency change.

1. Read provider/runtime/config and pinned SDK sampling contracts. Reproduce
   sampled-parent loss and unsampled-parent sampling using fixed trace IDs.
2. Add direct real-provider controls for local/remote parents, invalid/absent
   parents, explicit new roots and retained root ratio normalization. Add an
   HTTP traceparent propagation consumer check using runtime-local propagators.
3. Model root-policy versus parent-policy selection and retain the counterexample
   before implementation. Model assumes a correctly extracted valid parent;
   actual propagation is checked by the runtime test.
4. Obtain independent readiness review, then change the shared provider owner.
   Run affected runtime/HTTP/gRPC observability race suites and scoped lint.
5. Update public SampleRatio guidance and model correspondence. Freeze exact diff
   for independent review, then atomic commit/push and ticket closure.

Historical readiness evidence: direct local/remote parent mismatches and both actual HTTP propagated
cases fail at unchanged parent ee4552dc9 (0.619s). Existing root routing and ratio
normalization controls pass separately. Pinned SDK v1.47.0 explicitly recommends
ParentBased composition; no transport-local override or independent ratio
algorithm is proposed. Current model fails ParentDecision at50 states; checked
passes64. Fixed IDs avoid probabilistic tests. Production was unchanged at readiness review.

Independent readiness review approved all six sources and reproduced parent
race failures/root controls and both model outcomes. Implementation now composes
the SDK samplers at the single provider owner. Public configuration docs state
root-only ratio and retained normalization; external provider policy is unchanged.

Final validation: full observability/otel runtime and logrus bridge plus HTTP and
gRPC middleware race suites pass (1.729s/1.459s/1.345s/1.414s). Scoped pinned
lint reports zero issues and diffcheck passes. No generated artifact changes;
existing runtime and middleware suites exercise the shared configured provider.
Exact final source/patch manifest is /tmp/loom-748-final-manifest.json. Fresh
independent final review precedes the single atomic commit.
