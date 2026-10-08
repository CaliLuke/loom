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
- The release publisher attests its exact atomic `main` plus annotated-tag push
  after `release-preflight` succeeds. The pre-push hook accepts that exact
  two-ref publication without recursively running the full gate again. All
  ordinary direct-main pushes still run the CI-equivalent gate, and the hook
  clears Git's repository-local environment before tests create temporary
  repositories.
- Review user-facing docs affected by the release changes and update them before cutting the tag.
- Review the repo-local Loom skill at `.agents/skills/loom/SKILL.md` and update it when the release changes framework behavior, guidance, or version-facing command examples.
- The pushed `v*` tag must result in a matching GitHub Release via `.github/workflows/release.yml`.
  Stable tags must publish as stable releases and prerelease tags must publish as prereleases.
- Every GitHub Release must have meaningful notes before the release is considered done. A bare generated changelog link, empty body, or placeholder text is not acceptable. `make release` enforces this: it rejects a body that is only GitHub's auto-generated "What's Changed"/"New Contributors" bullets and "Full Changelog" link, even though `.github/workflows/release.yml` always creates that generated body first as the tag-triggered release's initial content.
- Release notes and release-facing commit messages must describe the Loom behavior shipped, user impact, and upgrade notes. Do not center another framework, upstream project, or inspiration source when the actual value is the Loom improvement.
- Do not add a routine `Verification` section or list standard CI commands in release notes. Readers need what changed and any action they must take; the repository and CI retain the verification record.
- The same workflow also supports manual backfill for an existing `v*` tag when a release entry needs repair.
- Do not call the release done until the tag exists on GitHub and the GitHub Releases page shows that version.

## Reuse verification evidence

Green CI for the exact source commit is release evidence. Do not repeat the
full test, coverage, integration, OpenAPI, or generated-code suites locally
merely because a release is being cut.

Before starting publication:

1. Resolve the clean, pushed `main` SHA after all preparation commits. Inspect
   GitHub Actions runs for that SHA, not the latest run on the branch or a PR's
   synthetic merge commit. For example:
   `gh run list --repo CaliLuke/loom --commit <sha> --json databaseId,headSha,workflowName,status,conclusion,url`.
2. Inspect the latest applicable run attempt and its jobs against the workflow
   definitions at that SHA, including matrix jobs and CodeQL. Record the SHA,
   run URLs, conclusions, and dependency/toolchain inputs. A pending, missing,
   skipped, cancelled, or failed required job is not passing evidence; resolve
   it before publication. Do not substitute an older successful attempt for a
   newer failure.
3. Reuse those results for unchanged source and inputs. A new preparation
   commit needs evidence for its own SHA; do not call an ancestor's CI green
   for the new commit. Reuse valid local results for unaffected checks during
   iteration, and rerun only checks invalidated by changes. Documentation-only
   or linter-configuration edits do not justify another local framework suite.
4. Treat the staged release diff separately from its verified parent. Require
   only the intended version constants, maintained documentation pins, and
   fixture `loom_version` stamps to change. Preserve fixture design digests and
   generated code. Run focused version/docs consistency checks (including
   `go run ./scripts/docscheck` in the staged tree), `git diff --cached --check`,
   and the independent exact-diff review required by `AGENTS.md`. Any change
   beyond version metadata returns to normal implementation validation before
   release preparation.

**Current implementation gap:** `internal/release/release.go` unconditionally
runs `make release-preflight`; it has no CI-evidence reuse option. Inspect the
current implementation before invoking it. While this gap remains, update the
release command to verify the evidence above and validate the staged version
diff before starting another release. Do not launch an hours-long duplicate
preflight simply because the command currently does so. Do not invent a skip
flag, replace a check with a no-op, or manually bypass the transactional
publisher. Keep `make release VERSION=...`, independent review, atomic
publication, and remote release verification as the release contract. A
skill-only edit does not implement this command change.

## Preflight environment (current implementation)

`make release` runs `release-preflight` (`lint test-release coverage-ratchet
integration-test openapi-contract generated-code-quality`) in an isolated detached worktree
before it creates a release commit or tag. That gate needs real tools on
`PATH`, or it fails on environmental gaps that look like release bugs:

- The stable Go version declared in `go.mod`; use a launcher that can download
  that exact toolchain automatically when it is not installed locally.
- `golangci-lint` — `make depend` installs the pinned version to
  `$(go env GOPATH)/bin`; make sure that bin directory is on `PATH`.
