# OpenAPI acquisition checkpoints

The previous checkpoint cloned every accumulated registration map, reference
bucket and annotation-owner map before each outer schema scope. A consuming
design profile attributed about 29% of sampled allocated bytes to this owner.
`TestAcquisitionCheckpointDoesNotCopyPublishedIndexes` reproduced 57 versus
4,133 allocations with 16 versus 2,048 prior entries and no scope mutations.
The changed-entry journal requires one allocation at either size.

`AcquisitionJournal.tla` models one outer scope with up to three ordered index
writes, commit, and abort. `kept` is a prior entry; zero abstracts an absent
entry. Dense charges the prior index size at checkpoint creation. Journal
records each immediately preceding value and undoes changes in reverse order.
Forward is a rejected alternative: repeated writes to the same entry do not
restore its original value when undone in insertion order. Commit and abort
release captured history. The bounded work invariant counts entry recording
and restoration, independent of prior index cardinality.

Use TLC 2.19, `tla2tools-1.7.4.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
These opt-in runs used Homebrew Java 26.0.1 and one worker:

```sh
java -Xmx256m -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Dense.cfg AcquisitionJournal.tla
java -Xmx256m -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Journal.cfg AcquisitionJournal.tla
java -Xmx256m -cp "$TLA_JAR" tlc2.TLC -workers 1 -config Forward.cfg AcquisitionJournal.tla
```

Dense fails `BoundedWork` at checkpoint creation, 2 distinct states, depth 2.
Journal passes all three invariants, 778 generated/288 distinct states, depth 6.
Forward fails `Restored`, 59 generated/53 distinct states, depth 5. The negative
controls must fail on these invariants, not a parser or tool error. These are
design checks, never build, hook, CI or release gates.

`acquisition_scope.go` owns the journal and its set/delete/lazy-allocation
operations. `schemaAnalysisScope` holds it through nested traversal, projection
completion and canonical comparison, then relinquishes it. Registration writers
use that owner, including nested annotation and mapping indexes. A changed
reference bucket is detached before element replacement or deletion; scoped
appends also detach from prior spare capacity so rolled-back bindings cannot
remain retained beyond a restored slice length. Prefix evaluation still precedes
reentrant contributor callbacks. Lazy index rollback restores nil rather than an empty
allocated map. The prior full-snapshot path is removed.

The model abstracts values as immutable scalars, not Go maps, slices, nil map
headers, callbacks, nested analyzer depth, mutable schema cells or error causes.
It proves neither schema equivalence nor a total generator memory bound.
`TestAcquisitionJournalScopeOutcomes` covers repeated writes/delete, nested
scopes, annotation owners, reference promotion, lazy nil indexes, success,
ordinary failure, panic preservation and journal release. Existing acquisition
completion/protocol tests cover actual contributor bindings, recursive graphs,
canonical conflicts and body-error cleanup. The
[projected acquisition model](../projected_acquisition/README.md) still owns
held completion ordering and publication; this change preserves that protocol.

## Appended backing storage

`AppendBacking.tla` refines the immutable-value assumption with one two-slot
backing buffer and a prior logical slice of length one. Shared appends write
into the prior spare slot. Restoring only the length leaves that failed binding
reachable from the prior buffer. Detached appends preserve its unused capacity
as well as its prefix. This models reachability, not GC timing or actual Go
allocation sizes; it does not model reentrant callback evaluation.

Use the same pinned tool command with `AppendBacking.tla` and `Shared.cfg` or
`Detached.cfg`. Shared fails `NoRetainedBinding` in 3 distinct states, depth 3.
Detached passes both invariants in 4 distinct states, depth 3. The direct scope
outcome test failed for error and panic before append detachment, even though the
restored logical reference bucket already matched. It now also checks the prior
backing storage, including unused capacity.
