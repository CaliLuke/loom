# Generation backup recovery — #899

Status: implementation complete; final independent review and delivery pending.
Filed [#899](https://github.com/CaliLuke/loom/issues/899). Implemented on
`2537b97acefb7be5e39312b524a61f5a6d371e9c`, after #751 destination publication.
Original inspection was on `271784d3c06bc68b6f6a95dcb36969ce76c4fede`.

## Confirmed failure

The parent `generation_transaction.go` kept gen backups in `t.backup` and external backups
in both `output.backup` and `t.externalBackups`. Rollback keeps those paths after
failed restoration, but deferred `cleanup` treats all of them as disposable.
Consequently the only original bytes disappear after an already-failed rollback.

Two actual filesystem probes fail on current source (0.595s):

- An external original is preserved, a generated file is installed, and another
  owner replaces the destination with a nonempty directory. Restore fails; the
  original backup remains immediately afterward, then cleanup deletes it.
- The original gen tree is moved aside. A generated tree contains an unreadable
  directory entry. Rollback cannot remove it or rename the old tree back; the
  old bytes remain in the sibling backup, then cleanup deletes that backup.

Inputs: `/tmp/loom-751-backup-loss_test.go` and
`/tmp/loom-recovery-backup-probe_test.go`; exact overlay and log are
`/tmp/loom-recovery-backup-probe-overlay.json` and
`/tmp/loom-recovery-backup-parent.log`. This is direct in-process Go test overlay
loading; no CLI/helper subprocess drops the overlay. No checkout file changed.
The tree probe uses current-user filesystem permissions and is not a claim
about permission behavior under a privileged test account.

The parent failure branch immediately after installing staged gen set
`t.backup = ""` when restoration fails. That avoids deletion accidentally by
forgetting cleanup ownership. Its OS rename error includes both paths, but this
special branch is not a shared or complete lifetime contract.

## Smallest owner correction

Use one transaction-owned backup record for the gen tree and external outputs.
Each record contains backup path, original destination and removal kind (file or
tree); the transaction owns its disposition. The gen and external installation
records may reference it, but must not maintain independent deletion lists.

A successful move-aside creates a **protected** record. It stays protected until
one of two confirmed transitions:

1. Successful restore releases the record because the original has returned to
   its destination.
2. Installation of the gen tree and every external output succeeds, establishing
   transaction commit. Only then do remaining records become **discardable**.

Cleanup may remove staging normally and discard committed backups. It must never
delete a protected record, including one whose rollback failed or was not reached.
A failed restore leaves its record protected and joins the original filesystem
error with explicit recovery information naming both preserved backup and target.
Cleanup must also report any unresolved protected record so interrupted error
paths cannot silently abandon recovery information. Repeated cleanup is safe;
discard deletion failures remain retryable rather than being forgotten.

Keep restoration and discard operations on that record, with directory/file
removal chosen once there. Delete `externalBackups`, `forgetExternalBackup` and
the manual early-branch ownership escape. Replace the separate gen/external
backup deletion loops with one commit/disposition path. This is a small private
transaction abstraction, not a filesystem interface, background recovery service,
new generic transaction framework or automatic retry protocol.

All rollback attempts still run and their errors are joined. Preserve current
create-only SameFile behavior and its separate concurrent replacement owner.
Do not strengthen arbitrary adversarial ancestor-swap or process-crash claims.
The contract protects originals that remain at the recorded backup location;
it cannot protect a backup removed or relocated by another process.

## Bounded model and tests before implementation

Extend the existing `cmd/loom/tla/artifact_write_intent` ownership model family
with the missing restoration-failure boundary. The current model assumes every
filesystem operation succeeds except the second installation and therefore
cannot establish this property. A bounded gen + external backup model needs
only move-aside, complete/failed installation, successful/failed restore,
commit authorization and repeated cleanup. No general filesystem model.

First reproduce unconditional cleanup deleting a failed-restore original. A
partial alternative protecting external backups alone must still fail for gen.
The corrected model requires: before confirmed commit, each original is at its
live destination or a protected backup; cleanup never deletes a protected
backup; terminal unresolved records have recovery reporting; successful complete
installation allows discard; successful restoration leaves no recovery debt.
Operations can fail indefinitely, so no unconditional eventual cleanup promise.

Implementation tests reuse the existing transaction/fake-generator seams:

- Preserve both exact native failure probes, then assert backup bytes survive
  repeated cleanup and returned errors name backup plus original destination.
- Cover failed restoration before staged-gen installation completes (the current
  special branch), plus failures after external installation begins.
- Mixed rollback: one restore succeeds and another fails; successful originals
  return, failed originals remain, all errors remain visible.
- Existing successful commit and rollback controls leave no backup debt; no-live
  gen and skipped create-only entries never gain recovery ownership.
- Retain `TestArtifactWriteIntentRollbackPreservesReplacementOwner` and existing
  transaction/plugin-output tests. Assert original file/tree contents, not just
  error text. Native permission test requires a real denial; a non-denial must
  be an explicit unsupported test condition, never a false successful proof.

No codegen output, public DSL or artifact bytes change. Direct transaction tests,
focused command-package race controls and the bounded model suffice; no generator
corpus, fixture regeneration or unrelated cleanup rewrite is justified.

Exit: all paths that preserve an original use the same backup lifetime record;
failed rollback plus automatic cleanup preserves those bytes and reports a
usable recovery location; ordinary successful commit/rollback still cleans up.

## Ordered implementation and readiness receipt

- [x] Reproduce both failed-restore losses and inspect all backup/deferred cleanup
  paths; coordinate separate candidate-publication work in #751.
- [x] Add native current-state failing tests and successful commit/rollback controls.
- [x] Model unconditional cleanup and external-only repair counterexamples, then
  verify shared protected-original disposition.
- [x] Independently review exact readiness before production changes.
- [x] Implement one private backup record and delete separate gen/external lifetime
  paths, preserving candidate ownership from #751.
- [x] Add mixed restore and early-error correspondence controls; verify exact bytes,
  joined failures, recovery destinations and repeat cleanup under native tests.
- [ ] Independently review the final exact diff, deliver one atomic commit, then
  preserve compact evidence and remove inactive checker/test output.

Scratch readiness is under `/tmp/loom-backup-recovery-readiness`. The two native
failure cases fail on current main (0.679s), while normal commit/rollback controls
remain green. The first draft's gen permissions blocked installation itself;
that invalid fixture was corrected to protect a child directory, so the final
witness reaches failed rollback and then demonstrates cleanup data loss.
Both current and external-only models violate `RecoverableOriginals` (36/29
states); the shared owner passes all invariants (55/40). Exact commands, logs and
source hashes are frozen in `/tmp/loom-899-readiness-manifest.json`.

The exact corrected readiness files were transferred from scratch after the
managed worktree advanced cleanly to the delivered #751 commit. Root and other
agents' candidates were not edited; checker outputs and raw logs stay outside
the repository.

## Readiness correction R1: publication phase correspondence

Independent review found that #751 now rejects a missing staged external file
before gen publication. The original tree/normal rollback tests therefore no
longer reached the intended phase under that accepted prerequisite. Revised
tests successfully prepare every external candidate first, explicitly move the
old gen tree aside and install the staged tree at existing transaction seams,
then remove a prepared candidate before installation. The normal rollback case
also asserts its first external output installed before the second fails.
The helper checks the original backup and live generated tree already exist;
there is no new filesystem interface, callback hook or timing race.

The unreadable-child cleanup now covers both its staged and live locations.
Against the exact frozen #751 production overlay, both losses again reproduce
after actual failed restoration (0.672s), and normal commit/rollback plus the
create-only replacement-owner control pass under race (1.960s). The original parent
receipts are retained; old missing-staged controls are not claimed to establish
#751 rollback. The ownership/model sources and their prior checks are unchanged.

The final composition checks run directly in the frozen #751 worktree with only
the readiness test overlaid. An intermediate check against root coincided with
#751 integration and failed test discovery/compilation; that attempt is retained
and superseded by the fixed-source runs, not counted as behavior evidence.

## Final implementation and acceptance

`generationBackup` is the shared original-byte lifetime owner. Both gen and
external publication call `preserveOriginal`; rollback calls the record's restore
operation; only complete installation calls `discardOriginals`. Deferred cleanup
reports every protected original without deleting it and retains failed discard
responsibility for another cleanup attempt. Separate backup lists, deletion loops
and the early gen ownership escape are gone. New disposable candidates and
create-only concurrent replacement identity remain unchanged.

The native tests now cover both loss witnesses, early installation failure, mixed
successful and failed restoration, repeated cleanup, joined errors and recovery
paths, normal commit/rollback, and file/tree discard failures followed by cleanup
retry. Permission-specific cases explicitly skip privileged or non-POSIX hosts;
they pass against real permission denials in this validation environment.

The focused command-package race selection passes (2.059s), including the
existing preparation-failure cleanup control, and golangci-lint 2.14.0 reports
zero issues for `./cmd/loom/...`, on Go 1.27.2 with JSON v2 enabled. The approved model source/configurations are
unchanged, so their current/external-only counterexamples and owned invariant
results remain applicable. No generated artifact contents or generator analysis
change: no fixture regeneration or generator matrix is needed for this owner.

Final source/evidence hashes, commands and results are in
`/tmp/loom-899-final-manifest.json`; readiness approval is
`/tmp/loom-899-independent-readiness-rereview-verdict.json`. Independent final
review must authorize the exact diff before one atomic delivery.
