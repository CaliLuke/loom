package openapiv3_test

import (
	"encoding/json/v2"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/stretchr/testify/require"
)

func TestHTTPBodyContributorResponsePolicy(t *testing.T) {
	for _, responseFirst := range []bool{false, true} {
		name := "request_first"
		if responseFirst {
			name = "response_first"
		}
		t.Run(name, func(t *testing.T) {
			bodyContributorResponsePolicy(t, responseFirst, true)
		})
	}
}

func TestHTTPBodyContributorFullJSONResponsePolicy(t *testing.T) {
	for _, responseFirst := range []bool{false, true} {
		name := "request_first"
		if responseFirst {
			name = "response_first"
		}
		t.Run(name, func(t *testing.T) {
			bodyContributorResponsePolicy(t, responseFirst, false)
		})
	}
}

func bodyContributorResponsePolicy(t *testing.T, responseFirst, partial bool) {
	root := httpgen.RunHTTPDSL(t, func() {
		API("response contributor policy", func() {
			Meta("openapi:version", "3.1")
		})
		parent := Type("Parent", func() {
			Attribute("id", String)
			Attribute("count", Int, func() {
				Default(0)
				Meta("struct:tag:json", "count,omitzero")
			})
			Example(map[string]any{"count": 0})
		})
		child := Type("Child", func() {
			Extend(parent)
			Meta("openapi:allOf:reference", "true")
			Attribute("thumb", String)
			Example(map[string]any{"count": 0, "thumb": "x"})
		})
		Service("policy", func() {
			request := func() {
				Method("request", func() {
					Payload(child)
					HTTP(func() {
						POST("/request")
						if partial {
							Header("id")
						}
					})
				})
			}
			response := func() {
				Method("response", func() {
					Result(child)
					HTTP(func() {
						GET("/response")
						Response(StatusOK, func() {
							if partial {
								Header("id")
							}
						})
					})
				})
			}
			if responseFirst {
				response()
				request()
			} else {
				request()
				response()
			}
		})
	})
	t.Log("DSL_ADMITTED")
	prepared, err := representation.PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	var request, response *transportir.ValueTarget
	for _, endpoint := range prepared.Endpoints {
		if endpoint.MethodName == "request" {
			request = endpoint.Request.BodyValue
		}
		if endpoint.MethodName == "response" {
			response = endpoint.Response.Responses[0].BodyValue
		}
	}
	require.NotNil(t, request)
	require.NotNil(t, response)
	observe := func(name string, target *transportir.ValueTarget) map[string]any {
		require.NoError(t, target.Error)
		node := target.Plan.Root()
		for node.Underlying().Valid() {
			node = node.Underlying()
		}
		for _, member := range node.Members() {
			if member.Name == "count" {
				t.Logf("%s count presence=%v", name, member.Presence)
			}
		}
		context := target.Source.Context
		value := context.Resolve(target.ExampleOccurrence, context.SupplyValue(expr.ValueInput{Raw: map[string]any{"count": 0, "thumb": "x"}}), expr.ValueRoleExample)
		projected := context.ProjectJSON(value, target.ExamplePlan)
		require.NoError(t, projected.Err())
		raw, ok := projected.JSON()
		require.True(t, ok)
		t.Logf("%s ordinary example=%s", name, raw)
		var result map[string]any
		require.NoError(t, json.Unmarshal(raw, &result))
		return result
	}
	require.Contains(t, observe("request", request), "count", "request presence retains supplied zero")
	require.NotContains(t, observe("response", response), "count", "authored omitzero omits native response zero")
	spec := bodyProjectionSpec(t, root)
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	paths := requireMap(t, spec["paths"], "paths")
	requestOperation := requireMap(t, requireMap(t, paths["/request"], "request path")["post"], "request operation")
	responseOperation := requireMap(t, requireMap(t, paths["/response"], "response path")["get"], "response operation")
	contentSchema := func(media any) map[string]any {
		content := requireMap(t, requireMap(t, media, "body")["content"], "content")
		return resolveComponent(t, schemas, requireMap(t, content["application/json"], "JSON")["schema"])
	}
	requestBody := contentSchema(requestOperation["requestBody"])
	responses := requireMap(t, responseOperation["responses"], "responses")
	responseBody := contentSchema(responses["200"])
	contributor := func(name string, body map[string]any) (map[string]any, any) {
		allOf := requireSlice(t, body["allOf"], name+" allOf")
		require.NotEmpty(t, allOf)
		parent := resolveComponent(t, schemas, allOf[0])
		t.Logf("%s contributor ref=%v example=%v", name, allOf[0], parent["example"])
		return requireMap(t, parent["example"], name+" contributor example"), allOf[0]
	}
	requestExample, requestRef := contributor("request", requestBody)
	responseExample, responseRef := contributor("response", responseBody)
	require.Contains(t, requestExample, "count", "request contributor retains supplied zero")
	if _, found := responseExample["count"]; found {
		t.Errorf("response contributor retained count=0 despite ordinary response omission; requestRef=%v responseRef=%v", requestRef, responseRef)
	}
}
