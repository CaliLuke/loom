# Adopt complete transform plans in the bytes-default fixture (#940)

The CI and local generated-module failure on d5c2f5f9f is a fixture integration
error: TestBytesDefaultTransformLayouts takes TransformPlan.Code and Helpers but
ignores ErrorAware. Custom String defaults now parse through the shared runtime,
so the generated module needs its Loom import and an observable error channel.
The shared production transform/default owner already supplies the correct plan.

1. Preserve the existing failing test and its 448 source/target layout cases.
2. Generate a transformErr channel only when the returned plan requires it;
   return errors from every generated conversion through the fixture callback.
3. Assert unexpected errors before inspecting results. Add a custom parser that
   rejects one default, with the same 64-layout matrix proving rejection occurs
   only on actual absence where default application is enabled. Explicit empty
   and supplied values remain supplied; native bytes/raw JSON behavior stays.
4. Run the focused generated-module matrix, existing storage metadata check and
   targeted lint; obtain independent exact-diff review before one atomic commit.

This changes only test adaptation, with no production/generated application
output or public policy change. No new formal model is needed: the fixture
consumes the existing TransformPlan error contract directly.

Acceptance completed on Go 1.27.2 with GOEXPERIMENT=jsonv2: the generated module
compiles, its 512 layout cases preserve the original matrix plus 64 rejected
custom-default layouts, and go vet passes. Focused outer race run passes 3.471s;
targeted codegen lint reports zero issues. A deliberate dropped-error mutation
fails the three materializing layouts (34,38,39), proving parser errors are
observed. The original main failure reports undefined transformErr and loom.
No production source or generated application output changes. Independent
review remains the final gate before delivery.
