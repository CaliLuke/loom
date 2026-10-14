package openapiimport

import (
	json "encoding/json/v2"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"
)

func TestNumericBoundRawNodesPreservePrecision(t *testing.T) {
	for _, source := range []string{
		`{"openapi":"3.1.1","info":{"title":"Exact","version":"1"},"paths":{},"components":{"schemas":{"Bound":{"type":"integer","minimum":9007199254740993,"exclusiveMaximum":9223372036854775807}}}}`,
		"openapi: 3.1.1\ninfo: {title: Exact, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Bound:\n      type: integer\n      minimum: 9007199254740993\n      exclusiveMaximum: 9223372036854775807\n",
	} {
		parsed, err := libopenapi.NewDocument([]byte(source))
		require.NoError(t, err)
		model, err := parsed.BuildV3Model()
		require.NoError(t, err)
		proxy, ok := model.Model.Components.Schemas.Get("Bound")
		require.True(t, ok)
		low := proxy.Schema().GoLow()
		require.Equal(t, "9007199254740993", low.Minimum.ValueNode.Value)
		require.Equal(t, "9223372036854775807", low.ExclusiveMaximum.ValueNode.Value)
	}
}

func TestAnalyzePreservesExactNumericBounds(t *testing.T) {
	for _, keyword := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"} {
		for _, value := range []string{"9007199254740993", "9223372036854775807", "-9007199254740993", "42"} {
			t.Run(keyword+"/"+value, func(t *testing.T) {
				source := fmt.Sprintf(`openapi: 3.1.1
info: {title: Exact bounds, version: "1"}
paths: {}
components:
  schemas:
    Bound:
      type: integer
      format: int64
      %s: %s
`, keyword, value)
				document, diagnostics, err := Analyze([]byte(source))
				require.NoError(t, err)
				require.Empty(t, diagnostics)
				require.Len(t, document.Components.Schemas, 1)
				schema := document.Components.Schemas[0].Schema
				var bound any
				switch keyword {
				case "minimum":
					bound = schema.Minimum
				case "maximum":
					bound = schema.Maximum
				case "exclusiveMinimum":
					bound = schema.ExclusiveMinimum
				case "exclusiveMaximum":
					bound = schema.ExclusiveMaximum
				}
				encoded, err := json.Marshal(bound)
				require.NoError(t, err)
				require.Equal(t, value, string(encoded))
			})
		}
	}
}

func TestAnalyzePreservesYAMLIntegerNumericBounds(t *testing.T) {
	for _, keyword := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"} {
		for _, test := range []struct {
			source string
			want   string
		}{
			{"0x20000000000001", "9007199254740993"},
			{"0o377", "255"},
			{"0b101", "5"},
			{"077", "77"},
			{"1_000", "1000"},
		} {
			t.Run(keyword+"/"+test.source, func(t *testing.T) {
				source := fmt.Sprintf("openapi: 3.1.1\ninfo: {title: YAML bounds, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Bound:\n      type: integer\n      %s: %s\n", keyword, test.source)
				document, diagnostics, err := Analyze([]byte(source))
				require.NoError(t, err)
				require.Empty(t, diagnostics)
				require.Len(t, document.Components.Schemas, 1)
				schema := document.Components.Schemas[0].Schema
				var bound any
				switch keyword {
				case "minimum":
					bound = schema.Minimum
				case "maximum":
					bound = schema.Maximum
				case "exclusiveMinimum":
					bound = schema.ExclusiveMinimum
				case "exclusiveMaximum":
					bound = schema.ExclusiveMaximum
				}
				encoded, err := json.Marshal(bound)
				require.NoError(t, err)
				expected, valid := new(big.Rat).SetString(test.want)
				require.True(t, valid)
				actual, valid := new(big.Rat).SetString(string(encoded))
				require.True(t, valid)
				require.Zero(t, expected.Cmp(actual), "exact source meaning must survive canonical spelling")
			})
		}
	}
}

