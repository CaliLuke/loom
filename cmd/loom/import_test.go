package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/openapiimport"
)

func TestParseOpenAPIImportArgs(t *testing.T) {
	cases := map[string]struct {
		args           []string
		wantInput      string
		wantOutput     string
		wantAllowLossy bool
		wantError      string
	}{
		"default output": {
			args:       []string{"openapi", "contract.yaml"},
			wantInput:  "contract.yaml",
			wantOutput: "design",
		},
		"short output": {
			args:       []string{"openapi", "contract.json", "-o", "api_design.go"},
			wantInput:  "contract.json",
			wantOutput: "api_design.go",
		},
		"long output": {
			args:       []string{"openapi", "contract.json", "--output", "generated"},
			wantInput:  "contract.json",
			wantOutput: "generated",
		},
		"allow lossy": {
			args:           []string{"openapi", "contract.json", "--allow-lossy"},
			wantInput:      "contract.json",
			wantOutput:     "design",
			wantAllowLossy: true,
		},
		"wrong format": {
			args:      []string{"swagger", "contract.json"},
			wantError: "usage: loom import openapi",
		},
		"missing input": {
			args:      []string{"openapi"},
			wantError: "usage: loom import openapi",
		},
		"stdin is not supported": {
			args:      []string{"openapi", "-"},
			wantError: "input must be a file; stdin is not supported",
		},
		"extra argument": {
			args:      []string{"openapi", "contract.json", "extra"},
			wantError: "unexpected import arguments: extra",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			parsed, err := parseOpenAPIImportArgs(test.args)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantInput, parsed.input)
			require.Equal(t, test.wantOutput, parsed.output)
			require.Equal(t, test.wantAllowLossy, parsed.allowLossy)
		})
	}
}