- Staticcheck runs through the pinned stable golangci-lint bundle for both
  handwritten and generated source; no separate installation is required.
- `protoc` 36.2, `protoc-gen-go` v1.36.12, and
  `protoc-gen-go-grpc` v1.6.2 — `make depend` installs the exact supported
  toolchain. Do not substitute `@latest`.
- `node`/`npx` — use the stable Node version pinned in `.github/workflows/test.yml`.
  OpenAPI contract tests run Redocly and generate a Hey API client.
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
   Use a `Highlights` or `What's Changed` heading: the current release validator
   requires one of those phrases as well as substantive prose. Exclude routine
   verification details and CI command lists.
4. Review release-facing commit messages. If a message frames the work as a port from another framework or otherwise undersells the Loom change, reword it before release so the history describes what was actually done.
5. Choose the exact target version and pass it explicitly as `VERSION=vX.Y.Z`
   or `VERSION=vX.Y.Z-prerelease`. Apply **Reuse verification evidence** above
   before invoking the publisher; resolve its implementation gap first.
6. Arrange independent review of the exact staged release diff after the
   applicable verification succeeds and before the automatic commit, as required by `AGENTS.md`. A
   temporary repository-local pre-commit hook can pause the isolated release
   worktree at that boundary; preserve existing hooks and remove only the
   temporary hook after publication. Do not change the reviewed diff before
   allowing the commit. Run `make release VERSION=<version>`. The command stages version changes in a
   detached worktree, runs preflight, rejects unexpected mutations, commits and
   tags only after success, atomically pushes `main` plus the annotated tag,
   and verifies both remote refs. Pushing the tag triggers
   `.github/workflows/release.yml`, which immediately creates the GitHub
   Release with `gh release create --generate-notes` — a body of only
   auto-generated "What's Changed"/"Full Changelog" content. `make release`
   then polls that release and blocks until its body is substantive; it never
   accepts the generated-only body, so it will not finish on its own.
7. As soon as the tag push in step 6 lands (watch for the `gh release create`
   run or poll `gh release view vX.Y.Z`), replace the generated body with the
   notes drafted in step 3: `gh release edit vX.Y.Z --notes-file <draft>`.
   Do this promptly — `make release` polls every few seconds for about five
   minutes (60 attempts) by default and exits with an error if that window
   elapses before the body becomes substantive. Once the edit lands, the next
   poll succeeds and `make release` fast-forwards the caller's `main`
   automatically.
8. If `make release` times out before you finish the edit, it has already
   published the tag and (generated-notes) release; follow the timeout
   recovery below instead of rerunning release preparation.
9. Before closing the release, update this release skill if the run exposed a
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
- If verification fails before publication, `make release` removes the isolated worktree and
  leaves the caller checkout, local tags, and remote refs untouched. Fix the
  real failure and commit and push it to `main`. Reassess the evidence for that
  SHA and rerun affected checks only; do not restart every passing suite after
  an unrelated tool or environment failure. Apply **Reuse verification
  evidence** before retrying the release command.
- If atomic push succeeds but GitHub Release verification times out — including
  the expected case where the tag-triggered workflow published only
  auto-generated notes and the substantive edit did not land within the poll
  window — the remote commit and tag already exist even though the caller may
  not have fast-forwarded. Do not cut another version or rerun release
  preparation. The workflow's `workflow_dispatch` path only creates a *missing*
  release and no-ops when one already exists, so it does not repair notes here.
  Instead: inspect the commits, write meaningful notes (or reuse the step 3
  draft), publish them with `gh release edit vX.Y.Z --notes-file ...`, verify
  the body with `gh release view vX.Y.Z --json body --jq .body`, then fetch and
  fast-forward local `main` (`git fetch origin && git merge --ff-only
  origin/main`) since `make release` exited before doing so.
- More generally, if a release was ever published with empty, placeholder, or
  changelog-only notes outside this exact flow, treat it the same way: inspect
  the commits, write meaningful notes, update the GitHub Release with
  `gh release edit`, then verify the published body.
- `mismatched file loom.json` from an integration test can mean a fixture's
  `loom_version` is out of sync with `pkg/version.go`. Compare the complete
  generated-tree diff: the comparison stops at the first mismatch. If the only
  change is the version stamp, let the publisher update it and preserve the
  design digest. Other differences require investigation and affected codegen
  validation; a version bump must not conceal them.
