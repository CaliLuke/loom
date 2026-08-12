---
name: release
description: Cut and publish a Loom release safely. Use this when the user asks to release, cut a tag, publish a version, or update the GitHub Releases page for this repo.
---

# Release

Use this skill when publishing a Loom release from this repo.

## Non-Negotiables

- Cut Loom releases with `make release VERSION=vX.Y.Z` or an explicit SemVer
  prerelease such as `make release VERSION=vX.Y.Z-alpha.1`.
- Do not rely on hardcoded version defaults. The release version must be explicit.
- The tagged commit must update every version-stamped file together:
  `pkg/version.go`, the versioned README install command, and every generated
  integration fixture that embeds `loom_version`
  (`*/integration_tests/fixtures/*/gen/loom.json`). `make release` updates and
  verifies all of them in its isolated release worktree.
- `make release` requires a clean `main` whose `HEAD` exactly matches the
  canonical `origin/main`. Commit and push every framework, documentation,
  skill, and release-process change before invoking it. The shared Loom source
  mode lives in worktree-local Git metadata and does not dirty the checkout.
- Do not mask the result of `make release`. Its exit code covers staged
  preflight, atomic main/tag publication, remote-ref verification, and matching
  substantive GitHub Release verification. Never pipe it through a command
  that can swallow the original status.
- Review user-facing docs affected by the release changes and update them before cutting the tag.
- Review the repo-local Loom skill at `.agents/skills/loom/SKILL.md` and update it when the release changes framework behavior, guidance, or version-facing command examples.
- The pushed `v*` tag must result in a matching GitHub Release via `.github/workflows/release.yml`.
  Stable tags must publish as stable releases and prerelease tags must publish as prereleases.
- Every GitHub Release must have meaningful notes before the release is considered done. A bare generated changelog link, empty body, or placeholder text is not acceptable.
- Release notes and release-facing commit messages must describe the Loom behavior shipped, user impact, and upgrade notes. Do not center another framework, upstream project, or inspiration source when the actual value is the Loom improvement.
- Do not add a routine `Verification` section or list standard CI commands in release notes. Readers need what changed and any action they must take; the repository and CI retain the verification record.
- The same workflow also supports manual backfill for an existing `v*` tag when a release entry needs repair.
- Do not call the release done until the tag exists on GitHub and the GitHub Releases page shows that version.

## Preflight environment

`make release` runs `release-preflight` (`lint test-release integration-test
openapi-contract generated-code-quality`) in an isolated detached worktree
before it creates a release commit or tag. That gate needs real tools on
`PATH`, or it fails on environmental gaps that look like release bugs:

- The Go version declared in `go.mod`; prerelease directives such as
  `go 1.27rc2` require that exact preview toolchain or a launcher that can
  download it automatically.
- `golangci-lint` — `make depend` installs the pinned version to
  `$(go env GOPATH)/bin`; make sure that bin directory is on `PATH`.
- `staticcheck` — Go 1.27 preview releases require the separately pinned
  Staticcheck RC installed by `make depend`; golangci-lint's bundled analyzer
  remains disabled until its Go 1.27-compatible release is available.
- `protoc` 25.0, `protoc-gen-go` v1.36.12, and
  `protoc-gen-go-grpc` v1.6.2 — `make depend` installs the exact supported
  toolchain. Do not substitute `@latest`.
- `node`/`npx` — the OpenAPI contract tests shell out to `npx @redocly/cli` and `openapi-typescript`; without Node they fail with `npx: executable file not found`.
- `gh` authenticated for `CaliLuke/loom` — release completion polls the
  published GitHub Release and validates its tag, state, and notes.

Export the bin dirs for the whole release invocation, e.g. `export PATH="$HOME/.local/node/bin:$(go env GOPATH)/bin:$PATH"`.

The release command sets `LOOM_DIR` to its isolated release worktree for
preflight. Persistent `make loom-local` / `make loom-remote` state therefore
cannot make a release exercise the wrong source. Remote mode never silently
falls back to a working tree in ordinary development checks either.

