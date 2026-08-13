package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOpenAPIImportArgs(t *testing.T) {
	cases := map[string]struct {
		args       []string
		wantInput  string
		wantOutput string
		wantError  string
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
			input, output, err := parseOpenAPIImportArgs(test.args)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantInput, input)
			require.Equal(t, test.wantOutput, output)
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

			target, err := importOpenAPIDesign(input, output)
			require.NoError(t, err)
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
		"renderer rejection": {
			source: `openapi: 3.0.3
info:
  title: Old Contract
  version: 1.0.0
paths: {}
`,
			wantError: "cannot target OpenAPI 3.0.3",
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

			target, err := importOpenAPIDesign(input, output)
			require.Empty(t, target)
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

	result, err := importOpenAPIDesign(input, output)
	require.Empty(t, result)
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

	target, err := importOpenAPIDesign(input, output)
	require.Empty(t, target)
	require.ErrorContains(t, err, `package name "bad-name" is not a Go identifier`)
	require.NoDirExists(t, output)
}

func TestImportOpenAPIDesignRejectsNonGoOutputFile(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "openapi.yaml")
	require.NoError(t, os.WriteFile(input, supportedOpenAPISource(t), 0o644))
	output := filepath.Join(root, "design.yaml")

	target, err := importOpenAPIDesign(input, output)
	require.Empty(t, target)
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
	importOpenAPI = func(input, output string) (string, error) {
		gotInput, gotOutput = input, output
		return filepath.Join("design", "design.go"), nil
	}
	os.Args = []string{"loom", "import", "openapi", "contract.yaml", "-o", "design"}

	stdout, stderr, err := captureOutput(t, func() error {
		main()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "contract.yaml", gotInput)
	require.Equal(t, "design", gotOutput)
	require.Equal(t, filepath.Join("design", "design.go")+"\n", stdout)
	require.Empty(t, strings.TrimSpace(stderr))
}
