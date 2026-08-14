package release

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		wantErr string
	}{
		{name: "valid", version: "v1.7.0"},
		{name: "valid prerelease", version: "v1.8.0-alpha.1"},
		{name: "missing prefix", version: "1.7.0", wantErr: "must match vX.Y.Z"},
		{name: "leading zero", version: "v1.08.0", wantErr: "must match vX.Y.Z"},
		{name: "invalid prerelease", version: "v1.8.0-alpha..1", wantErr: "must match vX.Y.Z"},
		{name: "missing component", version: "v1.7", wantErr: "must match vX.Y.Z"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateVersion(test.version)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateVersionAdvance(t *testing.T) {
	t.Parallel()

	require.NoError(t, validateVersionAdvance("v1.6.2", "v1.7.0"))
	require.NoError(t, validateVersionAdvance("v1.6.2", "v2.0.0"))
	require.NoError(t, validateVersionAdvance("v1.7.1", "v1.8.0-alpha.1"))
	require.NoError(t, validateVersionAdvance("v1.8.0-alpha.1", "v1.8.0-alpha.2"))
	require.NoError(t, validateVersionAdvance("v1.8.0-alpha.2", "v1.8.0"))
	require.ErrorContains(t, validateVersionAdvance("v1.6.2", "v1.6.2"), "greater than")
	require.ErrorContains(t, validateVersionAdvance("v1.6.2", "v1.5.9"), "greater than")
	require.ErrorContains(t, validateVersionAdvance("v1.8.0-alpha.2", "v1.8.0-alpha.1"), "greater than")
}

func TestSubstantiveRelease(t *testing.T) {
	t.Parallel()

	validBody := "## What's Changed\n\n- Transactional release preparation with isolated verification.\n\n" +
		"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.6.2...v1.7.0"
	pureGeneratedBody := "## What's Changed\n" +
		"* fix(codegen): skip transport-absent example services by @CaliLuke in " +
		"https://github.com/CaliLuke/loom/pull/245\n" +
		"* fix(openapiimport): avoid error field method collisions by @CaliLuke in " +
		"https://github.com/CaliLuke/loom/pull/244\n\n" +
		"## New Contributors\n" +
		"* @newcontributor made their first contribution in " +
		"https://github.com/CaliLuke/loom/pull/246\n\n" +
		"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.8.0-alpha.3...v1.8.0-alpha.4"
	generatedPlusHighlightsBody := "## Highlights\n\n" +
		"- Generated clients now retry idempotent requests with jittered backoff.\n\n" +
		"## What's Changed\n" +
		"* fix(codegen): skip transport-absent example services by @CaliLuke in " +
		"https://github.com/CaliLuke/loom/pull/245\n\n" +
		"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.8.0-alpha.3...v1.8.0-alpha.4"
	v180DraftStyleBody := "## Highlights\n\n" +
		"- **OpenAPI imports now avoid generated field and method collisions.** A schema\n" +
		"  property named `error` no longer collides with generated result accessors and\n" +
		"  produces uncompilable Loom design code.\n\n" +
		"## Upgrade notes\n\n" +
		"This alpha requires Go 1.27rc2 or later. Install and require the exact\n" +
		"prerelease tag.\n\n" +
		"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.8.0-alpha.3...v1.8.0-alpha.4"
	tests := []struct {
		name    string
		version string
		data    string
		wantErr string
	}{
		{name: "valid", version: "v1.7.0", data: releaseJSON("v1.7.0", validBody, false, false)},
		{name: "valid prerelease", version: "v1.8.0-alpha.1", data: releaseJSON("v1.8.0-alpha.1", validBody, false, true)},
		{name: "wrong tag", version: "v1.7.0", data: releaseJSON("v1.6.2", validBody, false, false), wantErr: "tag"},
		{name: "draft", version: "v1.7.0", data: releaseJSON("v1.7.0", validBody, true, false), wantErr: "draft"},
		{name: "prerelease", version: "v1.7.0", data: releaseJSON("v1.7.0", validBody, false, true), wantErr: "prerelease"},
		{name: "prerelease not marked", version: "v1.8.0-alpha.1", data: releaseJSON("v1.8.0-alpha.1", validBody, false, false), wantErr: "prerelease"},
		{name: "empty", version: "v1.7.0", data: releaseJSON("v1.7.0", "", false, false), wantErr: "substantive"},
		{name: "changelog only", version: "v1.7.0", data: releaseJSON("v1.7.0", "Full Changelog: https://example.com", false, false), wantErr: "substantive"},
		{
			name: "pure generated notes rejected", version: "v1.8.0-alpha.4",
			data: releaseJSON("v1.8.0-alpha.4", pureGeneratedBody, false, true), wantErr: "auto-generated",
		},
		{
			name: "generated notes plus substantive highlights accepted", version: "v1.8.0-alpha.4",
			data: releaseJSON("v1.8.0-alpha.4", generatedPlusHighlightsBody, false, true),
		},
		{
			name: "v1.8.0 draft style with highlights and upgrade notes accepted", version: "v1.8.0-alpha.4",
			data: releaseJSON("v1.8.0-alpha.4", v180DraftStyleBody, false, true),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRelease(test.version, []byte(test.data))
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestIsGeneratedNotesOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "empty body", body: "", want: false},
		{
			name: "pure what's changed and full changelog",
			body: "## What's Changed\n" +
				"* fix: skip transport-absent example services by @CaliLuke in " +
				"https://github.com/CaliLuke/loom/pull/245\n\n" +
				"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.0.0...v1.1.0",
			want: true,
		},
		{
			name: "pure with new contributors section",
			body: "## What's Changed\n" +
				"* fix: bug by @user in https://github.com/CaliLuke/loom/pull/1\n\n" +
				"## New Contributors\n" +
				"* @newuser made their first contribution in https://github.com/CaliLuke/loom/pull/1\n\n" +
				"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.0.0...v1.1.0",
			want: true,
		},
		{
			name: "commit link bullet still generated",
			body: "## What's Changed\n" +
				"* docs: fix typo in https://github.com/CaliLuke/loom/commit/abcdef1\n\n" +
				"Full changelog: https://github.com/CaliLuke/loom/compare/v1.0.0...v1.1.0",
			want: true,
		},
		{
			name: "highlights section is prose, not generated",
			body: "## Highlights\n\n" +
				"- Generated clients now retry idempotent requests with jittered backoff.\n\n" +
				"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.0.0...v1.1.0",
			want: false,
		},
		{
			name: "non-pr bullet is prose, not generated",
			body: "## What's Changed\n\n" +
				"- Transactional release preparation with isolated verification.\n\n" +
				"**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.0.0...v1.1.0",
			want: false,
		},
		{
			name: "changelog link alone has no prose so it is still flagged",
			body: "**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.0.0...v1.1.0",
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, isGeneratedNotesOnly(test.body))
		})
	}
}

