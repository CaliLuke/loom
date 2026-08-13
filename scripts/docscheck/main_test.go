package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid relative link and anchor",
			content: "# Guide\n\nSee [details](other.md#more-details). `Project[T](...)` is code, not a link.\n",
		},
		{
			name:    "missing file",
			content: "# Guide\n\nSee [missing](missing.md).\n",
			want:    "target does not exist",
		},
		{
			name:    "missing anchor",
			content: "# Guide\n\nSee [details](other.md#missing).\n",
			want:    "anchor does not exist",
		},
		{
			name:    "filesystem design command",
			content: "```bash\nloom gen ./design\n```\n",
			want:    "use a Go import path",
		},
		{
			name:    "machine-local path",
			content: "See [skill](/Users/example/loom/SKILL.md).\n",
			want:    "machine-local absolute path",
		},
		{
			name:    "legacy upstream command",
			content: "Run `goa gen example.com/service/design`.\n",
			want:    "legacy upstream naming",
		},
		{
			name:    "legacy authorizer name",
			content: "Implement the `Auth" + "er` interface.\n",
			want:    "legacy upstream naming",
		},
		{
			name:    "unsupported cookie location",
			content: "```go\nIn(\"cookie\")\n```\n",
			want:    "unsupported Loom DSL form",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "guide.md"), test.content)
			writeTestFile(t, filepath.Join(root, "other.md"), "# More Details\n")

			issues := checkMarkdown(root, []string{"guide.md", "other.md"})
			if test.want == "" {
				require.Empty(t, issues)
				return
			}
			require.Contains(t, strings.Join(issues, "\n"), test.want)
		})
	}
}

func TestCheckDuplicateSkillGuides(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "http-guide.md"), "# HTTP\n")
	writeTestFile(t, filepath.Join(root, ".agents", "skills", "loom", "references", "http-guide.md"), "# Copy\n")
	writeTestFile(t, filepath.Join(root, ".agents", "skills", "loom-framework", "references", "http-guide.md"), "# Copy\n")

	issues := checkDuplicateSkillGuides(root)

	require.Equal(t, []string{
		".agents/skills/loom-framework/references/http-guide.md: duplicates canonical guide name docs/http-guide.md",
		".agents/skills/loom/references/http-guide.md: duplicates canonical guide name docs/http-guide.md",
	}, issues)
}

func TestCheckRecommendedVersion(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "pkg/version.go"), `package loom

const (
	Major = 1
	Minor = 8
	Build = 0
	Suffix = "alpha.1"
)
`)
	writeDocsVersionFixture(t, root, "v1.8.0-alpha.1")
	require.Empty(t, checkRecommendedVersion(root))

	writeTestFile(t, filepath.Join(root, "docs/_index.md"),
		"> **Recommended release: `v1.7.1`.**\n")
	issues := checkRecommendedVersion(root)
	require.Equal(t, []string{
		"docs/_index.md: recommended version v1.7.1 does not match package version v1.8.0-alpha.1",
	}, issues)
}

func TestCheckObserverReasons(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "events.go"), `package events

type Reason string

const (
	ReasonOK Reason = "ok"
	ReasonTimeout Reason = "stream_timeout"
)
`)
	writeTestFile(t, filepath.Join(root, "guide.md"), "Reasons: `ok`.\n")

	issues := checkObserverReasons(root, "events.go", "guide.md")

	require.Equal(t, []string{"guide.md: missing documented observer reason `stream_timeout`"}, issues)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func writeDocsVersionFixture(t *testing.T, root, version string) {
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
}
