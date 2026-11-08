package openapiimport

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"
)

func testBundle(t *testing.T, source Source) *bundledDescription {
	t.Helper()
	resolved, _, err := resolveDescription(source)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	bundle, err := bundleResolution(resolved)
	require.NoError(t, err)
	return bundle
}

func TestBundleTransplantEquivalence(t *testing.T) {
	path, err := filepath.Abs("testdata/external/openapi.yaml")
	require.NoError(t, err)
	source, err := os.ReadFile(path)
	require.NoError(t, err)
	bundle := testBundle(t, Source{Bytes: source, RetrievalPath: path})
	require.Empty(t, bundle.diagnostics)
	wantBytes, err := os.ReadFile("testdata/external/bundled.yaml")
	require.NoError(t, err)
	var want, got map[string]any
	require.NoError(t, yaml.Unmarshal(wantBytes, &want))
	require.NoError(t, bundle.root.Decode(&got))
	require.Equal(t, want, got)
	actualDocument, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source, RetrievalPath: path}, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	wantDocument, diagnostics, err := Analyze(wantBytes)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	actualDesign, err := Render(actualDocument, Options{})
	require.NoError(t, err)
	wantDesign, err := Render(wantDocument, Options{})
	require.NoError(t, err)
	require.Equal(t, string(wantDesign), string(actualDesign))
}

func TestBundleNumericTokens(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"jsonLargeExponent", `{"X":{"type":"number","maximum":1e309}}`},
		{"jsonString", `{"X":{"type":"number","maximum":"42"}}`},
		{"yamlString", `X: {type: number, maximum: "42"}`},
		{"explicitString", `X: {type: number, maximum: !!str 42}`},
		{"explicitFloat", `X: {type: number, maximum: !!float "42"}`},
		{"tiny", `X: {type: number, maximum: 1e-1000001}`},
		{"hexFloat", `X: {type: number, maximum: 0x1.4p0}`},
		{"largeInteger", `X: {type: integer, maximum: 9007199254740993}`},
		{"maxInt64", `X: {type: integer, maximum: 9223372036854775807}`},
		{"alias", "n: &n 0x20000000000001\nX: {type: integer, maximum: *n}"},
		{"merge", "base: &base {type: integer, maximum: 42}\nX: {<<: *base}"},
		{"override", "base: &base {maximum: 42}\nX: {<<: *base, type: integer, maximum: 43}"},
		{"firstMergeWins", "a: &a {maximum: 42}\nb: &b {maximum: 43}\nX: {<<: [*a, *b], type: integer}"},
		{"shallow", "a: &a {properties: {n: {maximum: 42}}}\nX: {<<: *a, properties: {n: {minimum: 43}}}"},
		{"aliasedMergeSequence", "a: &a {maximum: 42}\nb: &b {maximum: 43}\ns: &s [*a, *b]\nX: {<<: *s, type: integer}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := resolverFiles(t, resolverEntry("", "models.yaml#/X"), map[string]string{"models.yaml": tc.body})
			bundle := testBundle(t, source)
			require.Empty(t, bundle.diagnostics)
			original, err := numericSources([]byte(tc.body))
			require.NoError(t, err)
			for path, value := range original {
				if len(path) > 3 && path[:4] == "#/X/" {
					require.Equal(t, value, bundle.numbers["#/components/schemas/X"+path[3:]], path)
				}
			}
		})
	}
}

func TestBundlePortableOrder(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		source := resolverFiles(t, resolverEntry("", "https://b.test/models.yaml#/Pet", "https://a.test/models.yaml#/Pet"), map[string]string{
			"alpha/models.yaml": "Pet: {type: string}", "beta/models.yaml": "Pet: {type: integer}",
		})
		root := filepath.Dir(source.RetrievalPath)
		source.Documents = []DocumentMapping{{URI: "https://a.test/", Path: filepath.Join(root, "alpha")}, {URI: "https://b.test/", Path: filepath.Join(root, "beta")}}
		if reverse {
			source.Documents[0], source.Documents[1] = source.Documents[1], source.Documents[0]
		}
		bundle := testBundle(t, source)
		node, err := descriptionNode(bundle.root, "/components/schemas/Pet_2/type")
		require.NoError(t, err)
		require.Equal(t, "string", node.Value)
		node, err = descriptionNode(bundle.root, "/components/schemas/Pet_3/type")
		require.NoError(t, err)
		require.Equal(t, "integer", node.Value)
	}
}

func TestBundleProvenance(t *testing.T) {
	source := resolverFiles(t, resolverEntry("", "models.yaml#/X"), map[string]string{"models.yaml": "X: {type: object, properties: {id: {type: integer}}}"})
	bundle := testBundle(t, source)
	require.Equal(t, SourceLocation{Document: "root:models.yaml", Pointer: "#/X/properties/id/type"}, bundle.origins["#/components/schemas/X/properties/id/type"])
}

func TestBundleBlockedOccurrences(t *testing.T) {
	bundle := testBundle(t, Source{Bytes: resolverEntry("", "missing.yaml#/X", "missing.yaml#/X")})
	require.Len(t, bundle.blocked, 2)
	require.Len(t, bundle.diagnostics, 2)
	for _, key := range []string{"/components/schemas/x-loom-unresolved-1", "/components/schemas/x-loom-unresolved-2"} {
		_, err := descriptionNode(bundle.root, key)
		require.NoError(t, err)
		require.Equal(t, "external-reference", bundle.blocked["#"+key].Code)
		require.NotNil(t, bundle.blocked["#"+key].Origin)
	}
}

func TestBundleJSONEntry(t *testing.T) {
	source := resolverFiles(t, []byte(`{"openapi":"3.2.0","info":{"title":"JSON entry","version":"1"},"paths":{},"components":{"schemas":{"Item":{"$ref":"models.yaml#/Pet"}}}}`), map[string]string{"models.yaml": `Pet: {type: string}`})
	document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.NotNil(t, document)
	_, err = Render(document, Options{})
	require.NoError(t, err)
}

func TestBundleNumericAnalysisParity(t *testing.T) {
	for _, token := range []string{"42", "9007199254740993", "9223372036854775807", "1e309", "1e-1000001", "0x1.4p0", `"42"`, `!!str 42`, `!!float "42"`} {
		t.Run(token, func(t *testing.T) {
			body := fmt.Sprintf("X: {type: integer, maximum: %s}", token)
			source := resolverFiles(t, resolverEntry("", "models.yaml#/X"), map[string]string{"models.yaml": body})
			actual, gotDiagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			inline := []byte(fmt.Sprintf("openapi: 3.2.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    X: {type: integer, maximum: %s}\n", token))
			expected, wantDiagnostics, err := Analyze(inline)
			require.NoError(t, err)
			var gotSchema *Schema
			for _, named := range actual.Components.Schemas {
				if named.Name == "X" {
					gotSchema = named.Schema
				}
			}
			require.NotNil(t, gotSchema)
			require.Equal(t, expected.Components.Schemas[0].Schema.Maximum, gotSchema.Maximum)
			gotCodes := make([]string, 0, len(gotDiagnostics))
			wantCodes := make([]string, 0, len(wantDiagnostics))
			for _, diagnostic := range gotDiagnostics {
				gotCodes = append(gotCodes, diagnostic.Code)
			}
			for _, diagnostic := range wantDiagnostics {
				wantCodes = append(wantCodes, diagnostic.Code)
			}
			require.ElementsMatch(t, wantCodes, gotCodes)
		})
	}
}
