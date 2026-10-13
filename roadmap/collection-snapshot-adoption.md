# Collection snapshot adoption — #896

The shared collection conversion owner delivered in #889 preserves nil sources
and allocates nonnil empty targets. Three service result/view snapshots still
expect unconditional allocation. This task completes fixture adoption without
changing production behavior, assertions, or the accepted presence contract.

## Sequence and acceptance

- [x] Reproduce the three exact golden failures on unchanged `2a98e4123` and
  current source. Retain `/tmp/loom-770-parent-stale-goldens.log` and
  `/tmp/loom-collection-snapshot-parent.log`.
- [x] Regenerate only `TestService` collection-multiple-views,
  with-result-collection and with-dashed-mime-type snapshots. Every difference
  must move allocation and element conversion under a nonnil-source guard,
  or explicitly clear an existing target for an absent source. Element conversion
  and nil-element handling remain unchanged.
- [x] Run those exact comparisons and existing root/nested/named collection
  runtime tests. No generator implementation changes or new harness.
- [ ] Obtain independent review of the exact three snapshots plus this record,
  commit and push one atomic adoption change, and close #896 with evidence.

Existing #889 compiled conversion and mounted wire tests own nil/empty semantics.
This change only updates saved expectations; it does not invalidate their
implementation evidence or require another full design-corpus run.

Validation: three regenerated comparisons pass under race (1.579s); existing
root/nested/named collection runtime controls pass (12.625s). The first combined
selection ran no service cases; the separate exact service command supplies that
coverage. Final diff has twelve guarded root conversions and six explicit
existing-field clears. No production source or assertion changes.