## Workflow

1. Confirm the repo is on clean `main`, inspect `git status --short`, and push
   the intended `HEAD` to canonical `origin/main`.
2. Review the pending changes for documentation impact. Update any affected user-facing docs, guides, examples, release-facing references, and `.agents/skills/loom/SKILL.md` before continuing.
3. Draft release notes from the actual commits and changed behavior before tagging. Include:
   - Highlights that explain concrete Loom behavior, not where the idea came from.
   - Breaking changes or required regeneration.
   - Upgrade notes for generated clients, servers, docs, or downstream repos.
   - The full changelog comparison link.
   Exclude routine verification details and CI command lists.
4. Review release-facing commit messages. If a message frames the work as a port from another framework or otherwise undersells the Loom change, reword it before release so the history describes what was actually done.
5. Choose the exact target version and pass it explicitly as `VERSION=vX.Y.Z`
   or `VERSION=vX.Y.Z-prerelease`.
6. Run `make release VERSION=<version>`. The command stages version changes in a
   detached worktree, runs preflight, rejects unexpected mutations, commits and
   tags only after success, atomically pushes `main` plus the annotated tag,
   verifies both remote refs, waits for the matching non-draft GitHub Release
   with the expected stable or prerelease state and substantive notes, and
   finally fast-forwards the caller.
7. Review the published notes against the draft. If generated notes omit user
   impact, upgrade guidance, or verification, repair them with
   `gh release edit vX.Y.Z --notes-file ...` and verify again.
8. Before closing the release, update this release skill if the run exposed a
   process gap.

## Required Verification

- `pkg/version.go` reports the released version.
- `README.md`, `.agents/skills/loom/SKILL.md`, and any other impacted user-facing docs reflect the released behavior and version references.
- `git tag --list 'v*' --sort=-creatordate | head` includes the new tag.
- GitHub shows a release object for the new tag, not just a tag entry, with
  prerelease state matching the tag.
- `git rev-parse HEAD`, `git ls-remote origin refs/heads/main`, and the peeled
  release tag all identify the same release commit.
- `gh release view vX.Y.Z --json body --jq .body` shows substantive release notes with user-facing highlights, upgrade notes when relevant, and the changelog link, without a routine verification section.
- The release body does not merely say `Full Changelog`, and it does not frame Loom work around another framework or upstream source unless that project is itself part of the user-visible contract.

## Recovery

- If the tag exists but the GitHub Release is missing, use the manual `workflow_dispatch` path in `.github/workflows/release.yml` with that tag before cutting another release.
- If `make release` is attempted without `VERSION=...`, stop and rerun with an explicit version instead of editing files by hand.
- If a release was published with empty, placeholder, or changelog-only notes, treat it as an incomplete release repair: inspect the commits, write meaningful notes, update the GitHub Release, then verify the published body.
- If `release-preflight` fails, `make release` removes the isolated worktree and
  leaves the caller checkout, local tags, and remote refs untouched. Fix the
  real failure, commit and push it to `main`, then rerun the same release
  command.
- If atomic push succeeds but GitHub Release verification times out, the remote
  commit and tag already exist even though the caller may not have
  fast-forwarded. Do not cut another version or rerun release preparation. Use
  the workflow's manual dispatch to publish/repair that exact tag, verify it
  with `gh release view`, then fetch and fast-forward local `main`.
- `mismatched file loom.json` from an integration test means a version-stamped fixture is out of sync with `pkg/version.go`. `loom.json` contains only `{"loom_version": "vX.Y.Z"}`; confirm that is the *only* diff (the fixture-compare stops at the first mismatch, so regenerate and diff the whole tree to be sure real generated code did not also change), then let `make release` bump the fixtures. If other generated files also differ, the release contains a codegen change and the fixtures need a genuine regeneration, not just a version bump.
