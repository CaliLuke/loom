package openapiv3_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"

	numericdata "github.com/CaliLuke/loom/codegen/testdata"
	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestExactNumericOpenAPI(t *testing.T) {
	artifacts := renderOpenAPIArtifactsForVersion(t, func() {
		API("exact", func() {
			Title("Exact numeric contract")
			Version("1")
			Description("Exact numeric contract across source languages.")
			Meta("openapi:example", "false")
		})
		payload := numericdata.ExactNumericPayload()
		Service("exact", func() {
			Description("Exact numeric values.")
			Method("read", func() {
				Description("Accept exact numeric values.")
				Payload(payload)
				HTTP(func() {
					POST("/exact")
				})
			})
		})
	}, "3.1", openapiv3.OpenAPICompatibilityVersion)
	var document map[string]jsontext.Value
	require.NoError(t, json.Unmarshal(artifacts.JSON, &document))
	var components map[string]jsontext.Value
	require.NoError(t, json.Unmarshal(document["components"], &components))
	var schemas map[string]jsontext.Value
	require.NoError(t, json.Unmarshal(components["schemas"], &schemas))
	// Inspect raw numeric tokens rather than decoding through a float carrier.
	for _, expected := range []string{"9007199254740993", "9223372036854775807", "18446744073709551615", "18446744073709551614", "-9223372036854775808", "1e-1000001"} {
		require.Contains(t, string(artifacts.JSON), ": "+expected)
	}
	var yamlDocument yaml.Node
	require.NoError(t, yaml.Unmarshal(artifacts.YAML, &yamlDocument))
	numeric := make(map[string]string)
	var visit func(*yaml.Node)
	visit = func(node *yaml.Node) {
		if node.Kind == yaml.ScalarNode && (node.Tag == "!!int" || node.Tag == "!!float") {
			numeric[node.Value] = node.Tag
		}
		for _, child := range node.Content {
			visit(child)
		}
	}
	visit(&yamlDocument)
	require.Contains(t, numeric, "9007199254740993")
	require.Contains(t, numeric, "18446744073709551615")
	require.Contains(t, numeric, "1e-1000001")
	lintEffectiveCompositionArtifacts(t, artifacts)
}

func TestExactNumericOpenAPIRejectsUnrepresentableAnnotations(t *testing.T) {
	for _, role := range []string{"default", "enum"} {
		t.Run(role, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, func() {
				API("exact", func() {
					Meta("openapi:example", "false")
				})
				base := numericdata.ExactNumericPayload()
				payload := Type("UnrepresentableArtifact", func() {
					Extend(base)
					Field(10, "authored", Float64, func() {
						Minimum(0.1)
						if role == "default" {
							Default(0.1)
						} else {
							Enum(0.1)
						}
					})
				})
				Service("exact", func() {
					Method("read", func() {
						Payload(payload)
						HTTP(func() {
							POST("/exact")
						})
					})
				})
			})
			_, err := openapiv3.Files(root)
			require.ErrorContains(t, err, role)
			require.ErrorContains(t, err, "authored")
			require.ErrorContains(t, err, "unrepresentable")
		})
	}
}

func TestExactNumericOpenAPINestedEnumRefinement(t *testing.T) {
	artifacts := renderOpenAPIArtifactsForVersion(t, func() {
		API("exact", func() {
			Title("Exact inherited enum")
			Version("1")
			Description("Inherited numeric predicates with child refinement.")
			Meta("openapi:example", "false")
		})
		base := Type("EnumBase", func() {
			Attribute("n", Int)
			Enum(map[string]any{"n": 1}, map[string]any{"n": 2})
		})
		derived := Type("EnumRefined", func() {
			Extend(base)
			Attribute("n", Int, func() {
				Minimum(2)
			})
		})
		payload := Type("EnumEnvelope", func() {
			Attribute("nested", derived)
		})
		Service("exact", func() {
			Description("Inherited numeric values.")
			Method("read", func() {
				Description("Accept the refined numeric contract.")
				Payload(payload)
				HTTP(func() {
					POST("/exact")
				})
			})
		})
	}, "3.1", openapiv3.OpenAPICompatibilityVersion)
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Enum []map[string]int `json:"enum"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(artifacts.JSON, &document))
	require.Equal(t, []map[string]int{{"n": 1}, {"n": 2}}, document.Components.Schemas["EnumRefined"].Enum)
	lintEffectiveCompositionArtifacts(t, artifacts)
}
