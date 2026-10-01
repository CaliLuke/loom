package openapiv3_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/CaliLuke/loom/internal/schematest"
)

type renderedExampleInstance struct {
	label     string
	schema    map[string]any
	instance  any
	wantValid bool
}

func TestRenderedSharedValueExamplesMatchTheirSchemas(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "1" {
		t.Skip("independent schema engine runs in make openapi-contract")
	}

	var instances []renderedExampleInstance
	for _, version := range []struct {
		target, number string
	}{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			for _, fixture := range []struct {
				name string
				dsl  func()
				add  func(*testing.T, string, map[string]any, *[]renderedExampleInstance)
			}{
				{"nested-unions", testdata.TypeIdentityDSL, addNestedUnionExampleInstances},
				{"collection-values", testdata.CollectionEnumDSL, addCollectionExampleInstances},
				{"streaming-surfaces", testdata.StreamingPartialExamplesDSL, addStreamingExampleInstances},
				{"named-request-body", testdata.OpenAPINamedRequestBodyExamplesDSL, addNamedRequestBodyExampleInstances},
				{"named-wrapper-body", testdata.OpenAPIExplicitBodyWrapperExamplesDSL, addNamedRequestBodyExampleInstances},
				{"precision", renderedSharedValuePrecisionDSL, addPrecisionExampleInstances},
			} {
				t.Run(fixture.name, func(t *testing.T) {
					artifacts := renderOpenAPIArtifactsForVersion(t, fixture.dsl, version.target, version.number)
					if fixture.name == "precision" {
						requireRenderedNumber(t, artifacts.JSON, artifacts.YAML, "signed", "9007199254740993")
						requireRenderedNumber(t, artifacts.JSON, artifacts.YAML, "unsigned", "18446744073709551615")
						requireRenderedNumber(t, artifacts.JSON, artifacts.YAML, "decimal", "1.2345679")
						require.Regexp(t, `"json"\s*:\s*"\{\\"value\\":9007199254740993\}"`, string(artifacts.JSON))
					}

					for _, rendered := range []struct {
						format   string
						document map[string]any
					}{
						{"json", decodeOpenAPIJSON(t, artifacts.JSON)},
						{"yaml", decodeOpenAPIExampleYAML(t, artifacts.YAML)},
					} {
						t.Run(rendered.format, func(t *testing.T) {
							fixture.add(t, version.target+"/"+rendered.format+"/"+fixture.name, rendered.document, &instances)
						})
					}
				})
			}
		})
	}

	batches := make([]schematest.Batch, 0, len(instances))
	for _, instance := range instances {
		batches = append(batches, schematest.Batch{
			Schema:    instance.schema,
			Instances: []any{instance.instance},
		})
	}
	results := schematest.New(t).Check(t, batches)
	for index, instance := range instances {
		result := results[index][0]
		require.Equal(t, instance.wantValid, result.Valid, "%s: %s", instance.label, result.Errors)
	}
}

func requireRenderedNumber(t *testing.T, jsonDocument, yamlDocument []byte, field, value string) {
	t.Helper()
	require.Regexp(t, regexp.MustCompile(`"`+regexp.QuoteMeta(field)+`"\s*:\s*`+regexp.QuoteMeta(value)+`(?:\s*[,}])`), string(jsonDocument))
	require.Regexp(t, regexp.MustCompile(`(?m)^\s*`+regexp.QuoteMeta(field)+`:\s*`+regexp.QuoteMeta(value)+`\s*$`), string(yamlDocument))
}

