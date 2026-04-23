---
name: release
description: Cut and publish a Loom release safely. Use this when the user asks to release, cut a tag, publish a version, or update the GitHub Releases page for this repo.
---

# Release

Use this skill when publishing a Loom release from this repo.

## Non-Negotiables

- Cut Loom releases with `make release VERSION=vX.Y.Z`.
- Do not rely on hardcoded version defaults. The release version must be explicit.
- The tagged commit must update `pkg/version.go` and the versioned README doc link together.
- Review user-facing docs affected by the release changes and update them before cutting the tag.
- Review the repo-local Loom skill at `.agents/skills/loom/SKILL.md` and update it when the release changes framework behavior, guidance, or version-facing command examples.
- The pushed `v*` tag must result in a matching GitHub Release via `.github/workflows/release.yml`.
- The same workflow also supports manual backfill for an existing `v*` tag when a release entry needs repair.
- Do not call the release done until the tag exists on GitHub and the GitHub Releases page shows that version.

## Workflow

1. Confirm the repo is in a releasable state and inspect `git status --short`.
2. Review the pending changes for documentation impact. Update any affected user-facing docs, guides, examples, release-facing references, and `.agents/skills/loom/SKILL.md` before continuing.
3. Choose the exact target version and pass it explicitly as `VERSION=vX.Y.Z`.
4. Run `make release VERSION=vX.Y.Z`.
5. Verify the tag push succeeded.
6. Verify GitHub Actions created the matching release entry for that tag.
7. Before closing the release, update this release skill too if the release exposed a gap in the documented release process.

## Required Verification

- `pkg/version.go` reports the released version.
- `README.md`, `.agents/skills/loom/SKILL.md`, and any other impacted user-facing docs reflect the released behavior and version references.
- `git tag --list 'v*' --sort=-creatordate | head` includes the new tag.
- GitHub shows a release object for the new tag, not just a tag entry.

## Recovery

- If the tag exists but the GitHub Release is missing, use the manual `workflow_dispatch` path in `.github/workflows/release.yml` with that tag before cutting another release.
- If `make release` is attempted without `VERSION=...`, stop and rerun with an explicit version instead of editing files by hand.
