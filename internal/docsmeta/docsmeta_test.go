package docsmeta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecommendedVersionLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeVersionFixture(t, root, "v1.7.1")
	writeTestFile(t, filepath.Join(root, "pkg/version.go"), `package loom

const (
	Major = 1
	Minor = 8
	Build = 0
	Suffix = "alpha.1"
)
`)

	current, err := ReadPackageVersion(filepath.Join(root, "pkg/version.go"))
	require.NoError(t, err)
	require.Equal(t, "v1.8.0-alpha.1", current)
	require.Len(t, CheckRecommendedVersion(root, current), 9)

	changed, err := UpdateVersionMetadata(root, current)
	require.NoError(t, err)
	require.Equal(t, []string{
		".agents/skills/loom/SKILL.md",
		"README.md",
		"docs/_index.md",
		"docs/code-generation.md",
		"docs/dsl-reference.md",
		"docs/quickstart.md",
	}, changed)
	require.Empty(t, CheckRecommendedVersion(root, current))

	for _, target := range documentationVersionTargets {
		contents, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(target.path)))
		require.NoError(t, readErr)
		require.NotContains(t, string(contents), "v1.7.1", target.path)
	}
	since, err := os.ReadFile(filepath.Join(root, "docs/dsl-reference.md"))
	require.NoError(t, err)
	require.Contains(t, string(since), "**Since: `v1.8.0-alpha.1`.**")
	require.NotContains(t, string(since), "Since: unreleased")
}

func TestCheckRecommendedVersionRequiresEveryMarker(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeVersionFixture(t, root, "v1.8.0-alpha.1")
	writeTestFile(t, filepath.Join(root, "docs/quickstart.md"),
		"go install github.com/CaliLuke/loom/cmd/loom@v1.8.0-alpha.1\n")

	issues := CheckRecommendedVersion(root, "v1.8.0-alpha.1")

	require.Equal(t, []string{
		"docs/quickstart.md: expected 2 recommended-version markers, found 1",
	}, issues)
}

func TestUpdateVersionMetadataRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	_, err := UpdateVersionMetadata(t.TempDir(), "next")
	require.ErrorContains(t, err, "invalid semantic version")
}

func writeVersionFixture(t *testing.T, root, version string) {
	t.Helper()

	writeTestFile(t, filepath.Join(root, "README.md"), strings.Join([]string{
		"go install github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get github.com/CaliLuke/loom@" + version,
	}, "\n"))
	writeTestFile(t, filepath.Join(root, ".agents/skills/loom/SKILL.md"),
		"go install github.com/CaliLuke/loom/cmd/loom@"+version+"\n")
	writeTestFile(t, filepath.Join(root, "docs/_index.md"),
		"> **Recommended release: `"+version+"`.**\n")
	writeTestFile(t, filepath.Join(root, "docs/code-generation.md"), strings.Join([]string{
		"go install github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get -tool github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get github.com/CaliLuke/loom/cmd/loom@" + version,
	}, "\n"))
	writeTestFile(t, filepath.Join(root, "docs/quickstart.md"), strings.Join([]string{
		"go install github.com/CaliLuke/loom/cmd/loom@" + version,
		"go get github.com/CaliLuke/loom@" + version,
	}, "\n"))
	writeTestFile(t, filepath.Join(root, "docs/dsl-reference.md"),
		"> **Since: unreleased.**\n")
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}