func TestUpdateVersionFilesSupportsPrerelease(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pkg/version.go"), `package loom

const (
	Major = 1
	Minor = 7
	Build = 1
	Suffix = ""
)
`)
	writeVersionDocumentation(t, root, "v1.7.1")
	writeFile(t, filepath.Join(root, "http/integration_tests/fixtures/ticktock/gen/loom.json"),
		"{\n  \"loom_version\": \"v1.7.1\"\n}\n")

	changed, err := updateVersionFiles(root, "v1.8.0-alpha.1")
	require.NoError(t, err)
	require.Equal(t, []string{
		".agents/skills/loom/SKILL.md",
		"README.md",
		"docs/_index.md",
		"docs/code-generation.md",
		"docs/dsl-reference.md",
		"docs/quickstart.md",
		"http/integration_tests/fixtures/ticktock/gen/loom.json",
		"pkg/version.go",
	}, changed)

	files := snapshotReleaseFiles(t, root)
	require.Contains(t, files["pkg/version.go"], "Minor = 8")
	require.Contains(t, files["pkg/version.go"], "Build = 0")
	require.Contains(t, files["pkg/version.go"], `Suffix = "alpha.1"`)
	for _, path := range []string{
		".agents/skills/loom/SKILL.md",
		"README.md",
		"docs/_index.md",
		"docs/code-generation.md",
		"docs/dsl-reference.md",
		"docs/quickstart.md",
	} {
		require.Contains(t, files[path], "v1.8.0-alpha.1", path)
		require.NotContains(t, files[path], "v1.7.1", path)
	}
	require.Contains(t, files["http/integration_tests/fixtures/ticktock/gen/loom.json"],
		`"loom_version": "v1.8.0-alpha.1"`)
}

func TestSameRemote(t *testing.T) {
	t.Parallel()

	for _, remote := range []string{
		"https://github.com/CaliLuke/loom.git",
		"git@github.com:CaliLuke/loom.git",
		"ssh://git@github.com/CaliLuke/loom.git",
	} {
		require.True(t, sameRemote(remote, "github.com/CaliLuke/loom"), remote)
	}
	require.False(t, sameRemote("https://github.com/elsewhere/loom.git", "github.com/CaliLuke/loom"))
}

func TestRunPreflightFailureLeavesRepositoryUntouched(t *testing.T) {
	repository := newTestRepository(t)
	preflight := writeCommand(t, "preflight", `#!/bin/sh
echo "preflight failed" >&2
exit 23
`)
	beforeHead := gitOutput(t, repository.root, "rev-parse", "HEAD")
	beforeFiles := snapshotReleaseFiles(t, repository.root)

	err := Run(context.Background(), Config{
		Root:             repository.root,
		Version:          "v1.7.0",
		CanonicalRemote:  repository.remote,
		PreflightCommand: preflight,
		PollAttempts:     1,
	})
	require.ErrorContains(t, err, "release preflight")

	require.Equal(t, beforeHead, gitOutput(t, repository.root, "rev-parse", "HEAD"))
	require.Equal(t, beforeHead, remoteRefOutput(t, repository.root, "refs/heads/main"))
	require.Empty(t, gitOutput(t, repository.root, "tag", "--list", "v1.7.0"))
	require.Empty(t, gitOutput(t, repository.root, "status", "--porcelain"))
	require.Equal(t, beforeFiles, snapshotReleaseFiles(t, repository.root))
	requireNoReleaseWorktree(t, repository.temp)
}

func TestRunPublishesVerifiedReleaseAndFastForwardsCaller(t *testing.T) {
	repository := newTestRepository(t)
	logPath := filepath.Join(repository.temp, "commands.log")
	writeFile(t, filepath.Join(repository.root, ".git/hooks/pre-push"), fmt.Sprintf(`#!/bin/sh
printf 'hook:%%s\n' "$LOOM_RELEASE_VERSION" >> %q
cat >> %q
`, logPath, logPath))
	require.NoError(t, os.Chmod(filepath.Join(repository.root, ".git/hooks/pre-push"), 0o755))
	writeFile(t, filepath.Join(repository.root, ".git/hooks/pre-commit"), fmt.Sprintf(`#!/bin/sh
test -n "$GOLANGCI_LINT_CACHE"
printf 'commit-cache:%%s\n' "$GOLANGCI_LINT_CACHE" >> %q
`, logPath))
	require.NoError(t, os.Chmod(filepath.Join(repository.root, ".git/hooks/pre-commit"), 0o755))
	preflight := writeCommand(t, "preflight", fmt.Sprintf(`#!/bin/sh
printf 'preflight:%%s:%%s\n' "$PWD" "$LOOM_DIR" >> %q
printf 'preflight-cache:%%s\n' "$GOLANGCI_LINT_CACHE" >> %q
test "$(pwd -P)" = "$(cd "$LOOM_DIR" && pwd -P)"
test -n "$GOLANGCI_LINT_CACHE"
test "$(dirname "$LOOM_DIR")" = "$(dirname "$GOLANGCI_LINT_CACHE")"
test "$LOOM_DIR" != "$GOLANGCI_LINT_CACHE"
`, logPath, logPath))
	gh := writeCommand(t, "gh", fmt.Sprintf(`#!/bin/sh
count_file=%q
log_file=%q
count=0
if test -f "$count_file"; then count=$(cat "$count_file"); fi
count=$((count + 1))
printf '%%s' "$count" > "$count_file"
printf 'gh:%%s\n' "$*" >> "$log_file"
if test "$count" -eq 1; then exit 1; fi
if test "$count" -eq 2; then
  printf '%%s\n' '{"tagName":"v1.7.0","body":"Full Changelog: https://example.com","isDraft":false,"isPrerelease":false}'
  exit 0
fi
printf '%%s\n' '{"tagName":"v1.7.0","body":"## Highlights\n\n- Transactional release preparation with isolated verification.\n\n**Full Changelog**: https://github.com/CaliLuke/loom/compare/v1.6.2...v1.7.0","isDraft":false,"isPrerelease":false}'
`, filepath.Join(repository.temp, "gh-count"), logPath))

	err := Run(context.Background(), Config{
		Root:             repository.root,
		Version:          "v1.7.0",
		CanonicalRemote:  repository.remote,
		PreflightCommand: preflight,
		GitHubCommand:    gh,
		PollAttempts:     3,
		PollInterval:     time.Millisecond,
	})
	require.NoError(t, err)

	head := gitOutput(t, repository.root, "rev-parse", "HEAD")
	require.Equal(t, head, remoteRefOutput(t, repository.root, "refs/heads/main"))
	require.Equal(t, head, gitOutput(t, repository.root, "rev-list", "-n", "1", "v1.7.0"))
	require.NotEmpty(t, gitOutput(t, repository.root, "ls-remote", "--tags", "origin", "refs/tags/v1.7.0"))
	require.Empty(t, gitOutput(t, repository.root, "status", "--porcelain"))

	files := snapshotReleaseFiles(t, repository.root)
	require.Contains(t, files["pkg/version.go"], "Major = 1")
	require.Contains(t, files["pkg/version.go"], "Minor = 7")
	require.Contains(t, files["pkg/version.go"], "Build = 0")
	require.Contains(t, files["README.md"], "cmd/loom@v1.7.0")
	for path, contents := range files {
		if strings.HasSuffix(path, "loom.json") {
			require.Contains(t, contents, `"loom_version": "v1.7.0"`)
		}
	}

	commands, err := os.ReadFile(logPath)
	require.NoError(t, err)
	require.Contains(t, string(commands), "preflight:")
	preflightCacheLine := lineWithPrefix(t, string(commands), "preflight-cache:")
	commitCacheLine := lineWithPrefix(t, string(commands), "commit-cache:")
	require.Equal(t, strings.TrimPrefix(preflightCacheLine, "preflight-cache:"),
		strings.TrimPrefix(commitCacheLine, "commit-cache:"))
	require.Contains(t, string(commands), "hook:v1.7.0")
	require.Contains(t, string(commands), "refs/heads/main")
	require.Contains(t, string(commands), "refs/tags/v1.7.0")
	require.Equal(t, 3, strings.Count(string(commands), "gh:release view v1.7.0"))
	requireNoReleaseWorktree(t, repository.temp)
}

func TestRunRejectsNonCanonicalRepositoryBeforeStaging(t *testing.T) {
	repository := newTestRepository(t)

	err := Run(context.Background(), Config{
		Root:             repository.root,
		Version:          "v1.7.0",
		CanonicalRemote:  "https://github.com/not-the-project/loom.git",
		PreflightCommand: "unused",
		PollAttempts:     1,
	})
	require.ErrorContains(t, err, "origin")
	requireNoReleaseWorktree(t, repository.temp)
}

type testRepository struct {
	root   string
	remote string
	temp   string
}

func newTestRepository(t *testing.T) testRepository {
	t.Helper()

	temp := t.TempDir()
	remote := filepath.Join(temp, "remote.git")
	root := filepath.Join(temp, "loom")
	runGit(t, temp, "init", "--bare", "--initial-branch=main", remote)
	runGit(t, temp, "init", "--initial-branch=main", root)
	runGit(t, root, "config", "user.name", "Loom Test")
	runGit(t, root, "config", "user.email", "loom@example.com")

	writeFile(t, filepath.Join(root, "pkg/version.go"), `package loom

const (
	Major = 1
	Minor = 6
	Build = 2
	Suffix = ""
)
`)
	writeVersionDocumentation(t, root, "v1.6.2")
	for _, transport := range []string{"http", "jsonrpc", "grpc"} {
		writeFile(t, filepath.Join(root, transport, "integration_tests/fixtures/ticktock/gen/loom.json"),
			"{\n  \"loom_version\": \"v1.6.2\"\n}\n")
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	runGit(t, root, "remote", "add", "origin", remote)
	runGit(t, root, "push", "-u", "origin", "main")
	return testRepository{root: root, remote: remote, temp: temp}
}

func snapshotReleaseFiles(t *testing.T, root string) map[string]string {
	t.Helper()

	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative != "pkg/version.go" &&
			relative != "README.md" &&
			relative != filepath.Join(".agents", "skills", "loom", "SKILL.md") &&
			!strings.HasPrefix(relative, "docs"+string(filepath.Separator)) &&
			!strings.HasSuffix(relative, "gen/loom.json") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(contents)
		return nil
	})
	require.NoError(t, err)
	return files
}

func releaseJSON(tag, body string, draft, prerelease bool) string {
	return fmt.Sprintf(`{"tagName":%q,"body":%q,"isDraft":%t,"isPrerelease":%t}`,
		tag, body, draft, prerelease)
}

func writeCommand(t *testing.T, name, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o755))
	return path
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}

func writeVersionDocumentation(t *testing.T, root, version string) {
	t.Helper()

	writeFile(t, filepath.Join(root, "README.md"), strings.Join([]string{
		"go install github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get github.com/CaliLuke/loom@" + version,
	}, "\n"))
	writeFile(t, filepath.Join(root, ".agents/skills/loom/SKILL.md"),
		"go install github.com/CaliLuke/loom/cmd/loom@"+version+"\n")
	writeFile(t, filepath.Join(root, "docs/_index.md"),
		"> **Recommended release: `"+version+"`.**\n")
	writeFile(t, filepath.Join(root, "docs/code-generation.md"), strings.Join([]string{
		"go install github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get -tool github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get github.com/CaliLuke/loom/cmd/loom@" + version,
	}, "\n"))
	writeFile(t, filepath.Join(root, "docs/quickstart.md"), strings.Join([]string{
		"go install github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get github.com/CaliLuke/loom@" + version,
	}, "\n"))
	writeFile(t, filepath.Join(root, "docs/dsl-reference.md"),
		"> **Since: unreleased.**\n")
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	return strings.TrimSpace(string(output))
}

func remoteRefOutput(t *testing.T, dir, ref string) string {
	t.Helper()

	return strings.Fields(gitOutput(t, dir, "ls-remote", "origin", ref))[0]
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}

func requireNoReleaseWorktree(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, strings.HasPrefix(entry.Name(), "loom-release-"), entry.Name())
	}
	worktrees := gitOutput(t, filepath.Join(root, "loom"), "worktree", "list", "--porcelain")
	require.NotContains(t, worktrees, "loom-release-")
}

func lineWithPrefix(t *testing.T, contents string, prefix string) string {
	t.Helper()

	for line := range strings.SplitSeq(contents, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	require.Fail(t, "missing line", "prefix %q in %q", prefix, contents)
	return ""
}
