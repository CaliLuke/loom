package openapiimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOneOfImportBehaviorTable(t *testing.T) {
	cases := oneOfImportCases()
	files, err := filepath.Glob("testdata/oneof/*.json")
	require.NoError(t, err)
	require.Len(t, cases, 44)
	require.Len(t, files, len(cases))
	accepted := 0
	for index, tc := range cases {
		require.Equal(t, tc.name+".json", filepath.Base(files[index]))
		if len(tc.diagnostics) == 0 {
			accepted++
		}
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(files[index])
			require.NoError(t, err)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if len(tc.diagnostics) > 0 {
				require.Equal(t, tc.diagnostics, diagnostics, "diagnostic code, source path and owning rule")
				return
			}
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			_, service, found := strings.Cut(string(rendered), "var _ = Service")
			require.True(t, found)
			for _, fragment := range tc.render {
				require.Contains(t, service, fragment)
			}
			for _, fragment := range tc.absent {
				require.NotContains(t, service, fragment)
			}
			for _, fragment := range tc.declarations {
				require.Contains(t, string(rendered), fragment)
			}
		})
	}
	require.Equal(t, 32, accepted)
}

// The accepted rows share the behavior table; none can be admitted without
// also being rendered, generated, compiled and checked against its source spec.
func TestOneOfImportedDesignGenerates(t *testing.T) {
	for _, tc := range oneOfImportCases() {
		if len(tc.diagnostics) > 0 {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("testdata", "oneof", tc.name+".json"))
			require.NoError(t, err)
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			moduleDir := requireRenderedDesignGenerates(t, design)
			requireOneOfSpec(t, source, moduleDir, tc.name)
			if tc.name == "03-inline-enums" || tc.name == "19-nested-union" || tc.name == "40-form-captured-type" {
				requireIndependentRegeneration(t, moduleDir)
			}
		})
	}
}

func TestOneOfReferencedErrorEnvelope(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Errors, version: "1"}
paths:
  /value:
    get:
      responses:
        "204": {description: done}
        "400":
          description: choice
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Choice'}
components:
  schemas:
    Choice: {oneOf: [{type: string}, {type: integer}]}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Attribute("body", ImportedChoice)`)
	require.Contains(t, string(design), `Body("body")`)
	requireRenderedDesignGenerates(t, design)
}

func TestOneOfImportIndependentRegeneration(t *testing.T) {
	source := readFixture(t, "oneof_symmetry.yaml")
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, design)
	requireIndependentRegeneration(t, directory)
}
