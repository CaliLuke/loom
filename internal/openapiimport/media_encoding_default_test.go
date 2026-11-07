package openapiimport

import (
	"encoding/json/v2"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

const capturedArrayUnion = `{"oneOf":[{"type":"object","properties":{"value":{"type":"array","oneOf":[{"type":"array","items":{"type":"string"}},{"type":"array","items":{"type":"integer"}}]}}},{"type":"object","properties":{"other":{"type":"string"}}}]}`

func TestMultipartCapturedDefaultPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, encoding, want, failure string
		reference                     bool
	}{
		{"source array default", `{}`, "text/plain", "", false},
		{"root style", `{"value":{"style":"form","explode":true}}`, "text/plain", "", false},
		{"referenced style", `{"value":{"style":"form","explode":true}}`, "text/plain", "", true},
		{"explicit false flag", `{"value":{"allowReserved":false}}`, "text/plain", "", true},
		{"explicit JSON", `{"value":{"contentType":"application/json"}}`, "application/json", "", true},
		{"style suppresses explicit JSON", `{"value":{"contentType":"application/json","style":"form"}}`, "text/plain", "", true},
		{"delimited style", `{"value":{"style":"form","explode":false}}`, "text/plain", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(defaultEncodingSource(t, tc.encoding, tc.reference, false))
			require.NoError(t, err)
			if tc.failure != "" {
				require.Contains(t, diagnostics.Error(), tc.failure)
				return
			}
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			switch {
			case tc.want == "text/plain":
				verifyRenderedMultipartContext(t, design)
			case strings.Contains(tc.encoding, "contentType"):
				require.Contains(t, string(design), `Meta("openapi:encoding:value:contentType", "application/json")`)
			default:
				require.Contains(t, string(design), `Meta("http:encoding:sourceType", "array")`)
			}
		})
	}
}

func TestMultipartCapturedDefaultIsolation(t *testing.T) {
	document, diagnostics, err := Analyze(defaultEncodingSource(t, `{"value":{"style":"form","explode":true}}`, true, true))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	for _, operation := range document.Operations {
		if operation.Path == "/value" {
			require.Equal(t, "form", operation.RequestBody.Schema.MediaEncoding["value"].Style)
			require.Empty(t, operation.RequestBody.Schema.MediaEncoding["value"].MediaType)
		} else {
			require.Empty(t, operation.RequestBody.Schema.MediaEncoding)
		}
	}
	for _, named := range document.Components.Schemas {
		require.Empty(t, named.Schema.MediaEncoding, "captured defaults must not become authored slots on %s", named.Name)
	}
	first, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	second, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Equal(t, first, second)
	verifyRenderedMultipartContext(t, first)
}

