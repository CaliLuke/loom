# Interceptor fixture responsibilities — #888

Required CI at `7ed1bfc63` fails test-source lint on
`InterceptorContractStorageDSL` (maintidx 31). The same configured linter failure
reproduces on local parent `60da15241`; this is fixture construction complexity,
not a production compilation or interceptor behavior failure.

1. [x] Preserve the exact CI and local lint failure.
2. [x] Extract logical type selection and deferred field construction into
   package-private fixture helpers. Keep every condition, DSL call, attachment
   and callback evaluation boundary unchanged; no lint suppression.
3. [x] Configured lint for `codegen/service/testdata` now reports zero issues.
   Existing interceptor contract/signature/storage tests pass under race (1.927s).
   Required and union generated harnesses compile and pass (6.029s).
4. [ ] Independent exact diff review, atomic commit, push and confirmed closure.

The extraction changes one test fixture file, with no production generator or
runtime edits. Prior generator acceptance remains valid; no corpus or formal
model is required for this mechanical fixture split. Existing generated harnesses
exercise the two native storage boundaries and the direct tests retain rejection
of incompatible logical signatures. Compact evidence is recorded in
`/tmp/loom-888-final-manifest.json`.

CI: https://github.com/CaliLuke/loom/actions/runs/38455349199/job/115496535377
