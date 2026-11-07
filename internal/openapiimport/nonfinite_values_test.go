package openapiimport

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeRejectsNonfiniteValues(t *testing.T) {
	const schemaPath = unconstrainedExamplePath + "/schema"
	for _, tc := range []struct {
		name         string
		media        string
		code         string
		path         string
		value        string
		defaultValue bool
	}{
		{"number schema infinity", "{schema: {type: number, example: .inf}}", "examples", schemaPath + "/example", "+Inf", false},
		{"number schema NaN", "{schema: {type: number, example: .nan}}", "examples", schemaPath + "/example", "NaN", false},
		{"number schema negative infinity", "{schema: {type: number, example: -.inf}}", "examples", schemaPath + "/example", "-Inf", false},
		{"number media infinity", "{schema: {type: number}, example: .inf}", "examples", unconstrainedExamplePath + "/example", "+Inf", false},
		{"number media NaN", "{schema: {type: number}, example: .nan}", "examples", unconstrainedExamplePath + "/example", "NaN", false},
		{"any schema infinity", "{schema: {example: .inf}}", "examples", schemaPath + "/example", "+Inf", false},
		{"any schema NaN", "{schema: {example: .nan}}", "examples", schemaPath + "/example", "NaN", false},
		{"any media infinity", "{schema: {}, example: .inf}", "examples", unconstrainedExamplePath + "/example", "+Inf", false},
		{"any media NaN", "{schema: {}, example: .nan}", "examples", unconstrainedExamplePath + "/example", "NaN", false},
		{"number nested map", "{schema: {type: object, properties: {x: {type: number}}, example: {x: .inf}}}", "examples", schemaPath + "/example", "+Inf", false},
		{"any nested array", "{schema: {}, example: [.nan, {x: .inf}]}", "examples", unconstrainedExamplePath + "/example", "NaN", false},
		{"number default infinity", "{schema: {type: number, default: .inf}}", "schema-keyword", schemaPath, "+Inf", true},
		{"number default NaN", "{schema: {type: number, default: .nan}}", "schema-keyword", schemaPath, "NaN", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(nonfiniteLiteralSource(tc.media))
			require.NoError(t, err)
			message := fmt.Sprintf("non-finite number %s is not a scalar Go literal", tc.value)
			if tc.defaultValue {
				message = "default value cannot be represented: " + message
			}
			require.Equal(t, Diagnostics{{Code: tc.code, Path: tc.path, Message: message}}, diagnostics)
			fatal, warnings := diagnostics.Classify(false)
			require.Equal(t, diagnostics, fatal)
			require.Empty(t, warnings)
			response := document.Operations[0].Responses[0].Response
			require.Empty(t, response.Examples)
			require.Empty(t, response.Schema.Examples)
			require.Nil(t, response.Schema.Default)
		})
	}
}

func TestAnalyzeRejectsNonfiniteValueDomains(t *testing.T) {
	for _, tc := range []struct {
		name    string
		schema  string
		keyword string
		member  string
		value   string
	}{
		{"mixed number enum", "type: number, enum: [1, .inf]", "enum", "/1", "+Inf"},
		{"only nonfinite enum", "type: number, enum: [.inf]", "enum", "/0", "+Inf"},
		{"nonfinite const", "type: number, const: .nan", "const", "", "NaN"},
		{"nonfinite before type intersection", `type: string, enum: ["valid", .inf]`, "enum", "/1", "+Inf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(nonfiniteLiteralSource("{schema: {" + tc.schema + "}}"))
			require.NoError(t, err)
			require.Equal(t, Diagnostics{{
				Code:    "schema-" + tc.keyword,
				Path:    unconstrainedExamplePath + "/schema/" + tc.keyword + tc.member,
				Message: fmt.Sprintf("non-finite number %s is not a scalar Go literal", tc.value),
			}}, diagnostics)
			fatal, warnings := diagnostics.Classify(true)
			require.Equal(t, diagnostics, fatal)
			require.Empty(t, warnings, "lossy mode must not silently change a value domain")
			_, err = Render(document, Options{PackageName: "design"})
			require.ErrorContains(t, err, "non-finite number")
		})
	}
}

func TestRenderRejectsNonfiniteNormalizedValues(t *testing.T) {
	for _, position := range []string{"schema example", "media example", "default", "enum"} {
		t.Run(position, func(t *testing.T) {
			document, diagnostics, err := Analyze(nonfiniteLiteralSource("{schema: {type: number}}"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			response := &document.Operations[0].Responses[0].Response
			switch position {
			case "schema example":
				response.Schema.Examples = []Example{{Value: math.Inf(1), Scope: "schema"}}
			case "media example":
				response.Examples = []Example{{Value: math.NaN(), Scope: "media"}}
			case "default":
				response.Schema.Default = &SchemaDefault{Value: math.Inf(-1)}
			case "enum":
				response.Schema.Enum = []any{1, math.NaN()}
			}
			_, err = Render(document, Options{PackageName: "design"})
			require.ErrorContains(t, err, "non-finite number")
		})
	}
}

func TestAnalyzeRetainsFiniteLiteralPolicies(t *testing.T) {
	for _, tc := range []struct {
		name  string
		media string
		want  string
	}{
		{"number schema", "{schema: {type: number, example: 1.25}}", "Value(1.25)"},
		{"number media", "{schema: {type: number}, example: 1.25}", "Value(1.25)"},
		{"any schema", "{schema: {example: 1.25}}", "Value(1.25)"},
		{"any media", "{schema: {}, example: 1.25}", "Value(1.25)"},
		{"number default", "{schema: {type: number, default: 1.25}}", "Default(1.25)"},
		{"quoted NaN", `{schema: {type: string, example: "NaN"}}`, `Value("NaN")`},
		{"finite mixed enum", `{schema: {type: number, enum: [1, "ignored", null]}}`, "Enum(1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(nonfiniteLiteralSource(tc.media))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(design), tc.want)
		})
	}
	for _, schema := range []string{"default: .inf", "type: number, default: null", "type: object, default: {}", "type: integer, default: 1.5"} {
		t.Run(schema, func(t *testing.T) {
			document, diagnostics, err := Analyze(nonfiniteLiteralSource("{schema: {" + schema + "}}"))
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			require.Equal(t, "schema-keyword", diagnostics[0].Code)
			require.Contains(t, diagnostics[0].Message, "not compatible with the strict import subset")
			require.Nil(t, document.Operations[0].Responses[0].Response.Schema.Default)
		})
	}
}

func nonfiniteLiteralSource(media string) []byte {
	return []byte(strings.Replace(`openapi: 3.2.0
info: {title: Nonfinite audit, version: "1"}
paths:
  /value:
    get:
      operationId: value
      responses:
        "200":
          description: value
          content:
            application/json: MEDIA
`, "MEDIA", media, 1))
}