func TestMultipartCapturedObjectDefault(t *testing.T) {
	schema := `{"oneOf":[{"type":"object","properties":{"value":{"type":"object","oneOf":[{"type":"object","required":["x"],"properties":{"x":{"type":"string"}}},{"type":"object","required":["y"],"properties":{"y":{"type":"integer"}}}]}}},{"type":"object","properties":{"other":{"type":"string"}}}]}`
	source := strings.ReplaceAll(string(rootEncodingSource(schema, `{}`)), "application/x-www-form-urlencoded", "multipart/form-data")
	document, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("http:encoding:sourceType", "object")`)
}

func defaultEncodingSource(t *testing.T, encoding string, reference, second bool) []byte {
	t.Helper()
	var source map[string]any
	require.NoError(t, json.Unmarshal(rootEncodingSource(capturedArrayUnion, encoding), &source))
	if reference {
		var schema any
		require.NoError(t, json.Unmarshal([]byte(capturedArrayUnion), &schema))
		source["components"].(map[string]any)["schemas"].(map[string]any)["Root"] = schema
		media := source["paths"].(map[string]any)["/value"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/x-www-form-urlencoded"].(map[string]any)
		media["schema"] = map[string]any{"$ref": "#/components/schemas/Root"}
	}
	if second {
		source["paths"].(map[string]any)["/default"] = map[string]any{"post": map[string]any{
			"operationId": "defaultValue",
			"requestBody": map[string]any{"required": true, "content": map[string]any{
				"application/x-www-form-urlencoded": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Root"}},
			}},
			"responses": map[string]any{"204": map[string]any{"description": "done"}},
		}}
	}
	wire, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	return []byte(strings.ReplaceAll(string(wire), "application/x-www-form-urlencoded", "multipart/form-data"))
}

// The root multipart generator is integrated separately. This direct boundary
// check parses emitted metadata and re-evaluates its alias/occurrence precedence
// through the expression owner and the shared part resolver.
func verifyRenderedMultipartContext(t *testing.T, design []byte) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "design.go", design, 0)
	require.NoError(t, err)
	sourceFacts := expr.MetaExpr{}
	authored := expr.MetaExpr{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name.Name != "Meta" {
			return true
		}
		key, keyOK := call.Args[0].(*ast.BasicLit)
		value, valueOK := call.Args[1].(*ast.BasicLit)
		if !keyOK || !valueOK {
			return true
		}
		keyText, err := strconv.Unquote(key.Value)
		require.NoError(t, err)
		valueText, err := strconv.Unquote(value.Value)
		require.NoError(t, err)
		if keyText == "http:encoding:sourceType" && valueText == "array" {
			sourceFacts[keyText] = []string{valueText}
		} else if strings.HasPrefix(keyText, "openapi:encoding:value:") {
			authored[keyText] = []string{valueText}
		}
		return true
	})
	require.Equal(t, []string{"array"}, sourceFacts["http:encoding:sourceType"])
	prefix := "openapi:encoding:value:"
	branches := []*expr.NamedAttributeExpr{
		{Name: "strings", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}},
		{Name: "integers", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Int}}}},
	}
	declaration := &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{Untagged: true, Values: branches}, Meta: sourceFacts}}
	occurrence := &expr.AttributeExpr{Type: declaration}
	body := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: occurrence}}, Meta: authored}
	serialization := expr.HTTPParameterSerialization(body, prefix)
	shape := expr.HTTPValueShape(occurrence)
	require.Equal(t, "array", shape.SourceType)
	require.NotNil(t, shape.Branches[0].Items)
	codec, err := httpencoding.ResolveMultipart("value", shape, serialization)
	require.NoError(t, err)
	if len(codec.Branches) > 0 {
		require.Equal(t, httpencoding.Items, codec.Application)
		require.Equal(t, httpencoding.BranchPart, codec.Kind)
		for _, branch := range codec.Branches {
			require.Equal(t, httpencoding.Items, branch.Application)
			require.Equal(t, "text/plain", branch.ContentType)
			require.Equal(t, httpencoding.TextPart, branch.Kind)
		}
	} else {
		require.Equal(t, "text/plain", codec.ContentType)
		require.Equal(t, httpencoding.TextPart, codec.Kind)
	}
	// Source type belongs to this declaration, never to a sibling field using
	// the same inherited authored Encoding Object.
	binary := expr.HTTPValueShape(&expr.AttributeExpr{Type: expr.Bytes})
	require.Empty(t, binary.SourceType)
}

func TestMultipartCapturedMixedBranchDefaults(t *testing.T) {
	var source map[string]any
	require.NoError(t, json.Unmarshal(defaultEncodingSource(t, `{"value":{"style":"form","explode":true}}`, true, false), &source))
	root := source["components"].(map[string]any)["schemas"].(map[string]any)["Root"].(map[string]any)
	root["oneOf"].([]any)[1] = map[string]any{"type": "object", "properties": map[string]any{
		"value": map[string]any{"type": "string", "format": "binary"},
	}}
	wire, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(wire)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	verifyRenderedMultipartContext(t, design)
	require.NotContains(t, string(design), `Meta("openapi:encoding:value:contentType", "text/plain")`)
}
