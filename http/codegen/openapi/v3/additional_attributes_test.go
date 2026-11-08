package openapiv3_test

import (
	"encoding/json/v2"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"

	. "github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

type typedWireCase struct {
	wire string
	want bool
}

// TestAdditionalAttributesTypedEmission checks the component of a typed object
// and of a type that extends it: declared properties, the typed policy with its
// validations and required members the object does not declare.
func TestAdditionalAttributesTypedEmission(t *testing.T) {
	for _, version := range []struct{ target, want string }{
		{"3.1", openapiv3.OpenAPICompatibilityVersion}, {"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			spec := renderTypedSpec(t, testdata.AdditionalAttributesTypedDSL, version.target, version.want)
			schemas := typedComponents(spec)
			labels := schemas["Labels"].(map[string]any)
			require.Equal(t, "object", labels["type"])
			require.Equal(t, []any{"id", "count"}, labels["required"], "an undeclared required member stays required")
			require.Equal(t, []string{"id"}, typedPropertyNames(labels), "an undeclared member is not a property")
			additional := labels["additionalProperties"].(map[string]any)
			require.Equal(t, "integer", additional["type"])
			require.Equal(t, float64(0), additional["minimum"])

			tagged := schemas["Tagged"].(map[string]any)
			require.Equal(t, []any{"id", "count"}, tagged["required"])
			require.Equal(t, []string{"id", "total"}, typedPropertyNames(tagged))
			require.Equal(t, additional, tagged["additionalProperties"], "Extend inherits the typed policy")
			total := tagged["properties"].(map[string]any)["total"].(map[string]any)
			require.Equal(t, float64(0), total["minimum"], "the base policy constrains a derived member")

			requireTypedWire(t, spec, "Labels", []typedWireCase{
				{`{"id":"a","count":1}`, true},
				{`{"id":"a","count":1,"other":2}`, true},
				{`{"id":"a"}`, false},
				{`{"id":"a","count":-1}`, false},
				{`{"id":"a","count":1,"other":"x"}`, false},
				{`{"id":1,"count":1}`, false},
			})
			requireTypedWire(t, spec, "Tagged", []typedWireCase{
				{`{"id":"a","count":1,"total":2}`, true},
				{`{"id":"a","count":1,"total":-2}`, false},
				{`{"id":"a","total":2}`, false},
			})
			requireTypedExamplesValid(t, spec)
		})
	}
}

