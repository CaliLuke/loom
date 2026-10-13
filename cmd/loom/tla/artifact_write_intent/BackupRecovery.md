# Recoverable-original backup lifetime (#899)

This bounded extension addresses the restoration-failure assumption omitted by
`ArtifactWriteIntent.tla`. That earlier model still covers create-only admission
and publication; its successful-filesystem assumption does not establish recovery
when a restore fails. No create-only or destination placement policy changes.

`BackupRecovery.tla` represents one gen-tree original and one external original.
Preserve moves an original from its live name into a protected backup. Failure may
occur before or after either move. Each rollback attempt independently succeeds
or fails; cleanup can repeat. Successful complete installation permits deleting
all backups. Failed installation requires each original to remain at its live
name or at a reported backup. The model abstracts actual paths to the two owners.

Run with the existing pinned TLA+ tools 1.7.4 (TLC 2.19):

```sh
MODEL_TMP=$(mktemp -d)
for mode in current external-only owned; do
  java -XX:+UseParallelGC -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar \
    tlc2.TLC -config "BackupRecovery-$mode.cfg" \
    -metadir "$MODEL_TMP/$mode" BackupRecovery
done
```

| Configuration | Result | Generated / distinct states |
| --- | --- | --- |
| current | `RecoverableOriginals` counterexample after cleanup | 36 / 29 |
| external-only | Same invariant counterexample for gen backup | 36 / 29 |
| owned | All five invariants pass | 55 / 40 |

The initial scratch model needed parentheses around conditional assignment
expressions; its checker exceptions are not counted as counterexamples. The
results above are from the corrected source and actual invariant checks.

Implementation correspondence:

- Preserve: moving gen or an external destination aside creates one protected
  transaction backup record.
- Commit: only installation of the tree and all outputs authorizes discard.
- Restore: successful rename releases the backup; failure retains its protected
  state and recovery location.
- Cleanup: stage disposal is independent; unresolved protected originals survive
  and are reported, while committed backups may be discarded.

`TestGenerationRecoveryKeepsOriginals` reproduces both actual data-loss paths and
checks repeated cleanup plus recovery diagnostics. The gen case prepares external
candidates, moves the old tree aside and installs the staged tree through the existing transaction seams. Removing a prepared
candidate then fails external installation; an unreadable child directory blocks
tree restoration. The external case uses
a nonempty destination directory. `TestGenerationRecoveryNormalCompletion` and
existing transaction/create-only controls check successful cleanup and ownership.

Limits: the model assumes original bytes remain in the recorded backup until an
owned operation moves or deletes them. It does not prove filesystem atomicity,
path aliasing, process-crash recovery, arbitrary ancestor mutation, permission
behavior under privileged accounts, byte preservation or error formatting.
Tests check native paths/contents/errors. Unending filesystem failures imply no
unconditional eventual-cleanup guarantee. Failed deletion after confirmed commit
and aggregation of several errors are implementation-test obligations.

Implemented owner: `generationTransaction.preserveOriginal` creates the protected
record; `generationBackup.restore` releases only a confirmed restore;
`generationTransaction.discardOriginals` authorizes disposal only after all new
outputs install. `cleanupBackups` joins every record's outcome without releasing
failed discards. The ordinary deferred cleanup reports protected records without
touching their bytes. Old external backup lists and the gen escape branch are
removed. `externalCandidates` remains the independent new-byte owner.

`TestGenerationRecoveryBeforeInstall` covers the early install-failure seam and
recovery after a blocked restore. `TestGenerationRecoveryMixedRestore` checks
that two blocked external originals remain while another external output and
gen restore successfully, with all errors retained.
`TestGenerationCommittedBackupRemovalRetries` checks file/tree discard failure
and subsequent cleanup after permissions recover. These tests cover native
error and retry details outside the finite model.