func addNestedUnionExampleInstances(
	t *testing.T,
	label string,
	document map[string]any,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	t.Run("component-record", func(t *testing.T) {
		record := requireComponentSchema(t, document, "Record")
		recordExample, ok := record["example"]
		require.True(t, ok, label+" record example")
		addRenderedExampleInstance(t, label+"/component-record", document, record, recordExample, true, instances)
	})
	t.Run("invalid-discriminator", func(t *testing.T) {
		record := requireComponentSchema(t, document, "Record")
		recordExample, ok := record["example"]
		require.True(t, ok, label+" record example")
		mutated := cloneRenderedExampleValue(recordExample)
		block := requireMap(t, requireMap(t, mutated, label+" mutated record")["block"], label+" outer union")
		inner := requireMap(t, block["value"], label+" inner union")
		inner["type"] = "not-a-rendered-branch"
		addRenderedExampleInstance(t, label+"/invalid-discriminator", document, record, mutated, false, instances)
	})

	for _, path := range []string{"/put", "/puts"} {
		t.Run(path, func(t *testing.T) {
			operation := requireOperation(t, document, path, "post")
			responses := requireMap(t, operation["responses"], label+path+" responses")
			response := resolveRenderedLocalObject(t, document, requireMap(t, responses["200"], label+path+" response"))
			content := requireMap(t, response["content"], label+path+" response content")
			for contentType, raw := range content {
				media := requireMap(t, raw, label+path+" "+contentType+" response media")
				addRenderedMediaExample(t, label+path+"/response/"+contentType, document, media, instances)
			}
		})
	}
}

func addCollectionExampleInstances(
	t *testing.T,
	label string,
	document map[string]any,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	for _, path := range []string{"/records", "/precise", "/labels", "/bytes", "/nested"} {
		t.Run(path, func(t *testing.T) {
			operation := requireOperation(t, document, path, "post")
			t.Run("request", func(t *testing.T) {
				addRenderedMediaExample(t, label+path+"/request", document, requestMediaType(t, document, operation), instances)
			})
			t.Run("response", func(t *testing.T) {
				addRenderedMediaExample(t, label+path+"/response", document, requireResponseMediaType(t, operation, "application/json"), instances)
			})
		})
	}
}

func addStreamingExampleInstances(
	t *testing.T,
	label string,
	document map[string]any,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	t.Run("chunks-parameter", func(t *testing.T) {
		metadata := requireOperation(t, document, "/metadata", "get")
		addRenderedObjectExample(t, label+"/chunks-parameter", document, parameterByName(t, metadata, "chunks"), instances)
	})
	t.Run("checksum-header", func(t *testing.T) {
		metadata := requireOperation(t, document, "/metadata", "get")
		responses := requireMap(t, metadata["responses"], label+" metadata responses")
		response := resolveRenderedLocalObject(t, document, requireMap(t, responses["200"], label+" metadata response"))
		headers := requireMap(t, response["headers"], label+" metadata headers")
		checksum := resolveRenderedLocalObject(t, document, requireMap(t, headers["X-Checksum"], label+" checksum header"))
		addRenderedObjectExample(t, label+"/checksum-header", document, checksum, instances)
	})
	t.Run("project-id", func(t *testing.T) {
		projectSocket := requireOperation(t, document, "/ws/projects/{projectID}", "get")
		addRenderedObjectExample(t, label+"/project-id", document, parameterByName(t, projectSocket, "projectID"), instances)
	})
	for _, name := range []string{"attachment", "note"} {
		t.Run("schema-"+name, func(t *testing.T) {
			projectSocket := requireOperation(t, document, "/ws/projects/{projectID}", "get")
			async := requireMap(t, projectSocket[openAPIAsyncExtension], label+" async extension")
			messages := requireMap(t, async["messages"], label+" async messages")
			outbound := requireMap(t, messages["outbound"], label+" outbound message")
			envelope := requireMap(t, outbound["schema"], label+" outbound schema")
			properties := requireMap(t, envelope["properties"], label+" envelope properties")
			schema := requireMap(t, properties[name], label+" "+name+" schema")
			example, ok := schema["example"]
			require.True(t, ok, label+" "+name+" example")
			addRenderedExampleInstance(t, label+"/schema-"+name, document, schema, example, true, instances)
		})
	}
}