// TestAdditionalAttributesTypedPolicies checks policy values that are named
// types or nullable, and Extend-derived types rendered as allOf references.
func TestAdditionalAttributesTypedPolicies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
		check  func(t *testing.T, spec map[string]any)
	}{
		{"user type policy", func() {
			label := Type("Label", func() {
				Attribute("name", String)
				Required("name")
			})
			typedRecordService(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(label)
				Required("id")
			}))
		}, func(t *testing.T, spec map[string]any) {
			record := typedComponents(spec)["Record"].(map[string]any)
			require.Equal(t, map[string]any{"$ref": "#/components/schemas/Label"}, record["additionalProperties"])
			requireTypedWire(t, spec, "Record", []typedWireCase{
				{`{"id":"a","x":{"name":"n"}}`, true},
				{`{"id":"a","x":{}}`, false},
				{`{"id":"a","x":"n"}`, false},
			})
		}},
		{"nullable policy", func() {
			typedRecordService(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(String, func() {
					Nullable()
				})
			}))
		}, func(t *testing.T, spec map[string]any) {
			requireTypedWire(t, spec, "Record", []typedWireCase{
				{`{"id":"a","x":null}`, true},
				{`{"id":"a","x":"v"}`, true},
				{`{"id":null}`, false},
				{`{"id":"a","x":1}`, false},
			})
		}},
		{"non-nullable policy", func() {
			typedRecordService(Type("Record", func() {
				Attribute("id", String)
				AdditionalAttributes(String)
			}))
		}, func(t *testing.T, spec map[string]any) {
			requireTypedWire(t, spec, "Record", []typedWireCase{
				{`{"id":"a","x":"v"}`, true},
				{`{"id":"a","x":null}`, false},
			})
		}},
		{"Extend as allOf reference", func() {
			base := Type("Base", func() {
				Attribute("id", String)
				AdditionalAttributes(Int, func() {
					Minimum(0)
				})
				Required("id", "count")
			})
			typedRecordService(Type("Derived", func() {
				Meta("openapi:allOf:reference", "true")
				Extend(base)
				Attribute("total", Int)
			}))
		}, func(t *testing.T, spec map[string]any) {
			derived := typedComponents(spec)["Derived"].(map[string]any)
			allOf := derived["allOf"].([]any)
			base := requireMap(t, allOf[0], "base reference")
			require.Equal(t, "#/components/schemas/Base", base["$ref"])
			// Reference-local annotations do not change the inherited policy.
			require.NotContains(t, base, "additionalProperties")
			requireTypedWire(t, spec, "Derived", []typedWireCase{
				{`{"id":"a","count":1,"total":2}`, true},
				{`{"id":"a","count":1,"total":2,"other":3}`, true},
				{`{"id":"a","count":1,"total":-1}`, false},
				{`{"id":"a","total":2}`, false},
				{`{"id":"a","count":1,"other":"x"}`, false},
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := renderTypedSpec(t, tc.design, "3.2", openapiv3.OpenAPIVersion)
			tc.check(t, spec)
			requireTypedExamplesValid(t, spec)
		})
	}
}

// TestAdditionalAttributesTypedIdentity checks that structurally identical
// inline bodies share a component only when their typed policies agree.
func TestAdditionalAttributesTypedIdentity(t *testing.T) {
	body := func(policy any, readOnly bool) func() {
		return func() {
			if readOnly {
				Attribute("id", String, func() {
					Meta("openapi:readOnly", "true")
				})
			}
			Attribute("name", String)
			AdditionalAttributes(policy)
		}
	}
	spec := renderTypedSpec(t, func() {
		API("typed", func() {})
		Service("records", func() {
			for _, method := range []struct {
				name     string
				policy   any
				readOnly bool
			}{
				{"ints", Int, false}, {"moreInts", Int, false}, {"strings", String, false}, {"pruned", Int, true},
			} {
				Method(method.name, func() {
					Payload(body(method.policy, method.readOnly))
					HTTP(func() {
						POST("/" + method.name)
					})
				})
			}
		})
	}, "3.2", openapiv3.OpenAPIVersion)
	refs := make(map[string]string)
	for _, method := range []string{"ints", "moreInts", "strings", "pruned"} {
		operation := spec["paths"].(map[string]any)["/"+method].(map[string]any)["post"].(map[string]any)
		schema := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
		refs[method] = schema["$ref"].(string)
	}
	require.Equal(t, refs["ints"], refs["moreInts"], "identical typed bodies share a component")
	require.NotEqual(t, refs["ints"], refs["strings"], "bodies that differ only in policy are distinct")
	require.NotEqual(t, refs["ints"], refs["pruned"], "a body that forbids a pruned member is distinct")
	requireTypedExamplesValid(t, spec)
}

