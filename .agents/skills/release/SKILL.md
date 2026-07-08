---
name: release
description: Cut and publish a Loom release safely. Use this when the user asks to release, cut a tag, publish a version, or update the GitHub Releases page for this repo.
---

# Release

Use this skill when publishing a Loom release from this repo.

## Non-Negotiables

- Cut Loom releases with `make release VERSION=vX.Y.Z`.
- Do not rely on hardcoded version defaults. The release version must be explicit.
- The tagged commit must update every version-stamped file together: `pkg/version.go`, the versioned README doc link, and the generated integration fixtures that embed `loom_version` (`*/integration_tests/fixtures/*/gen/loom.json`). `make release` now bumps all of these automatically; if you ever bump by hand, bump all of them or `release-preflight` will fail the fixture-comparison integration tests.
- `make release` aborts if the working tree is dirty (`git status --porcelain` must be empty). Commit or revert everything — including this skill and any Makefile/docs edits you make for the release — **before** invoking it. The `.loom_source_mode` file is gitignored and does not count as dirty.
- Do not mask the result of `make release`. It runs a long `release-preflight` and only commits/tags/pushes if that passes, so its exit code is the source of truth. Never pipe it through `tee`/another command that swallows the status (a green `tee` can hide a failed `make`); if you must capture output, check `${PIPESTATUS[0]}` or run it directly and read the exit code.
- Review user-facing docs affected by the release changes and update them before cutting the tag.
- Review the repo-local Loom skill at `.agents/skills/loom/SKILL.md` and update it when the release changes framework behavior, guidance, or version-facing command examples.
- The pushed `v*` tag must result in a matching GitHub Release via `.github/workflows/release.yml`.
- Every GitHub Release must have meaningful notes before the release is considered done. A bare generated changelog link, empty body, or placeholder text is not acceptable.
- Release notes and release-facing commit messages must describe the Loom behavior shipped, user impact, upgrade notes, and verification. Do not center another framework, upstream project, or inspiration source when the actual value is the Loom improvement.
- The same workflow also supports manual backfill for an existing `v*` tag when a release entry needs repair.
- Do not call the release done until the tag exists on GitHub and the GitHub Releases page shows that version.

## Preflight environment

`make release` runs `release-preflight` (`lint test-release integration-test openapi-contract generated-code-quality`) before it commits anything. That gate needs real tools on `PATH`, or it fails on environmental gaps that look like release bugs:

- `golangci-lint` — installed to `$(go env GOPATH)/bin` by the lint target; make sure that bin dir is on `PATH`.
- `protoc` **and** `protoc-gen-go` — `protoc-gen-go` is a separate `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`; without it the gRPC codegen tests fail with `protoc-gen-go: program not found`.
- `node`/`npx` — the OpenAPI contract tests shell out to `npx @redocly/cli` and `openapi-typescript`; without Node they fail with `npx: executable file not found`.

Export the bin dirs for the whole release invocation, e.g. `export PATH="$HOME/.local/node/bin:$(go env GOPATH)/bin:$PATH"`.

**Loom source mode governs what the integration tests actually exercise.** `make loom-status` shows it. In `local` mode (`make loom-local`) the JSON-RPC temp-module generator uses the working tree, so preflight tests the code you are about to release — use this when the release contains unpushed changes. In `remote` mode (`make loom-remote`, the default) it pins to `git rev-parse HEAD` and checks that commit out from the remote; if HEAD is not pushed yet it silently falls back to the local tree. Either way, preflight regenerates the checked-in fixtures and compares them, so the version-stamped `loom.json` files must already match the version being released (see Non-Negotiables) — which is why `make release` bumps them before running preflight.

## Workflow

1. Confirm the repo is in a releasable state and inspect `git status --short`.
2. Review the pending changes for documentation impact. Update any affected user-facing docs, guides, examples, release-facing references, and `.agents/skills/loom/SKILL.md` before continuing.
3. Draft release notes from the actual commits and changed behavior before tagging. Include:
   - Highlights that explain concrete Loom behavior, not where the idea came from.
   - Breaking changes or required regeneration.
   - Upgrade notes for generated clients, servers, docs, or downstream repos.
   - Verification commands that were run.
   - The full changelog comparison link.
4. Review release-facing commit messages. If a message frames the work as a port from another framework or otherwise undersells the Loom change, reword it before release so the history describes what was actually done.
5. Choose the exact target version and pass it explicitly as `VERSION=vX.Y.Z`.
6. Run `make release VERSION=vX.Y.Z`.
7. Verify the tag push succeeded.
8. Verify GitHub Actions created the matching release entry for that tag.
9. Replace auto-generated empty notes or changelog-only notes with the drafted release notes using `gh release edit vX.Y.Z --notes-file ...` or equivalent.
10. Before closing the release, update this release skill too if the release exposed a gap in the documented release process.

## Required Verification

- `pkg/version.go` reports the released version.
- `README.md`, `.agents/skills/loom/SKILL.md`, and any other impacted user-facing docs reflect the released behavior and version references.
- `git tag --list 'v*' --sort=-creatordate | head` includes the new tag.
- GitHub shows a release object for the new tag, not just a tag entry.
- `gh release view vX.Y.Z --json body --jq .body` shows substantive release notes with user-facing highlights, upgrade notes when relevant, verification, and the changelog link.
- The release body does not merely say `Full Changelog`, and it does not frame Loom work around another framework or upstream source unless that project is itself part of the user-visible contract.

## Recovery

- If the tag exists but the GitHub Release is missing, use the manual `workflow_dispatch` path in `.github/workflows/release.yml` with that tag before cutting another release.
- If `make release` is attempted without `VERSION=...`, stop and rerun with an explicit version instead of editing files by hand.
- If a release was published with empty, placeholder, or changelog-only notes, treat it as an incomplete release repair: inspect the commits, write meaningful notes, update the GitHub Release, then verify the published body.
- If `release-preflight` fails, `make release` stops **before** committing, tagging, or pushing — so there is no partial release to unwind on the remote, but `pkg/version.go`, `README.md`, and the `loom.json` fixtures are left bumped in the working tree. Revert those to the previous version before retrying (do **not** use `git checkout`/`reset`/`restore` — they are forbidden here; edit the values back by hand or let the next `make release VERSION=...` re-bump from a committed state). Then fix the real failure and re-run.
- `mismatched file loom.json` from an integration test means a version-stamped fixture is out of sync with `pkg/version.go`. `loom.json` contains only `{"loom_version": "vX.Y.Z"}`; confirm that is the *only* diff (the fixture-compare stops at the first mismatch, so regenerate and diff the whole tree to be sure real generated code did not also change), then let `make release` bump the fixtures. If other generated files also differ, the release contains a codegen change and the fixtures need a genuine regeneration, not just a version bump.
