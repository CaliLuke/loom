package main

import (
	"os"
	"path/filepath"
	"strings"

	"testing"

	"github.com/CaliLuke/loom/internal/testingx"

	"github.com/stretchr/testify/require"
)

func TestParseOpenAPIImportDocumentArgs(t *testing.T) {
	arguments, err := parseOpenAPIImportArgs([]string{"openapi", "entry.yaml", "--document", "https://e.test/models.yaml=models.yaml", "--document", "https://e.test/shared/=shared", "--reference-root", "contracts"})
	require.NoError(t, err)
	require.Len(t, arguments.documents, 2)
	require.Equal(t, "https://e.test/models.yaml", arguments.documents[0].URI)
	require.Equal(t, "shared", arguments.documents[1].Path)
	require.Equal(t, "contracts", arguments.referenceRoot)
	for _, flags := range [][]string{
		{"--document", ""}, {"--document", "https://e.test/a="},
		{"--document", "models.yaml=models.yaml"},
		{"--reference-root", ""},
		{"--document", "HTTPS://E.TEST:443/a=one", "--document", "https://e.test/a=two"},
	} {
		_, err := parseOpenAPIImportArgs(append([]string{"openapi", "entry.yaml"}, flags...))
		require.Error(t, err, flags)
	}
	_, err = parseOpenAPIImportArgs([]string{"openapi", "-", "--document", "https://e.test/a=a.yaml"})
	require.ErrorContains(t, err, "stdin")
}

func TestRunOpenAPIImportExternalReferences(t *testing.T) {
	for _, tc := range []struct {
		name     string
		flags    []string
		wantExit int
		write    bool
	}{
		{"strictMapped", nil, 0, true},
		{"tag", []string{"--tag", "pets"}, 0, true},
		{"path", []string{"--path-prefix", "/pets"}, 0, true},
		{"partial", []string{"--skip-unrenderable"}, 0, true},
		{"report", []string{"--report"}, 0, false},
		{"strictBlocked", nil, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			input, model, output := filepath.Join(root, "entry.yaml"), filepath.Join(root, "model.yaml"), filepath.Join(root, "design", "design.go")
			require.NoError(t, os.WriteFile(model, []byte("Pet: {type: string}\n"), 0o600))
			require.NoError(t, os.WriteFile(input, []byte(`openapi: 3.2.0
info: {title: External, version: '1'}
paths:
  /pets:
    get:
      tags: [pets]
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {$ref: 'https://e.test/models.yaml#/Pet'}
`), 0o600))
			args := []string{"openapi", input, "-o", output}
			if tc.name != "strictBlocked" {
				args = append(args, "--document", "https://e.test/models.yaml="+model)
			}
			args = append(args, tc.flags...)
			var exit int
			stdout, stderr, err := captureOutput(t, func() error {
				exit = runOpenAPIImport(args)
				return nil
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantExit, exit, stdout+stderr)
			if tc.write {
				require.FileExists(t, output)
			} else {
				require.NoFileExists(t, output)
			}
			if tc.name == "report" {
				require.Contains(t, stdout, "provenance:")
				require.Contains(t, stdout, "https://e.test/models.yaml#/Pet")
				require.Contains(t, stdout, "#/components/schemas/Pet")
			}
		})
	}
}

func TestMainListsOpenAPITagsWithUnresolvedPathItems(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "entry.yaml")
	require.NoError(t, os.WriteFile(input, []byte(`openapi: 3.2.0
info: {title: Unknown, version: '1'}
paths:
  /known:
    get: {tags: [pets], responses: {'204': {description: OK}}}
  /unknown: {$ref: 'missing.yaml'}
`), 0o600))
	var exit int
	stdout, stderr, err := captureOutput(t, func() error {
		exit = runOpenAPIImport([]string{"openapi", input, "--list-tags"})
		return nil
	})
	require.NoError(t, err)
	require.Zero(t, exit, stderr)
	require.Contains(t, stdout, "lower bounds")
	require.Contains(t, stdout, "unresolved path: /unknown")
}

func TestExternalImportProcessDeterminism(t *testing.T) {
	root := testingx.RepoRoot()
	loomBin := filepath.Join(t.TempDir(), "loom")
	output, err := testingx.RunCmd(root, "go", "build", "-o", loomBin, "./cmd/loom")
	require.NoError(t, err, output)
	var first []byte
	for attempt := range 3 {
		dir := t.TempDir()
		fixtures := filepath.Join(dir, "source")
		require.NoError(t, os.CopyFS(fixtures, os.DirFS(filepath.Join(root, "internal", "openapiimport", "testdata", "external"))))
		input, mappings := externalMappedInput(t, fixtures, dir)
		if attempt == 1 {
			mappings = append(mappings[2:4:4], mappings[:2]...)
		}
		output, err = testingx.RunCmd(dir, loomBin, append([]string{"import", "openapi", input, "-o", "design"}, mappings...)...)
		require.NoError(t, err, output)
		design, err := os.ReadFile(filepath.Join(dir, "design", "design.go"))
		require.NoError(t, err)
		if attempt == 0 {
			first = design
		} else {
			require.Equal(t, string(first), string(design))
		}
	}
}

func externalMappedInput(t *testing.T, fixtures, dir string) (string, []string) {
	t.Helper()
	entry, err := os.ReadFile(filepath.Join(fixtures, "openapi.yaml"))
	require.NoError(t, err)
	input := filepath.Join(dir, "external.yaml")
	require.NoError(t, os.WriteFile(input, []byte(strings.ReplaceAll(string(entry), "paths/pets.yaml", "https://e.test/paths/pets.yaml")), 0o600))
	return input, []string{
		"--document", "https://e.test/paths/pets.yaml=" + filepath.Join(fixtures, "paths", "pets.yaml"),
		"--document", "https://e.test/models.yaml=" + filepath.Join(fixtures, "models.yaml"),
	}
}