// TestAdditionalAttributesTypedUsagePruning checks request and response
// variants of typed and closed objects with readOnly and writeOnly members.
// A pruned member stays required nowhere, never falls under the typed policy,
// and the examples of each variant validate against it.
func TestAdditionalAttributesTypedUsagePruning(t *testing.T) {
	spec := renderTypedSpec(t, func() {
		API("typed", func() {})
		record := Type("Record", func() {
			Attribute("id", String, func() {
				Meta("openapi:readOnly", "true")
			})
			Attribute("secret", String, func() {
				Meta("openapi:writeOnly", "true")
			})
			Attribute("label", String)
			AdditionalAttributes(Int)
			Required("id", "secret", "name", "note")
		})
		closed := Type("Closed", func() {
			Attribute("id", String, func() {
				Meta("openapi:readOnly", "true")
			})
			Attribute("name", String)
			Required("id", "name")
			Meta("openapi:additionalProperties", "false")
		})
		Service("records", func() {
			Method("create", func() {
				Payload(func() {
					Attribute("record", record)
					Attribute("closed", closed)
					Required("record", "closed")
				})
				Result(func() {
					Attribute("record", record)
					Attribute("closed", closed)
					Required("record", "closed")
				})
				HTTP(func() {
					POST("/records")
				})
			})
		})
	}, "3.2", openapiv3.OpenAPIVersion)
	schemas := typedComponents(spec)
	request := schemas["RecordRequestBodyRequest"].(map[string]any)
	response := schemas["RecordResponseBodyResponse"].(map[string]any)
	require.Equal(t, []any{"secret", "name", "note"}, request["required"], "undeclared required members survive pruning")
	require.Equal(t, []any{"id", "name", "note"}, response["required"])
	require.Equal(t, map[string]any{"not": map[string]any{}}, request["properties"].(map[string]any)["id"])
	require.Equal(t, map[string]any{"not": map[string]any{}}, response["properties"].(map[string]any)["secret"])
	require.Equal(t, []any{"name"}, schemas["ClosedRequestBodyRequest"].(map[string]any)["required"])

	requireTypedWire(t, spec, "RecordRequestBodyRequest", []typedWireCase{
		{`{"secret":"s","name":1,"note":2}`, true},
		{`{"secret":"s","name":1}`, false},
		{`{"secret":"s","name":1,"note":2,"id":3}`, false},
		{`{"secret":"s","name":1,"note":2,"id":"x"}`, false},
	})
	requireTypedWire(t, spec, "RecordResponseBodyResponse", []typedWireCase{
		{`{"id":"x","name":1,"note":2}`, true},
		{`{"id":"x","name":1,"note":2,"secret":3}`, false},
	})
	requireTypedWire(t, spec, "ClosedRequestBodyRequest", []typedWireCase{
		{`{"name":"n"}`, true},
		{`{"name":"n","id":"x"}`, false},
	})
	requireTypedExamplesValid(t, spec)
}

// TestAdditionalAttributesTypedPathBoundMembers checks that a typed body
// forbids a member bound to a path parameter instead of admitting it as an
// undeclared member or requiring it.
func TestAdditionalAttributesTypedPathBoundMembers(t *testing.T) {
	spec := renderTypedSpec(t, func() {
		API("typed", func() {})
		labels := Type("Labels", func() {
			Attribute("id", String)
			AdditionalAttributes(Int)
			Required("id", "count")
		})
		Service("records", func() {
			Method("create", func() {
				Payload(labels)
				HTTP(func() {
					POST("/p/{id}")
				})
			})
		})
	}, "3.2", openapiv3.OpenAPIVersion)
	var component string
	for name := range typedComponents(spec) {
		if strings.HasPrefix(name, "Labels") {
			component = name
		}
	}
	require.NotEmpty(t, component)
	body := typedComponents(spec)[component].(map[string]any)
	require.Equal(t, []any{"count"}, body["required"])
	require.Equal(t, map[string]any{"not": map[string]any{}}, body["properties"].(map[string]any)["id"])
	requireTypedWire(t, spec, component, []typedWireCase{
		{`{"count":1}`, true},
		{`{"count":1,"other":2}`, true},
		{`{"count":1,"id":2}`, false},
		{`{}`, false},
	})
	requireTypedExamplesValid(t, spec)
}

func typedRecordService(record any) {
	API("typed", func() {})
	Service("records", func() {
		Method("create", func() {
			Payload(func() {
				Attribute("record", record)
			})
			Result(record)
			HTTP(func() {
				POST("/records")
			})
		})
	})
}

