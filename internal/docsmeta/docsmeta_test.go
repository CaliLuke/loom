package docsmeta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecommendedVersion(t *testing.T) {
	root := t.TempDir()
	writeVersionFixture(t, root, "v1.10.0-alpha.5")
	version, err := RecommendedVersion(root)
	require.NoError(t, err)
	require.Equal(t, "v1.10.0-alpha.5", version)
	require.Empty(t, CheckRecommendedVersion(root, version))
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