func TestAnalyzeDiagnosesNumericBoundOutsideDSLAdmission(t *testing.T) {
	for _, format := range []string{"JSON", "YAML"} {
		for _, keyword := range []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"} {
			t.Run(format+"/"+keyword, func(t *testing.T) {
				source := fmt.Sprintf(`{"openapi":"3.1.1","info":{"title":"Exact","version":"1"},"paths":{},"components":{"schemas":{"Bound":{"type":"number","%s":1e309}}}}`, keyword)
				if format == "YAML" {
					source = fmt.Sprintf("openapi: 3.1.1\ninfo: {title: Exact, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Bound:\n      type: number\n      %s: 1e309\n", keyword)
				}
				_, diagnostics, err := Analyze([]byte(source))
				require.NoError(t, err, "1e309 is a numeric source token, not a syntax failure")
				require.NotEmpty(t, diagnostics, "cannot silently discard a numeric bound or render a rejected DSL argument")
				var found bool
				for _, diagnostic := range diagnostics {
					if diagnostic.Path == "#/components/schemas/Bound/"+keyword {
						found = true
						require.Contains(t, diagnostic.Message, "1e309")
						require.Contains(t, diagnostic.Message, "DSL")
					}
				}
				require.True(t, found, "diagnostic must locate the unsupported endpoint")
				fatal, _ := diagnostics.Classify(true)
				require.NotEmpty(t, fatal, "allow-lossy must not authorize dropping a numeric bound")
			})
		}
	}
}

func TestExactNumericImportGenerateExport(t *testing.T) {
	for _, syntax := range []string{"json", "yaml"} {
		t.Run(syntax, func(t *testing.T) {
			source := `{"openapi":"3.1.1","info":{"title":"Exact","version":"1"},"paths":{"/exact":{"get":{"operationId":"read","responses":{"200":{"description":"exact","content":{"application/json":{"schema":{"$ref":"#/components/schemas/Bound"}}}}}}}},"components":{"schemas":{"Bound":{"type":"object","properties":{"lower":{"type":"integer","format":"int64","minimum":9007199254740993,"maximum":9223372036854775807},"upper":{"type":"integer","format":"int64","exclusiveMinimum":-9223372036854775808,"exclusiveMaximum":9007199254740993}}}}}}`
			if syntax == "yaml" {
				source = `openapi: 3.1.1
info: {title: Exact, version: '1'}
paths:
  /exact:
    get:
      operationId: read
      responses:
        '200':
          description: exact
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Bound'}
components:
  schemas:
    Bound:
      type: object
      properties:
        lower: {type: integer, format: int64, minimum: 0x20000000000001, maximum: 9223372036854775807}
        upper: {type: integer, format: int64, exclusiveMinimum: -9223372036854775808, exclusiveMaximum: 9007199254740993}
`
			}
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(rendered), `Minimum("9007199254740993")`)
			require.Contains(t, string(rendered), `ExclusiveMaximum("9007199254740993")`)
			module := requireRenderedDesignGenerates(t, rendered)
			wire, err := os.ReadFile(filepath.Join(module, "gen", "http", "openapi.json"))
			require.NoError(t, err)
			for _, bound := range []string{"9007199254740993", "9223372036854775807", "-9223372036854775808"} {
				require.Contains(t, string(wire), ": "+bound)
			}
		})
	}
}

func TestExactNumericOpenAPI30Exclusives(t *testing.T) {
	source := []byte(`openapi: 3.0.3
info: {title: Exact, version: '1'}
paths: {}
components:
  schemas:
    Bound:
      type: integer
      format: int64
      minimum: 9007199254740993
      maximum: 9223372036854775807
      exclusiveMinimum: true
      exclusiveMaximum: true
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	bounds := document.Components.Schemas[0].Schema
	require.Nil(t, bounds.Minimum)
	require.Nil(t, bounds.Maximum)
	require.Equal(t, "9007199254740993", bounds.ExclusiveMinimum.String())
	require.Equal(t, "9223372036854775807", bounds.ExclusiveMaximum.String())
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 0)
}