func renderTypedSpec(t *testing.T, design func(), target, want string) map[string]any {
	t.Helper()
	artifacts := renderOpenAPIArtifactsForVersion(t, design, target, want)
	var spec map[string]any
	require.NoError(t, json.Unmarshal(artifacts.JSON, &spec))
	return spec
}

func typedComponents(spec map[string]any) map[string]any {
	return spec["components"].(map[string]any)["schemas"].(map[string]any)
}

func typedPropertyNames(schema map[string]any) []string {
	properties, _ := schema["properties"].(map[string]any)
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func requireTypedWire(t *testing.T, spec map[string]any, component string, cases []typedWireCase) {
	t.Helper()
	require.Contains(t, typedComponents(spec), component)
	root := map[string]any{"components": spec["components"], "$ref": "#/components/schemas/" + component}
	for _, value := range cases {
		result, err := gojsonschema.Validate(gojsonschema.NewGoLoader(root), gojsonschema.NewStringLoader(value.wire))
		require.NoError(t, err)
		require.Equal(t, value.want, result.Valid(), "%s: %s: %v", component, value.wire, result.Errors())
	}
}

// requireTypedExamplesValid validates every component and media example
// against the schema that carries it.
func requireTypedExamplesValid(t *testing.T, spec map[string]any) {
	t.Helper()
	validate := func(location string, schema map[string]any, example any) {
		t.Helper()
		root := map[string]any{"components": spec["components"]}
		for key, value := range schema {
			if key != "example" && key != "examples" {
				root[key] = value
			}
		}
		result, err := gojsonschema.Validate(gojsonschema.NewGoLoader(root), gojsonschema.NewGoLoader(example))
		require.NoError(t, err, location)
		require.True(t, result.Valid(), "%s example %v: %v", location, example, result.Errors())
	}
	for name, raw := range typedComponents(spec) {
		schema := raw.(map[string]any)
		if example, ok := schema["example"]; ok {
			validate("component "+name, schema, example)
		}
	}
	for path, item := range spec["paths"].(map[string]any) {
		for method, raw := range item.(map[string]any) {
			operation, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			var contents []map[string]any
			if body, ok := operation["requestBody"].(map[string]any); ok {
				contents = append(contents, body["content"].(map[string]any))
			}
			if responses, ok := operation["responses"].(map[string]any); ok {
				for _, response := range responses {
					if content, ok := response.(map[string]any)["content"].(map[string]any); ok {
						contents = append(contents, content)
					}
				}
			}
			for _, content := range contents {
				for media, raw := range content {
					mediaType := raw.(map[string]any)
					schema, _ := mediaType["schema"].(map[string]any)
					if example, ok := mediaType["example"]; ok && schema != nil {
						validate(strings.Join([]string{method, path, media}, " "), schema, example)
					}
				}
			}
		}
	}
}

func TestRequiredAdditionalLiteralKeysEmission(t *testing.T) {
	for _, version := range []struct{ target, want string }{
		{"3.1", openapiv3.OpenAPICompatibilityVersion}, {"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			spec := renderTypedSpec(t, testdata.AdditionalCompositeDSL, version.target, version.want)
			literal := typedComponents(spec)["LiteralKeys"].(map[string]any)
			require.Equal(t, []string{"a"}, typedPropertyNames(literal))
			require.Equal(t, []any{"a:b", "a,b", "", "-", "quote\"slash\\", "line\nend", "日本語", "AdditionalAttributes", "foo", "Foo"}, literal["required"])
			requireTypedWire(t, spec, "LiteralKeys", []typedWireCase{
				{`{"a":42,"a:b":0,"a,b":1,"":2,"-":3,"quote\"slash\\":4,"line\nend":5,"日本語":6,"AdditionalAttributes":7,"foo":8,"Foo":9}`, true},
				{`{"a":42,"a,b":1,"":2,"-":3,"quote\"slash\\":4,"line\nend":5,"日本語":6,"AdditionalAttributes":7,"foo":8,"Foo":9}`, false},
			})
		})
	}
}