func addNamedRequestBodyExampleInstances(
	t *testing.T,
	label string,
	document map[string]any,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	for _, name := range []string{"simple", "advanced"} {
		t.Run(name, func(t *testing.T) {
			operation := requireOperation(t, document, "/search", "post")
			media := requestMediaType(t, document, operation)
			schema := requireMap(t, media["schema"], label+" request schema")
			examples := requireMap(t, media["examples"], label+" named examples")
			example := resolveExample(t, document, requireMap(t, examples[name], label+" "+name+" example"))
			value, ok := example["value"]
			require.True(t, ok, label+" "+name+" value")
			addRenderedExampleInstance(t, label+"/named-"+name, document, schema, value, true, instances)
		})
	}
}

func addPrecisionExampleInstances(
	t *testing.T,
	label string,
	document map[string]any,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	media := requireResponseMediaType(t, requireOperation(t, document, "/precision", "get"), "application/json")
	addRenderedMediaExample(t, label+"/response", document, media, instances)
	example := requireMap(t, media["example"], label+" precision example")
	require.Equal(t, `{"value":9007199254740993}`, example["json"], label)
}

func addRenderedMediaExample(
	t *testing.T,
	label string,
	document map[string]any,
	media map[string]any,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	schema := requireMap(t, media["schema"], label+" schema")
	example, ok := media["example"]
	require.True(t, ok, label+" example")
	addRenderedExampleInstance(t, label, document, schema, example, true, instances)
}

func addRenderedObjectExample(
	t *testing.T,
	label string,
	document map[string]any,
	object map[string]any,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	schema := requireMap(t, object["schema"], label+" schema")
	example, ok := object["example"]
	require.True(t, ok, label+" example")
	addRenderedExampleInstance(t, label, document, schema, example, true, instances)
}

func addRenderedExampleInstance(
	t *testing.T,
	label string,
	document map[string]any,
	schema map[string]any,
	instance any,
	wantValid bool,
	instances *[]renderedExampleInstance,
) {
	t.Helper()
	components := requireMap(t, document["components"], label+" components")
	schemas := requireMap(t, components["schemas"], label+" component schemas")
	*instances = append(*instances, renderedExampleInstance{
		label: label,
		schema: map[string]any{
			"allOf":      []any{schema},
			"components": map[string]any{"schemas": schemas},
		},
		instance:  instance,
		wantValid: wantValid,
	})
}

func requestMediaType(
	t *testing.T,
	document map[string]any,
	operation map[string]any,
) map[string]any {
	t.Helper()
	request := resolveRenderedLocalObject(t, document, requireMap(t, operation["requestBody"], "request body"))
	content := requireMap(t, request["content"], "request content")
	return requireMap(t, content["application/json"], "request media type")
}

func resolveRenderedLocalObject(t *testing.T, document, object map[string]any) map[string]any {
	t.Helper()
	ref, _ := object["$ref"].(string)
	if ref == "" {
		return object
	}
	require.True(t, strings.HasPrefix(ref, "#/"), "unexpected external reference %q", ref)
	var current any = document
	for _, raw := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		current = requireMap(t, current, "reference parent")[token]
	}
	return requireMap(t, current, "referenced object")
}

func cloneRenderedExampleValue(value any) any {
	switch actual := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(actual))
		for name, member := range actual {
			cloned[name] = cloneRenderedExampleValue(member)
		}
		return cloned
	case []any:
		cloned := make([]any, len(actual))
		for index, member := range actual {
			cloned[index] = cloneRenderedExampleValue(member)
		}
		return cloned
	default:
		return value
	}
}

func renderedSharedValuePrecisionDSL() {
	Service("precision", func() {
		Method("show", func() {
			NoSecurity()
			Result(func() {
				Attribute("signed", Int64, func() {
					Example(int64(9007199254740993))
				})
				Attribute("unsigned", UInt64, func() {
					Example(^uint64(0))
				})
				Attribute("decimal", Float32, func() {
					Example(float32(1.23456789))
				})
				Attribute("json", String, func() {
					Example(`{"value":9007199254740993}`)
				})
				Required("signed", "unsigned", "decimal", "json")
			})
			HTTP(func() {
				GET("/precision")
				Response(StatusOK)
			})
		})
	})
}
