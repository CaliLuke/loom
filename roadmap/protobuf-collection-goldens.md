# Protobuf collection golden alignment — #839

Parent: pushed main `2a8337700`. Production semantics were corrected by #684;
only four checked-in expectations still require nonnil collection contents.

1. Retain current-parent failures for the exact four affected cases. Read their
   generated diffs and the existing collection-wrapper contract regressions.
2. Regenerate these four goldens through the owning test helper. Remove only
   obsolete contents-required checks and validators/calls with no remaining
   predicates; retain wrapper presence and authored length validation.
3. Run those comparisons and the existing direct collection-wrapper contract
   suite. Production bytes are unchanged, so reuse #684 generated/runtime
   evidence rather than run another generator corpus.
4. Freeze the exact five-file diff for independent review. Integrate and push
   one atomic ticket commit; close #839 and preserve compact evidence.

The four goldens regenerate with 110 obsolete lines removed: three predicate-only
validators disappear from each nested-type snapshot, the map decoder loses its
empty validator call, and the constrained element retains its minimum-length
check while losing contents-requiredness. No production source changes.

Exact four parent failures retained at /tmp/loom-839-parent-red.log; regenerated
comparisons pass at /tmp/loom-839-candidate.log. Direct TestCollectionWrapper*
controls pass at /tmp/loom-839-owner-controls.log, covering containing-message
presence, collection contents and scalar/union requiredness. No generator
re-run comparison is needed: this commit only changes checked-in expected text.
