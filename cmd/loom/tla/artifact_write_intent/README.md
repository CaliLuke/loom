# Artifact write intent

The bounded model covers the create-only external artifact contract of #670.
Staging preserves content, but the former path-only result erased ownership
intent. An existing live destination was then replaced. A check followed by an
unconditional rename also loses a file created between those two operations.

Run with TLA+ tools release 1.7.4 (TLC 2.19), using the shared pinned tool at
`/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar`, from this directory:

```sh
MODEL_TMP=$(mktemp -d)
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -config Current.cfg -metadir "$MODEL_TMP/current" ArtifactWriteIntent
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -config CheckThenRename.cfg -metadir "$MODEL_TMP/check" ArtifactWriteIntent
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -config Corrected.cfg -metadir "$MODEL_TMP/corrected" ArtifactWriteIntent
```

`Current.cfg` and `CheckThenRename.cfg` must fail `PreserveUserContent`.
`Corrected.cfg` must pass all invariants. A parse/tool failure does not count as
a counterexample. Task-owned temporary state and logs are removed after review;
keep the model/configuration sources and these findings.

Observed before implementation with parent `eb5a169b`:

| Configuration | Result | Generated/distinct states |
| --- | --- | --- |
| Current | `PreserveUserContent` violated: pre-existing user bytes replaced | 9 / 7 |
| CheckThenRename | `PreserveUserContent` violated: user creates after check | 14 / 12 |
| Corrected | All three invariants hold | 18 / 16 |

## Scope and assumptions

One create-only external path has three abstract contents: absent, user, and
generated. The user can create a file before installation. After installation,
a second output either succeeds or fails; failure requires rollback of the new
entry and restoration of the previous `gen/` tree. Filesystem operations are
atomic and successful except the modeled second-output failure. The successful
create operation acquires rollback ownership; a skipped entry does not.

This is bounded checking, not a Go refinement proof, crash-recovery guarantee,
or proof for arbitrary filesystem mount layout. Whole-tree policy for
create-only entries inside `gen/` is outside this model. Concurrent modifications
after publication, ancestor-directory swaps, permission errors, byte formats,
and helper serialization are implementation-test obligations. The implementation
checks filesystem identity before rolling back a create-only file whose name
may have been replaced; the model does not claim that identity check and remove
are atomic against an adversarial process.

## Implementation correspondence

| Model action/property | Implementation owner | Regression |
| --- | --- | --- |
| Stage preserves declared policy | `codegen.Artifact`, `generator.writeFiles` and `computeOutputs`; JSON helper result | `TestArtifactWriteIntentSurvivesRendering`, `TestArtifactWriteIntentRealPlugin` |
| Current intent loss | Former path-only `Generate` and transaction result | `TestArtifactWriteIntentPreservesExisting` failed with `template` instead of `user` before repair |
| Check/creator/install race | `File.Render` exclusive open and `installExternalOutputs` no-replace link | `TestFileSkipExistPreservesRacingCreator`, `TestArtifactWriteIntentPreservesCreatorAfterPreparation` |
| SkipNeverOwned | Link existence result acquires no backup or installed flag | `TestArtifactWriteIntentLiveEntries` and creator-after-preparation test |
| Failure/rollback | `rollbackInstalledOutputs` removes only installed entries and restores previous output | `TestArtifactWriteIntentRollback`, existing replaceable-output rollback test |
| Unmodeled replacement owner | Same-file identity check before create-only rollback | `TestArtifactWriteIntentRollbackPreservesReplacementOwner` |

The real plugin test loads a registered generator plugin through the compiled
launcher and its separately compiled helper; it checks fresh creation, modified
live content, skipped public stdout, and compilation of the generated module.
Standalone renderer tests cover directories and dangling/live symlinks. Conflicting
policies for a normalized destination are rejected before rendering in both
orders; equal policies still coalesce sections. Those concrete cases are outside
the finite contents abstraction above.

## Accepted gen-tree admission policy

The user explicitly chose to reject create-only files inside `gen/`, whose
whole-tree replacement contract conflicts with retaining user ownership there.
`GenPolicy.tla` checks this admission choice separately from the external-file
installation model. It ranges over internal/external and create-only/replaceable
artifacts. In the original model, `GenPolicyCurrent.cfg` reproduced accepting
the contradictory internal/create-only case and violated `RejectContradiction` (8 generated,
8 distinct states). `GenPolicyCorrected.cfg` passed `RejectContradiction` and
`AdmitNormalOutputs` (8 generated, 8 distinct states).

