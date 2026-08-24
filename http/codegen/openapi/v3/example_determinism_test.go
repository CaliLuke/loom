package openapiv3_test

import (
	"bytes"
	"path/filepath"
	"regexp"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestFilesSynthesizedExamplesAreStableAcrossRepeatedGeneration(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	first := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))
	openapi.Definitions = make(map[string]*openapi.Schema)
	second := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))

	require.Equal(t, first, second)
}

func TestFilesUnrelatedSchemaChangesDoNotShiftSynthesizedExamples(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	before := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))
	beforeJSON := decodeOpenAPIJSON(t, []byte(before[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	openapi.Definitions = make(map[string]*openapi.Schema)
	after := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(true, false)))
	afterJSON := decodeOpenAPIJSON(t, []byte(after[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	require.Equal(t, responseExample(t, beforeJSON, "/stable"), responseExample(t, afterJSON, "/stable"))
	require.Equal(t, componentSchema(t, beforeJSON, "StableResponseBody"), componentSchema(t, afterJSON, "StableResponseBody"))
}

func TestFilesSynthesizedExamplesDistinguishSchemaOccurrences(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	artifacts := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))
	spec := decodeOpenAPIJSON(t, []byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	example := requireMap(t, responseExample(t, spec, "/stable"), "stable response example")
	require.NotEqual(t, example["message"], example["detail"])
	properties := requireMap(t, componentSchema(t, spec, "StableResponseBody")["properties"], "stable properties")
	message := requireMap(t, properties["message"], "message schema")
	detail := requireMap(t, properties["detail"], "detail schema")
	require.NotEqual(t, message["example"], detail["example"])
}

func TestFilesCanOmitSynthesizedExamplesWithoutRemovingAuthoredExamples(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	root := httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, true))
	originalRandomizer := root.API.ExampleGenerator.Randomizer
	artifacts := renderOpenAPIExampleArtifacts(t, root)

	jsonSpec := decodeOpenAPIJSON(t, []byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.json")]))
	require.NotContains(t, responseMedia(t, jsonSpec, "/stable"), "example")
	require.Equal(t, "authored-example", responseExample(t, jsonSpec, "/authored"))
	stableOperation := requireOperation(t, jsonSpec, "/stable", "get")
	require.Equal(t, "authored-header", parameterByName(t, stableOperation, "X-Authored")["example"])
	require.Equal(t, "authored-cookie", parameterByName(t, stableOperation, "authored_cookie")["example"])
	yamlSpec := decodeOpenAPIExampleYAML(
		t,
		[]byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.yaml")]),
	)
	require.NotContains(t, responseMedia(t, yamlSpec, "/stable"), "example")
	require.Equal(t, "authored-example", responseExample(t, yamlSpec, "/authored"))
	require.True(t, originalRandomizer == root.API.ExampleGenerator.Randomizer)
}

func TestFilesPatternExamplesArePrintable(t *testing.T) {
	openapi.Definitions = make(map[string]*openapi.Schema)
	artifacts := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, patternExampleDSL))
	spec := decodeOpenAPIJSON(t, []byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	example := requireMap(t, responseExample(t, spec, "/pattern"), "pattern response example")
	value := requireString(t, example["value"], "pattern value example")
	require.Regexp(t, regexp.MustCompile(`^\S+$`), value)
	for _, candidate := range value {
		require.True(t, unicode.IsPrint(candidate), "example contains non-printable rune %U", candidate)
		require.False(t, unicode.IsControl(candidate), "example contains control rune %U", candidate)
	}
}

func renderOpenAPIExampleArtifacts(t *testing.T, root *expr.RootExpr) map[string]string {
	t.Helper()
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	artifacts := make(map[string]string, len(files))
	for _, file := range files {
		var output bytes.Buffer
		for _, section := range file.AllSections() {
			require.NoError(t, section.Write(&output))
		}
		artifacts[file.Path] = output.String()
	}
	return artifacts
}

func responseExample(t *testing.T, spec map[string]any, path string) any {
	t.Helper()
	return responseMedia(t, spec, path)["example"]
}

func responseMedia(t *testing.T, spec map[string]any, path string) map[string]any {
	t.Helper()
	operation := requireOperation(t, spec, path, "get")
	responses := requireMap(t, operation["responses"], path+" responses")
	response := requireMap(t, responses["200"], path+" response")
	content := requireMap(t, response["content"], path+" content")
	return requireMap(t, content["application/json"], path+" JSON media")
}

func componentSchema(t *testing.T, spec map[string]any, name string) map[string]any {
	t.Helper()
	components := requireMap(t, spec["components"], "components")
	schemas := requireMap(t, components["schemas"], "component schemas")
	return requireMap(t, schemas[name], name+" component schema")
}

func parameterByName(t *testing.T, operation map[string]any, name string) map[string]any {
	t.Helper()
	for _, raw := range requireSlice(t, operation["parameters"], "operation parameters") {
		parameter := requireMap(t, raw, "parameter")
		if parameter["name"] == name {
			return parameter
		}
	}
	t.Errorf("parameter %q not found", name)
	return nil
}

func decodeOpenAPIExampleYAML(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &spec))
	return spec
}

func synthesizedExampleStabilityDSL(addMutableField, omitSynthesized bool) func() {
	return func() {
		API("Example Stability", func() {
			if omitSynthesized {
				Meta("openapi:example", "false")
			}
		})
		Service("Examples", func() {
			Method("Mutable", func() {
				Result(func() {
					Attribute("value", String)
					if addMutableField {
						Attribute("count", Int)
					}
				})
				HTTP(func() {
					GET("/mutable")
					Response(StatusOK)
				})
			})
			Method("Stable", func() {
				Payload(func() {
					Attribute("authored_header", String, func() {
						Example("authored-header")
					})
					Attribute("authored_cookie", String, func() {
						Example("authored-cookie")
					})
				})
				Result(func() {
					Attribute("message", String)
					Attribute("detail", String)
				})
				HTTP(func() {
					Header("authored_header:X-Authored")
					Cookie("authored_cookie:authored_cookie")
					GET("/stable")
					Response(StatusOK)
				})
			})
			Method("Authored", func() {
				Result(String, func() {
					Example("authored-example")
				})
				HTTP(func() {
					GET("/authored")
					Response(StatusOK)
				})
			})
		})
	}
}

func patternExampleDSL() {
	Service("Pattern", func() {
		Method("Show", func() {
			Result(func() {
				Attribute("value", String, func() {
					Pattern(`^\S+$`)
				})
			})
			HTTP(func() {
				GET("/pattern")
				Response(StatusOK)
			})
		})
	})
}