func TestParseOpenAPIImportSelectionArgs(t *testing.T) {
	parsed, err := parseOpenAPIImportArgs([]string{
		"openapi",
		"contract.json",
		"--tag", "Face capture",
		"--tag", "Videoselfie",
		"--path-prefix", "/omni/b2b/v1",
		"--path", "/omni/*/device-*",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Face capture", "Videoselfie"}, parsed.selection.Tags)
	require.Equal(t, []string{"/omni/b2b/v1"}, parsed.selection.PathPrefixes)
	require.Equal(t, []string{"/omni/*/device-*"}, parsed.selection.Paths)
	require.False(t, parsed.listTags)

	parsed, err = parseOpenAPIImportArgs([]string{"openapi", "contract.json", "--list-tags"})
	require.NoError(t, err)
	require.True(t, parsed.listTags)

	_, err = parseOpenAPIImportArgs([]string{"openapi", "contract.json", "--path", "["})
	require.ErrorContains(t, err, "invalid OpenAPI path pattern")

	_, err = parseOpenAPIImportArgs([]string{"openapi", "contract.json", "--list-tags", "--tag", "Face"})
	require.ErrorContains(t, err, "--list-tags cannot be combined")
}

func TestParseOpenAPIImportPartialArgs(t *testing.T) {
	report, err := parseOpenAPIImportArgs([]string{"openapi", "contract.json", "--report"})
	require.NoError(t, err)
	require.True(t, report.report)

	partial, err := parseOpenAPIImportArgs([]string{"openapi", "contract.json", "--skip-unrenderable"})
	require.NoError(t, err)
	require.True(t, partial.skipUnrenderable)

	_, err = parseOpenAPIImportArgs([]string{"openapi", "contract.json", "--report", "--skip-unrenderable"})
	require.ErrorContains(t, err, "cannot be combined")
}

func TestImportOpenAPIDesignLossyPolicy(t *testing.T) {
	const metadataOnly = `openapi: 3.1.1
info:
  title: Metadata
  version: 1.0.0
  contact: {name: Loom}
paths: {}
`
	const metadataAndContract = `openapi: 3.1.1
info:
  title: Metadata
  version: 1.0.0
  contact: {name: Loom}
servers: [{url: https://api.example.com}]
paths: {}
`

	cases := map[string]struct {
		source       string
		allowLossy   bool
		wantWarnings openapiimport.Diagnostics
		wantError    string
	}{
		"strict metadata does not write": {
			source:    metadataOnly,
			wantError: "OpenAPI import cannot preserve the input contract",
		},
		"lossy metadata writes with warnings": {
			source:     metadataOnly,
			allowLossy: true,
			wantWarnings: openapiimport.Diagnostics{{
				Code: "info-metadata", Path: "#/info", Message: "summary, terms, contact, and license metadata are not in the strict import subset",
			}},
		},
		"lossy does not downgrade contract diagnostics": {
			source:     metadataAndContract,
			allowLossy: true,
			wantError:  "OpenAPI import cannot preserve the input contract",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			input := filepath.Join(root, "openapi.yaml")
			require.NoError(t, os.WriteFile(input, []byte(test.source), 0o644))
			output := filepath.Join(root, "design")

			target, warnings, err := importOpenAPIDesign(input, output, test.allowLossy)
			if test.wantError != "" {
				require.Empty(t, target)
				require.ErrorContains(t, err, test.wantError)
				require.NoDirExists(t, output)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantWarnings, warnings)
			require.FileExists(t, target)
		})
	}
}

func TestImportOpenAPIDesignSelection(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Selection, version: "1"}
paths:
  /face:
    get:
      operationId: getFace
      tags: [Face]
      responses: {"204": {description: done}}
  /other:
    get:
      operationId: getOther
      tags: [Other]
      callbacks: {ignored: {}}
      responses: {"204": {description: done}}
`)
	root := t.TempDir()
	input := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(input, source, 0o644))
	output := filepath.Join(root, "design")

	target, warnings, report, err := importOpenAPIDesignSelected(
		input,
		output,
		false,
		openapiimport.Selection{Tags: []string{"Face"}},
	)
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, []string{"/other"}, report.UnclaimedPaths)
	rendered, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Contains(t, string(rendered), `Method("GetFace"`)
	require.NotContains(t, string(rendered), `Method("GetOther"`)

	missingOutput := filepath.Join(root, "missing")
	target, warnings, _, err = importOpenAPIDesignSelected(
		input,
		missingOutput,
		false,
		openapiimport.Selection{Tags: []string{"Missing"}},
	)
	require.ErrorContains(t, err, "selection matched no operations")
	require.Empty(t, target)
	require.Empty(t, warnings)
	require.NoDirExists(t, missingOutput)
}

func TestRunOpenAPIImportPartialModes(t *testing.T) {
	const partialSource = `openapi: 3.1.1
info: {title: Partial, version: "1"}
paths:
  /good:
    get:
      operationId: getGood
      responses: {"204": {description: done}}
  /bad:
    post:
      operationId: postBad
      callbacks: {unsupported: {}}
      responses: {"204": {description: done}}
`
	const blockedSource = `openapi: 3.1.1
info: {title: Blocked, version: "1"}
paths:
  /bad:
    post:
      operationId: postBad
      callbacks: {unsupported: {}}
      responses: {"204": {description: done}}
`
	tests := []struct {
		name            string
		source          string
		flag            string
		wantExit        int
		wantOutput      bool
		wantStdout      []string
		wantStderr      []string
		wantNotInDesign string
	}{
		{
			name:       "report is a dry run",
			source:     partialSource,
			flag:       "--report",
			wantExit:   2,
			wantStdout: []string{"importable: 1/2 operations", "blocked:", "callbacks\t1", "POST /bad"},
		},
		{
			name:            "partial output",
			source:          partialSource,
			flag:            "--skip-unrenderable",
			wantExit:        2,
			wantOutput:      true,
			wantStdout:      []string{"design.go"},
			wantStderr:      []string{"importable: 1/2 operations", "POST /bad"},
			wantNotInDesign: `Method("PostBad"`,
		},
		{
			name:       "nothing importable",
			source:     blockedSource,
			flag:       "--skip-unrenderable",
			wantExit:   1,
			wantStderr: []string{"importable: 0/1 operations", "POST /bad"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			input := filepath.Join(root, "openapi.yaml")
			output := filepath.Join(root, "design")
			require.NoError(t, os.WriteFile(input, []byte(test.source), 0o644))

			var exitCode int
			stdout, stderr, err := captureOutput(t, func() error {
				exitCode = runOpenAPIImport([]string{"openapi", input, "-o", output, test.flag})
				return nil
			})
			require.NoError(t, err)
			require.Equal(t, test.wantExit, exitCode)
			for _, expected := range test.wantStdout {
				require.Contains(t, stdout, expected)
			}
			for _, expected := range test.wantStderr {
				require.Contains(t, stderr, expected)
			}
			target := filepath.Join(output, "design.go")
			if !test.wantOutput {
				require.NoFileExists(t, target)
				return
			}
			require.FileExists(t, target)
			rendered, err := os.ReadFile(target)
			require.NoError(t, err)
			require.NotContains(t, string(rendered), test.wantNotInDesign)
		})
	}
}

func TestImportOpenAPIDesignOutputResolution(t *testing.T) {
	source := supportedOpenAPISource(t)
	cases := map[string]struct {
		output     func(string) string
		prepare    func(*testing.T, string)
		wantTarget func(string) string
		wantPkg    string
	}{
		"default directory": {
			output:     func(string) string { return defaultImportOutput },
			wantTarget: func(string) string { return filepath.Join("design", "design.go") },
			wantPkg:    "design",
		},
		"existing directory": {
			output: func(root string) string { return filepath.Join(root, "contract") },
			prepare: func(t *testing.T, output string) {
				require.NoError(t, os.MkdirAll(output, 0o755))
			},
			wantTarget: func(root string) string { return filepath.Join(root, "contract", "design.go") },
			wantPkg:    "contract",
		},
		"non-existing extensionless directory": {
			output:     func(root string) string { return filepath.Join(root, "generated") },
			wantTarget: func(root string) string { return filepath.Join(root, "generated", "design.go") },
			wantPkg:    "generated",
		},
		"trailing separator directory": {
			output: func(root string) string {
				return filepath.Join(root, "trailing") + string(filepath.Separator)
			},
			wantTarget: func(root string) string { return filepath.Join(root, "trailing", "design.go") },
			wantPkg:    "trailing",
		},
		"exact Go file": {
			output:     func(root string) string { return filepath.Join(root, "contract", "imported.go") },
			wantTarget: func(root string) string { return filepath.Join(root, "contract", "imported.go") },
			wantPkg:    "contract",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			input := filepath.Join(root, "openapi.yaml")
			require.NoError(t, os.WriteFile(input, source, 0o644))
			output := test.output(root)
			if test.prepare != nil {
				test.prepare(t, output)
			}

			target, warnings, err := importOpenAPIDesign(input, output, false)
			require.NoError(t, err)
			require.Empty(t, warnings)
			require.Equal(t, test.wantTarget(root), target)
			rendered, err := os.ReadFile(target)
			require.NoError(t, err)
			parsed, err := parser.ParseFile(token.NewFileSet(), target, rendered, parser.AllErrors)
			require.NoError(t, err)
			require.Equal(t, test.wantPkg, parsed.Name.Name)
			require.NotContains(t, string(rendered), "TODO")
			temporary, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".loom-import-*.go"))
			require.NoError(t, err)
			require.Empty(t, temporary)
		})
	}
}

func TestImportOpenAPIDesignDoesNotMutateOutputOnFailure(t *testing.T) {
	cases := map[string]struct {
		source    string
		wantError string
	}{
		"unsupported construct diagnostics": {
			source: `openapi: 3.1.1
info:
  title: Unsupported
  version: 1.0.0
paths: {}
components:
  schemas:
    Choice:
      oneOf:
        - type: string
        - type: integer
`,
			wantError: "OpenAPI import cannot preserve the input contract",
		},
		"unsupported OpenAPI version": {
			source: `openapi: 3.0.3
info:
  title: Old Contract
  version: 1.0.0
paths: {}
`,
			wantError: "unsupported OpenAPI version",
		},
		"invalid document": {
			source:    "openapi: [",
			wantError: "analyze OpenAPI input",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			input := filepath.Join(root, "openapi.yaml")
			require.NoError(t, os.WriteFile(input, []byte(test.source), 0o644))
			output := filepath.Join(root, "generated")

			target, warnings, err := importOpenAPIDesign(input, output, false)
			require.Empty(t, target)
			require.Empty(t, warnings)
			require.ErrorContains(t, err, test.wantError)
			require.NoDirExists(t, output)
		})
	}
}

func TestImportOpenAPIDesignRefusesToOverwrite(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(input, supportedOpenAPISource(t), 0o644))
	output := filepath.Join(root, "design")
	require.NoError(t, os.MkdirAll(output, 0o755))
	target := filepath.Join(output, "design.go")
	require.NoError(t, os.WriteFile(target, []byte("sentinel\n"), 0o600))

	result, warnings, err := importOpenAPIDesign(input, output, false)
	require.Empty(t, result)
	require.Empty(t, warnings)
	require.ErrorContains(t, err, "already exists; refusing to overwrite")
	contents, readErr := os.ReadFile(target)
	require.NoError(t, readErr)
	require.Equal(t, "sentinel\n", string(contents))
	temporary, globErr := filepath.Glob(filepath.Join(output, ".loom-import-*.go"))
	require.NoError(t, globErr)
	require.Empty(t, temporary)
}

func TestImportOpenAPIDesignRejectsInvalidPackageBeforeCreatingOutput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(input, supportedOpenAPISource(t), 0o644))
	output := filepath.Join(root, "bad-name")

	target, warnings, err := importOpenAPIDesign(input, output, false)
	require.Empty(t, target)
	require.Empty(t, warnings)
	require.ErrorContains(t, err, `package name "bad-name" is not a Go identifier`)
	require.NoDirExists(t, output)
}

func TestImportOpenAPIDesignRejectsNonGoOutputFile(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(input, supportedOpenAPISource(t), 0o644))
	output := filepath.Join(root, "design.yaml")

	target, warnings, err := importOpenAPIDesign(input, output, false)
	require.Empty(t, target)
	require.Empty(t, warnings)
	require.ErrorContains(t, err, "must be a .go file or directory")
	require.NoFileExists(t, output)
}

func supportedOpenAPISource(t *testing.T) []byte {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "internal", "openapiimport", "testdata", "supported.yaml"))
	require.NoError(t, err)
	return source
}

func TestMainRoutesOpenAPIImport(t *testing.T) {
	originalArgs := os.Args
	originalImport := importOpenAPI
	defer func() {
		os.Args = originalArgs
		importOpenAPI = originalImport
	}()

	var gotInput, gotOutput string
	var gotAllowLossy bool
	importOpenAPI = func(input, output string, allowLossy bool) (string, openapiimport.Diagnostics, error) {
		gotInput, gotOutput = input, output
		gotAllowLossy = allowLossy
		return filepath.Join("design", "design.go"), openapiimport.Diagnostics{{Code: "examples", Path: "#/paths", Message: "examples omitted"}}, nil
	}
	os.Args = []string{"loom", "import", "openapi", "contract.yaml", "-o", "design", "--allow-lossy"}

	stdout, stderr, err := captureOutput(t, func() error {
		main()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "contract.yaml", gotInput)
	require.Equal(t, "design", gotOutput)
	require.True(t, gotAllowLossy)
	require.Equal(t, filepath.Join("design", "design.go")+"\n", stdout)
	require.Equal(t, "warning: #/paths: examples omitted (examples)\n", stderr)
}

func TestMainRoutesSelectedOpenAPIImport(t *testing.T) {
	originalArgs := os.Args
	originalImport := importSelectedOpenAPI
	defer func() {
		os.Args = originalArgs
		importSelectedOpenAPI = originalImport
	}()

	var gotSelection openapiimport.Selection
	importSelectedOpenAPI = func(
		_ string,
		_ string,
		_ bool,
		selection openapiimport.Selection,
	) (string, openapiimport.Diagnostics, openapiimport.SelectionReport, error) {
		gotSelection = selection
		return filepath.Join("design", "design.go"), nil, openapiimport.SelectionReport{
			UnclaimedPaths: []string{"/other"},
		}, nil
	}
	os.Args = []string{"loom", "import", "openapi", "contract.yaml", "--tag", "Face"}

	stdout, stderr, err := captureOutput(t, func() error {
		main()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"Face"}, gotSelection.Tags)
	require.Equal(t, filepath.Join("design", "design.go")+"\n", stdout)
	require.Equal(t, "unclaimed path: /other\n", stderr)
}

func TestMainListsOpenAPITags(t *testing.T) {
	originalArgs := os.Args
	originalList := listOpenAPITags
	defer func() {
		os.Args = originalArgs
		listOpenAPITags = originalList
	}()

	listOpenAPITags = func(input string) ([]openapiimport.TagSummary, error) {
		require.Equal(t, "contract.yaml", input)
		return []openapiimport.TagSummary{
			{Name: "Face", Operations: 3, Paths: 2},
			{Name: "Other", Operations: 1, Paths: 1},
		}, nil
	}
	os.Args = []string{"loom", "import", "openapi", "contract.yaml", "--list-tags"}

	stdout, stderr, err := captureOutput(t, func() error {
		main()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "TAG\tOPERATIONS\tPATHS\nFace\t3\t2\nOther\t1\t1\n", stdout)
	require.Empty(t, stderr)
}