```sh
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -config GenPolicyCurrent.cfg -metadir "$MODEL_TMP/gen-current" GenPolicy
java -cp /Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar tlc2.TLC -config GenPolicyCorrected.cfg -metadir "$MODEL_TMP/gen-corrected" GenPolicy
```

`internal/artifactpolicy.Validate` is the production owner. Standalone rendering,
generator preflight and staged launcher validation share it. Regressions are
`TestFileSkipExistRejectsGenerationTree`,
`TestArtifactWriteIntentRejectsCreateOnlyGenerationTree`,
`TestArtifactWriteIntentRejectsGenerationTreeBeforeCommit`, and the rejected
registered-plugin case in `TestArtifactWriteIntentRealPlugin`. The static model
abstracts path normalization to a Boolean; Go tests check concrete paths.


## Review correction: live aliases

The first integrated review found a P1 counterexample outside the initial path
extraction assumption. With live `alias -> gen`, rendering `alias/custom.txt`
into an empty stage passes lexical admission. Swapping `gen/` then exposes an
empty target through the alias, and the create-only link installs template
bytes where the old user file had been. The exact reviewer overlay reproduced
`user` becoming `template` with no generation error before repair.

The earlier `GenPolicy` Boolean `inside` conflated path spelling and actual live
ownership. Its original conditional result did not establish the concrete
extraction claim; the broader alias-preservation claim was invalidated by this
review. The revised model separates `inside` (reserved spelling) and `alias`
(live ownership reached through a stable directory alias). The old complete
admission control and the newly explicit lexical-only control both violate
`RejectContradiction`; corrected classification preserves it:

| Revised configuration | Outcome | Generated/distinct states |
| --- | --- | --- |
| `GenPolicyCurrent.cfg` | `RejectContradiction` violation | 12 / 12 |
| `GenPolicyLexical.cfg` | `RejectContradiction` violation | 12 / 12 |
| `GenPolicyCorrected.cfg` | Both invariants hold | 16 / 16 |

Run `GenPolicyLexical.cfg` with the same TLC command as the other gen-policy
configurations, using a separate temporary metadir. The external atomic-install
model and its results remain unchanged; this repair changes admission, not
installation or rollback.

`artifactpolicy.Validate` now receives the actual output root, follows stable
parent-directory aliases component by component, retains nonexistent suffixes,
and compares canonical containment and filesystem ancestor identity. Resolving
symlink targets must process a linked component before `..`; cleaning the target
first can erase ownership information. The final path entry is not followed:
an existing external symlink is an occupied entry to preserve. All spellings of
the top-level `gen` component are reserved case-insensitively, including before
the generated tree exists. Unrelated sibling names remain admissible.

The launcher checks the complete artifact batch against the live root before
creating any output directories, including the direct commit path. Rendering
and generator preflight use the same owner against their actual output root.
New correspondence tests are:

- `TestArtifactWriteIntentRejectsLiveAliases`: existing aliases, missing
  descendants and dangling aliases reject without replacing the old generation
  directory or changing its user bytes.
- `TestArtifactWriteIntentRejectsAliasBeforePreparingBatch`: a rejected later
  artifact prevents even an earlier admissible output's directory creation.
- `TestFileSkipExistRejectsLiveAlias` and
  `TestArtifactWriteIntentRejectsLiveAliasBeforeRendering`: standalone and
  generator preflight reject before writing.
- `internal/artifactpolicy` tests cover `GEN`/`Gen` with absent/existing trees,
  symlink targets containing `..`, external sibling acceptance, symlink loops,
  and SameFile detection of differently spelled directory aliases on a
  case-insensitive filesystem. Output-root identity also reserves absent `gen`
  reached through differently cased root paths. Both filesystem-identity tests
  ran and passed on the validation filesystem; neither was skipped.

The model still abstracts filesystem resolution rather than proving it. These
checks assume parent-directory bindings remain stable between admission and
installation; they do not claim safety against an adversary swapping ancestors
concurrently. Leaf-name concurrent creation remains covered by atomic no-replace
installation. The existing whole-tree crash and mount-layout limitations remain.
