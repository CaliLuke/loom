package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

func TestOpenAPIImportPrivacy(t *testing.T) {
	for _, uri := range []string{"https://review-user:review-secret@auth.test/security", "https://review-user:review-secret@auth.test/%ZZ", "https://review-user:review-secret with-space@auth.test/security", "https://review-user:review-secret with=space@auth.test/path"} {
		t.Run(uri, func(t *testing.T) {
			_, err := parseOpenAPIImportArgs([]string{"openapi", "entry.yaml", "--document", uri + "=document.yaml"})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "review-user")
			require.NotContains(t, err.Error(), "review-secret")
			root := t.TempDir()
			input := filepath.Join(root, "entry.yaml")
			require.NoError(t, os.WriteFile(input, []byte(fmt.Sprintf("openapi: 3.2.0\ninfo: {title: Privacy, version: '1'}\npaths:\n  /pets:\n    get:\n      security: [{%q: []}]\n      responses: {'204': {description: OK}}\n", uri)), 0o600))
			for _, flags := range [][]string{nil, {"--report"}, {"--skip-unrenderable"}} {
				args := append([]string{"openapi", input, "-o", filepath.Join(root, "design")}, flags...)
				var status int
				stdout, stderr, captureErr := captureOutput(t, func() error { status = runOpenAPIImport(args); return nil })
				require.NoError(t, captureErr)
				require.NotZero(t, status)
				require.NotContains(t, stdout+stderr, "review-user")
				require.NotContains(t, stdout+stderr, "review-secret")
			}
		})
	}
}

func TestOpenAPIImportQuotedProse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		marker string
		flags  []string
	}{
		{"quoted_prose_format.yaml", 2, "schema-format", []string{"--report"}},
		{"duplicate_property_prose.yaml", 1, "already defined", nil},
		{"property_pointer_prose.json", 2, "schema-format", []string{"--report"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := filepath.Join(testingx.RepoRoot(), "internal", "openapiimport", "testdata", tc.name)
			var status int
			stdout, stderr, err := captureOutput(t, func() error {
				args := append([]string{"openapi", input, "-o", filepath.Join(t.TempDir(), "design")}, tc.flags...)
				status = runOpenAPIImport(args)
				return nil
			})
			require.NoError(t, err)
			require.Equal(t, tc.status, status)
			require.Contains(t, stdout+stderr, tc.marker)
			if tc.name == "property_pointer_prose.json" {
				require.Contains(t, stdout+stderr, "https:~1~1host.example is unavailable; contact user@example.test")
			} else {
				require.Contains(t, stdout+stderr, "https://host.example is unavailable; contact user@example.test")
			}
			require.NotContains(t, stdout+stderr, "[redacted]")
		})
	}
}

func TestOpenAPIImportMergedURIPrivacy(t *testing.T) {
	for _, name := range []string{"merged_paths.yaml", "merged_components.yaml"} {
		t.Run(name, func(t *testing.T) {
			input := filepath.Join(testingx.RepoRoot(), "internal", "openapiimport", "testdata", name)
			var status int
			stdout, stderr, err := captureOutput(t, func() error {
				status = runOpenAPIImport([]string{"openapi", input, "-o", filepath.Join(t.TempDir(), "design")})
				return nil
			})
			require.NoError(t, err)
			require.Equal(t, 1, status)
			require.Contains(t, stdout+stderr, "already defined")
			require.NotContains(t, stdout+stderr, "review-user")
			require.NotContains(t, stdout+stderr, "review-secret")
		})
	}
}
