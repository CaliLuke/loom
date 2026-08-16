package openapiv3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestRenderedNullablePresenceContractForAllTargets(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantVersion string
	}{
		{name: "OpenAPI 3.1", target: "3.1", wantVersion: openapiv3.OpenAPICompatibilityVersion},
		{name: "OpenAPI 3.2", wantVersion: openapiv3.OpenAPIVersion},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, presenceOpenAPIDSL, test.target, test.wantVersion)
			require.Contains(t, string(artifacts.YAML), "example: null")
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			schema := requireComponentSchema(t, spec, "PresencePatch")
			properties := requireMap(t, schema["properties"], "PresencePatch properties")

			nickname := requireMap(t, properties["nickname"], "nickname schema")
			assertNullableSchema(t, nickname, "string")
			require.Contains(t, nickname, "example")
			require.Nil(t, nickname["example"])

			status := requireMap(t, properties["status"], "status schema")
			assertNullableSchema(t, status, "string")
			require.Contains(t, requireSlice(t, schema["required"], "required properties"), "status")

			values := requireMap(t, properties["values"], "values schema")
			assertNullableSchema(t, requireMap(t, values["items"], "values items"), "integer")

			labels := requireMap(t, properties["labels"], "labels schema")
			assertNullableSchema(t, requireMap(t, labels["additionalProperties"], "label values"), "string")

			note := requireMap(t, properties["note"], "note schema")
			require.Equal(t, "#/components/schemas/NullableNote", note["$ref"])
			require.NotContains(t, note, "anyOf")
			require.Contains(t, note, "example")
			require.Nil(t, note["example"])
			noteComponent := requireComponentSchema(t, spec, "NullableNote")
			noteVariants := requireSlice(t, noteComponent["anyOf"], "nullable named type variants")
			require.Len(t, noteVariants, 2)
			require.False(t, strings.Contains(string(artifacts.JSON), `"anyOf":[{"$ref":"#/components/schemas/NullableNote"}`))
		})
	}
}

func assertNullableSchema(t *testing.T, schema map[string]any, valueType string) {
	t.Helper()
	variants := requireSlice(t, schema["anyOf"], "nullable schema variants")
	require.Len(t, variants, 2)
	require.Equal(t, valueType, requireMap(t, variants[0], "value schema")["type"])
	require.Equal(t, "null", requireMap(t, variants[1], "null schema")["type"])
}

func presenceOpenAPIDSL() {
	dsl.API("presence", func() {
		dsl.Title("Presence API")
		dsl.Description("Exercises authored JSON presence contracts.")
		dsl.Version("1.0.0")
		dsl.Server("presence", func() {
			dsl.Host("production", func() {
				dsl.URI("https://presence.example.test")
			})
		})
	})

	var note = dsl.Type("NullableNote", func() {
		dsl.Nullable()
		dsl.Attribute("text", dsl.String)
		dsl.Required("text")
	})
	var patch = dsl.Type("PresencePatch", func() {
		dsl.Attribute("nickname", dsl.String, func() {
			dsl.Nullable()
			dsl.Example("clear", dsl.Null())
		})
		dsl.Attribute("status", dsl.String, func() {
			dsl.Nullable()
		})
		dsl.Attribute("values", dsl.ArrayOf(dsl.Int, func() {
			dsl.Nullable()
		}))
		dsl.Attribute("labels", dsl.MapOf(dsl.String, dsl.String, func() {
			dsl.Elem(func() {
				dsl.Nullable()
			})
		}))
		dsl.Attribute("note", note, func() {
			dsl.Example("clear", dsl.Null())
		})
		dsl.Required("status")
	})

	dsl.Service("presence", func() {
		dsl.Method("update", func() {
			dsl.NoSecurity()
			dsl.Payload(patch)
			dsl.HTTP(func() {
				dsl.POST("/presence")
			})
		})
	})
}
