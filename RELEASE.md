# Releasing Loom

This document is intended to help Loom maintainers release new versions of Loom.

## Using `make release`

1. Update `MAJOR`, `MINOR` and `BUILD` as needed in `Makefile`.
2. Confirm the top-level docs surface is current before tagging:
   - `README.md` points at `CaliLuke/loom`
   - `LICENSE` reflects Loom contributors
   - the CLI install path uses `github.com/CaliLuke/loom/cmd/loom`
   - the module path in `go.mod` is `github.com/CaliLuke/loom`
3. Run `make release`

`make release` runs a preflight check (`release-preflight`) after bumping the version and updating
the README badge. The preflight runs `lint`, `test-release` (no coverage artifact), and
`integration-test` before tagging and pushing `main` plus the release tag.

## Manual release procedure

1. Update `MAJOR`, `MINOR` and `BUILD` as needed in `Makefile`.
2. Update `pkg/version.go` and `README.md` to reflect the new version.
3. Commit and push to `main`.
4. Create and push release git tag.
5. Wait for the Go proxy to observe the tag.
