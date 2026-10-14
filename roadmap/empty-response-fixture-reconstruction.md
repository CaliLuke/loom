# Empty-response fixture reconstruction — #913

These fixtures test empty response bodies and tagged status selection, but their
required result `h` has no source in the ordinary/untagged response. Required
reconstruction admission correctly rejects that contradiction. Preserve required
values and bodyless responses: map `h` to `X-Result-H` on the ordinary 200 and
untagged 204 branches. The tagged 304 branch already reconstructs `h` from its
Tag binding and needs no redundant mapping. No production contract changes.

1. [x] Preserve current encoder/decoder admission failures.
2. [x] Add the two explicit header mappings without changing requiredness or tags.
3. [x] Regenerate the four affected golden expectations through their owning tests.
4. [x] Run affected encoder/decoder and bounded-body controls; generate the two
   existing CLI specimens, build/vet, and compare a separate generation exactly.
   Preserve the parent rejection and inspect every intended golden change.
5. [ ] Freeze one atomic fixture-adoption diff for independent review and delivery.

Existing required-reconstruction admission tests retain rejected-design coverage.
The separate importer mandatory-cycle diagnostic mismatch is outside this task.

## Evidence and intended differences

Parent `16749ea5b` rejects both designs before generation: four owning golden
cases and two bounded-body decoder controls fail (0.388 s). After fixture repair,
these checks pass under race (1.963 s). The four regenerated golden differences
are limited to writing/reading required `X-Result-H`, validating a missing header,
and supplying the resulting value to the correct response constructor. The 304
branch still reconstructs `h = "true"` from its tag; empty body handling, bounded
unexpected-body reads, and restoration behavior remain covered unchanged.

The existing exported-design CLI harness runs only the two repaired designs and
`ResultHeaderStringDSL` as an unchanged control. An immutable parent archive
retains both expected admission failures; the control builds and vets. An
immutable candidate with only the two fixture edits generates all three modules
and builds/vets them (39.688 s). A separate generation compares exact bytes;
identical output is not rebuilt unnecessarily. No generated file is hand-edited,
no production source or test assertion is changed, and required-field admission
remains intact.

The unchanged control retains all 21 artifacts byte-for-byte. All 63 candidate
artifacts match the independent repeat (10.695 s); scoped fixture lint reports
zero issues. The exact patch is frozen for independent review; delivery is pending.
