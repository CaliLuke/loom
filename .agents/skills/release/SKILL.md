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

The publisher's generated commit list is source material for release notes,
not a finished account of a substantial release. Curate the published notes
before reporting the release complete. Publication verification remains required;
its results belong in the release evidence and task handoff, not the notes.

### Write for application developers

Release notes explain what changed for users, why it matters, and what they
need to do when upgrading. They are not an agent work log or a delivery report.

- Compare a stable release with the preceding **stable** release, covering the
  whole alpha series rather than just the last dependency update or alpha.
  For an alpha, compare with the preceding alpha/release.
- Inspect the complete commit range, including merged work, and check material
  claims against the released source and public guides. Group related changes
  by capability or subsystem; do not turn every implementation commit into a
  separate feature or call an existing capability new.
- Explain substantial features and fixes concretely: what users can now express
  or do, what previously failed, and what behavior changes. Use examples when
  they make the difference clearer. A handful of generic highlights cannot
  stand in for a large release; more words or commit counts cannot substitute
  for meaningful coverage either.
- Give breaking changes and migration steps their own clear section. Include
  required toolchains, regeneration, changed APIs/wire formats, deployment
  ordering and persisted-data migrations where applicable. Link the tagged
  public documentation that helps users adopt the change.
- Mention a library update only when its shipped fix, behavior or adoption
  requirement matters to users. Distinguish runtime, generator and transitive
  dependencies when relevant. Omit version bumps for test consumers, fixtures,
  lint tools and CI environments. For example, TanStack Query installed in a
  generated-client smoke test is not a Loom application dependency or feature.
- Keep the full comparison link. An optional collapsed commit appendix can
  preserve history, but cannot replace the curated explanation. Keep commit
  subjects descriptive of Loom behavior.

### Exclude delivery bookkeeping

Do not put CI pass statements, workflow/CodeQL links, test command or coverage
inventories, race/shuffle run summaries, evidence-asset descriptions, release
eligibility checks, source SHA/promotion mechanics, or agent verification and
review narratives in the editorial notes. Those are maintainer records.
Do not list internal test-tool pins or explain why they are not dependencies.
Do not add probe methodology or unmeasured-scenario disclaimers merely to report
the work performed. Include a limitation when it changes a user's supported
behavior or upgrade decision, not because it appeared in an agent's test report.

Before publishing or editing, read the entire editorial body as an application
developer. Every paragraph should explain a relevant change, its impact, or an
upgrade action. Remove process commentary rather than relabeling it as
"verification" or adding a disclaimer. Check the published body after editing.
User requests for thorough notes mean thorough product coverage, not exhaustive
development bookkeeping.

### Independent editorial review

For a substantial release or corrective rewrite, have a fresh agent that did
not draft the notes review the exact proposed body with this skill. Review
reader value and factual accuracy, not just spelling: flag vague benefits,
implementation jargon, repeated explanations, unnecessary caveats, inflated
claims, and missing migration actions. Preserve meaningful coverage when
cutting prose. Ask for exact passages and concrete edits, not a generic verdict.
Resolve findings and have the reviewer check the revisions in context before
publishing. A review of an earlier body does not approve later unreviewed edits.

Update public usage and upgrade guidance with implementation commits.
Documentation recommendation pins are curated separately; daily publication
does not rewrite them or stamp historical `Since` annotations.

## Reflect and improve

After a release or a material correction, reflect before closing the task:

1. Use actual user feedback, accepted review findings and publication failures
   to identify what went wrong. Distinguish a missing or ambiguous instruction
   from an existing instruction that was not followed. Do not invent lessons
   merely to produce a retrospective.
2. Find the cause and its owner. Improve the relevant skill, example, review
   step or release tooling within the authorized scope. Prefer tightening the
   existing rule or replacing a weak example over appending another overlapping
   prohibition. A missed instruction may need a better review step rather than
   more instructions. Do not turn one incident into unrelated process changes.
3. Check the proposed improvement against the actual failure and a useful
   counterexample: would it reject the bad passage while retaining a concrete
   feature, bug fix or migration instruction? Record a short before/after example
   in the owning guidance only when it clarifies a reusable distinction. Apply
   the correction to the current notes and check for the same problem elsewhere
   in the editorial body.
4. Get independent review of any guidance or tooling diff before committing,
   as required by AGENTS.md. Re-review corrected notes under the improved
   guidance. Match validation to the change; prose-only corrections do not
   require framework or generator suites.
5. Briefly report the durable improvement in the task handoff. Keep reflection,
   review findings and process changes out of the public release notes. If the
   existing guidance already covers the issue, say what enforcement changed;
   do not manufacture a skill edit or permanent retrospective document.

Reflection should leave a smaller, clearer or more effective process, not a
growing log of mistakes. Remove superseded or redundant guidance as it evolves.

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
