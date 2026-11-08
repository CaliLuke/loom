package openapiimport

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportParameterStylesAccepted(t *testing.T) {
	files, err := filepath.Glob("testdata/parameter_styles/*.yaml")
	require.NoError(t, err)
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".yaml")
		if strings.HasPrefix(name, "reject-") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(file)
			require.NoError(t, err)
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(doc, Options{PackageName: "design"})
			require.NoError(t, err)
			requireRenderedDesignGenerates(t, rendered)
		})
	}
}

func TestImportParameterStylesRejected(t *testing.T) {
	reasons := map[string]string{
		"reject-content-style.yaml":    "content-based parameters cannot declare style",
		"reject-cookie-form-list.yaml": "wrong cookie delimiter",
		"reject-deep-array.yaml":       "deepObject is defined only for objects",
		"reject-deep-nested.yaml":      "nested values have no style serialization",
		"reject-nullable.yaml":         "value shape has no supported parameter codec",
		"reject-query-matrix.yaml":     "style matrix is not defined for this parameter location",
		"reject-space-explode.yaml":    "n/a for explode: true",
		"reject-two-residual.yaml":     "overlapping parameter namespaces",
	}
	files, err := filepath.Glob("testdata/parameter_styles/reject-*.yaml")
	require.NoError(t, err)
	require.Len(t, files, 8)
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			source, err := os.ReadFile(file)
			require.NoError(t, err)
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			fatal, _ := diagnostics.Classify(true)
			require.NotEmpty(t, fatal)
			require.Contains(t, fatal.Error(), reasons[filepath.Base(file)])
			_, err = Render(doc, Options{})
			require.Error(t, err)
			partial, _, err := AnalyzePartial(Source{Bytes: source}, Selection{}, true)
			require.NoError(t, err)
			require.Empty(t, partial.Document.Operations)
		})
	}
}

func TestImportParameterRenderMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, location, fields, schema string
		want, absent                   []string
	}{
		{"cookie default", "cookie", "", `{type: string}`, []string{`Meta("openapi:style", "form")`}, nil},
		{"cookie raw", "cookie", ", style: cookie", `{type: string}`, []string{`Meta("openapi:style", "cookie")`}, nil},
		{"query map", "query", "", `{type: object, additionalProperties: {type: integer}}`, []string{`Meta("openapi:style", "form")`}, nil},
		{"query map reserved", "query", ", allowReserved: true", `{type: object, additionalProperties: {type: integer}}`, []string{`Meta("openapi:style", "form")`, `Meta("openapi:allowReserved", "true")`}, nil},
		{"query map escaping", "query", ", allowReserved: false", `{type: object, additionalProperties: {type: integer}}`, []string{`Meta("openapi:style", "form")`, `Meta("openapi:allowReserved", "false")`}, nil},
		{"query scalar defaults", "query", ", style: form, explode: true", `{type: string}`, nil, []string{`Meta("openapi:style"`, `Meta("openapi:explode"`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: style, version: '1'}
paths:
  /value:
    get:
      parameters: [{name: value, in: %s, schema: %s%s}]
      responses: {'204': {description: ok}}
`, tc.location, tc.schema, tc.fields))
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(doc, Options{})
			require.NoError(t, err)
			for _, want := range tc.want {
				require.Contains(t, string(rendered), want)
			}
			for _, absent := range tc.absent {
				require.NotContains(t, string(rendered), absent)
			}
			// A manually assembled schema parameter gets the same OpenAPI defaults.
			if tc.fields == "" {
				doc.Operations[0].Parameters[0].Serialization.Style = ""
				direct, err := Render(doc, Options{})
				require.NoError(t, err)
				require.Equal(t, rendered, direct)
			}
		})
	}
	for _, name := range []string{"form-content-explode", "form-reserved-false"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile("testdata/parameter_styles/" + name + ".yaml")
			require.NoError(t, err)
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(doc, Options{})
			require.NoError(t, err)
			require.NotContains(t, string(rendered), `Meta("openapi:encoding:field:style"`)
			if name == "form-content-explode" {
				require.Contains(t, string(rendered), `Meta("openapi:encoding:field:contentType", "application/json")`)
				require.Contains(t, string(rendered), `Meta("openapi:encoding:field:explode", "false")`)
			} else {
				require.Contains(t, string(rendered), `Meta("openapi:encoding:field:allowReserved", "false")`)
			}
			// Preserve authored flag presence; the shared owner, not the importer,
			// derives the same grammar as an explicit form style after DSL evaluation.
			requireRenderedDesignEvaluates(t, rendered, 1, `
 body := expr.Root.Services[0].Methods[0].Payload
 captured, err := expr.HTTPFormBodyEncoding(body)
 if err != nil {
  t.Fatal(err)
 }
 explicit := expr.DupAtt(body)
 explicit.Meta["openapi:encoding:field:style"] = []string{"form"}
 canonical, err := expr.HTTPFormBodyEncoding(explicit)
 if err != nil {
  t.Fatal(err)
 }
 actual, expected := captured.Fields["field"].Codec, canonical.Fields["field"].Codec
 if actual.Style != expected.Style || actual.Explode != expected.Explode || actual.Application != expected.Application {
  t.Fatalf("captured codec %v does not match explicit form %v", actual, expected)
 }
 `)
		})
	}
}

func TestImportParameterStylesIndependentRegeneration(t *testing.T) {
	for _, name := range []string{"cookie-map", "form-content-explode", "query-deep-object"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile("testdata/parameter_styles/" + name + ".yaml")
			require.NoError(t, err)
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(doc, Options{})
			require.NoError(t, err)
			directory := requireRenderedDesignGenerates(t, rendered)
			requireIndependentRegeneration(t, directory)
		})
	}
}
