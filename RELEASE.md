# Releasing Loom

This document is intended to help Loom maintainers release new versions of Loom.

## Using `make release`

1. Update `MAJOR`, `MINOR` and `BUILD` as needed in `Makefile`.
2. Make sure any downstream examples/plugins repositories you intend to release alongside Loom exist locally and are clean.
3. Confirm the top-level docs surface is current before tagging:
   - `README.md` points at `CaliLuke/loom`
   - `LICENSE` reflects Loom contributors
   - the CLI install path uses `github.com/CaliLuke/loom/v3/cmd/loom`
3. Run `make release`

`make release` runs a preflight check (`release-preflight`) after bumping the version and updating
the README badge. The preflight runs `lint`, `test-release` (no coverage artifact) and
`integration-test` before tagging and pushing.

## Manual release procedure

1. Update `MAJOR`, `MINOR` and `BUILD` as needed in `Makefile`.
2. Update `pkg/version.go` and `README.md` to reflect the new version.
3. Commit and push to v3.
4. Create and push release git tag.
5. Update `go.mod` in the examples repo `master` branch.
6. Run `make` in the examples repo.
7. Push the examples repo `master` branch.
8. Create and push release git tag.
9. Update `go.mod` in the plugins repo `v3` branch.
10. Run `make` in the plugins repo.
11. Create and push release git tag.
