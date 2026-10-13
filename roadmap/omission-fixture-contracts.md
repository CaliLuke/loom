# Omission fixture contracts — #861

Parent `58defc6c7`. Correct only stale acceptance fixtures after the delivered
physical-layout JSON omission contract. Production code and DSL inputs stay
unchanged; explicitly supplied scalar zero remains supplied.

1. [x] Reproduce optional-union anonymous tag compilation and OpenAPI TestFiles
   golden failures on main; independently review and file #861.
2. [x] Capture current rendered module/spec bytes before changes. Use temporary
   test overlays only to retain outputs outside TempDir; no generator changes.
   Existing TestFiles table costs a few seconds and validates rendered specs;
   this is not the exported-design compile corpus.
3. [x] Change the three stale anonymous harness tags to match actual declarations.
   Regenerate affected stored goldens using TestFiles -update. Inspect every
   difference; allow only restored supplied scalar-zero examples.
4. [x] Run the original optional-union vet/runtime harness and TestFiles checks;
   retain their assertions. Compare parent/candidate and independent-process
   repeat outputs exactly. Run representative pinned Vacuum contract lint.
5. [ ] Run focused lint and diff checks, freeze final evidence, obtain fresh
   exact-diff review, and deliver one atomic commit before closing #861.

Expected production output difference: none. Golden files intentionally change;
the renderer output they describe does not. Keep #862 identity comparisons and
WebSocket completion observation separate.

Validation complete before review: parent/candidate/independent-repeat captures
contain 165 artifacts each, identical byte for byte. The original optional-union
runtime/vet harness and all TestFiles parser checks pass. Only three harness tags
and 18 stored JSON/YAML goldens change; JSON comparison finds 55 additions of
supplied empty-string examples and no removed/changed values. All 18 changed
rendered specs pass pinned Vacuum using the existing install/lint owner, together
with its five invalid-contract controls. Scoped lint reports zero issues.
Temporary overlays retain outputs and reuse test helpers; none are committed.
Review packet: `/tmp/loom-861-final-manifest.json`.
