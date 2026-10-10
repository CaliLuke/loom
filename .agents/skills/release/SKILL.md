---
name: release
description: Publish Loom alphas or promote an existing alpha to stable using the CI-verified GitHub workflow.
---

# Release

Use this skill with `loom-framework` for Loom publication. The publisher is
`internal/release`, invoked by `.github/workflows/release.yml`. Local commands
dispatch that workflow; they never create release commits or publish tags.

## Select and dispatch

- Manual alpha: select a full commit SHA on canonical `origin/main`, then run
  `make release SOURCE=<sha>`. `VERSION=vX.Y.Z-alpha.N` optionally asserts the
  next version; it does not override allocation or allow duplicate alphas.
- Daily alpha: the workflow runs at 23:47 America/Los_Angeles and selects the
  immutable `GITHUB_SHA` captured for that scheduled run. This is a daily
  snapshot, not an exact wall-clock guarantee: GitHub can delay or drop schedules.
  No new source commits means no new alpha. Never substitute an older green
  commit if the selected one fails CI.
- Stable promotion: run
  `make release-promote ALPHA=vX.Y.Z-alpha.N VERSION=vX.Y.Z`. The stable tag must
  identify the alpha's exact source, even if main has advanced. Keep the alpha
  tag and release intact. Promotion is always explicit.
- Alphas use `internal/release/train.json` from the selected source. After that
  train is promoted, daily publication stops until a reviewed source commit
  advances the train. Do not guess a new minor or patch version automatically.

Run from the canonical repository. The workflow runs on trusted main and
serializes all publication modes with one repository-wide concurrency group.
The publisher validates the fetch URL and every effective push URL of `origin`
before fetching or publishing. Git URL rewrites and separate push destinations
must still resolve to the canonical Loom repository.
Inputs are passed through environment variables, not interpolated into shell
source. Do not publish from PR workflows or add credentials to untrusted jobs.

## Reuse CI

The publisher requires successful main-push CI for the exact selected SHA:
`test.yml` (including its aggregate Release eligibility job) and `codeql.yml`.
It checks the latest applicable run and attempt, and requires every returned
job to succeed. Missing, skipped, cancelled, or failed jobs are not evidence.
Pending CI waits for up to 45 minutes; no local test suite substitutes for it.

Do not rerun unit, coverage, integration, generator or OpenAPI suites to release
an unchanged green commit. Do not create a version-only commit after CI.
`pkg.Version` uses Go module/build metadata, so alpha and stable installations
can report different versions from the same source commit. Local builds report
`(devel)`; never stamp them with a guessed release version.

## Publication and notes

The same publisher creates an annotated tag, a draft GitHub Release, and a
`release-evidence.json` asset recording the source SHA and successful CI runs.
It publishes the draft with notes and verifies the resulting tag and Release.
Tag pushes do not trigger a second publisher. Main is never moved or rewritten.

Notes list source commit subjects since the preceding alpha/release; stable
notes cover changes since the preceding stable release and identify the promoted
alpha. Keep commit subjects descriptive of Loom behavior. Update public usage
and upgrade guidance with implementation commits. Documentation recommendation
pins are curated separately; daily publication does not rewrite them or stamp
historical `Since` annotations. Enhance release notes when significant changes
need more explanation, without adding routine CI command lists.

## Retries and completion

A successful dispatch is not a completed release. Inspect the workflow run and
verify its result before reporting publication:

- the release tag identifies the selected source SHA;
- GitHub has a non-draft Release with the correct prerelease state and notes;
- `release-evidence.json` identifies that source and its required CI runs.

Rerun the same workflow with the same source or alpha after interruption. It
reuses the allocated tag and repairs incomplete draft publication. Conflicting
tags fail; never move a published tag. A completed matching release is a no-op.
If publication is blocked, report the actual failed check rather than adding
skip flags, manual tag pushes, or fallback source selection.

Version reporting and the CI eligibility gate changed with this publisher.
Historical alphas predating `internal/release/train.json` cannot be promoted by
this workflow: publish and validate a new alpha containing this implementation.

For changes to the publisher, run focused policy and retry tests plus the model
in `internal/release/tla/`. Follow AGENTS.md independent review before committing
implementation changes. There are no per-release source commits to review.
