# Immutable release source

Run with TLA+ tools 1.7.4 and Java:

```
java -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -metadir /tmp/loom-publication-state -config internal/release/tla/Publication.cfg internal/release/tla/Publication.tla
```

`Unsafe.cfg` deliberately tags the moving main tip after checking the selected
source. TLC reproduces a counterexample: select commit 1, advance main to 2,
check commit 1, tag commit 2. `Publication.cfg` fixes tag ownership to the
selected source and checks TestedSource, PinnedSource and Complete.

The safe bounded model explores 14 distinct states. It assumes one publisher
(the workflow concurrency group), immutable commits, and authoritative CI
results. It allows main to advance independently and publication to repeat;
a repeated publish does not change the tag. The implementation seams are
`publisher.publish`, `checkCI`, and `chooseAlpha`. Go tests cover version
allocation, no-change retries, CI job completeness, conflicting refs and
promotion. The model does not prove GitHub API availability, token security,
CI test coverage, version parsing or the behavior of external administrators.
`TestPublicationValidatesEffectivePushDestinations` separately exercises real
Git URL resolution, including multiple push URLs and URL rewrites, and checks
that noncanonical destinations are rejected before publication starts.

No source commit is created by publication. Promotion uses the same selected
source as its alpha and therefore has the same source-identity invariant.
