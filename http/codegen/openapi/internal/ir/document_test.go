package ir

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/http/codegen/openapi/v3/testdata/dsls"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestBuildDocumentIncludesRequestBodyAndResponses(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "request_object_body"
	)

	root := codegen.RunDSL(t, dsls.RequestObjectBody(serviceName, methodName))
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	path := root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].FullPaths()[0]
	operation := doc.Paths[path].Operations["POST"]
	require.NotNil(t, operation)
	require.NotNil(t, operation.RequestBody)
	require.NotNil(t, operation.RequestBody.Value)
	require.True(t, operation.RequestBody.Value.Required)
	require.Contains(t, operation.RequestBody.Value.Content, "application/json")
	require.NotEmpty(t, operation.RequestBody.Value.Content["application/json"].Schema.Ref)
	require.Contains(t, operation.Responses, "204")
	require.NotNil(t, operation.Responses["204"])
	require.NotNil(t, operation.Responses["204"].Value)
	require.Equal(t, "No Content response.", operation.Responses["204"].Value.Description)
}

func TestBuildDocumentNullableAliasComponentReferencesUnderlyingType(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		widget := dsl.Type("Widget", func() {
			dsl.Meta("openapi:typename:canonical", "true")
			dsl.Attribute("name", dsl.String)
			dsl.Required("name")
		})
		nullableWidget := dsl.Type("NullableWidget", widget, func() {
			dsl.Meta("openapi:typename:canonical", "true")
			dsl.Nullable()
		})
		dsl.Service("Widgets", func() {
			dsl.Method("getWidget", func() {
				dsl.Result(nullableWidget)
				dsl.HTTP(func() {
					dsl.GET("/widget")
					dsl.Response(dsl.StatusOK)
				})
			})
		})
	})

	doc := BuildDocument(root.API, root.Types, root.ResultTypes)

	component := doc.Components.Schemas["NullableWidget"]
	require.NotNil(t, component)
	require.Len(t, component.AnyOf, 2)
	require.Equal(t, "#/components/schemas/Widget", component.AnyOf[0].Ref)
	require.Equal(t, "null", component.AnyOf[1].Type)
}

func TestOpenAPIExampleValueMatchesUntaggedUnionValidation(t *testing.T) {
	start := &expr.UserTypeExpr{
		TypeName: "ExampleStart",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{
				Name: "kind",
				Attribute: &expr.AttributeExpr{
					Type:       expr.String,
					Validation: &expr.ValidationExpr{Values: []any{"start"}},
				},
			}},
			Validation: &expr.ValidationExpr{Required: []string{"kind"}},
		},
	}
	stop := &expr.UserTypeExpr{
		TypeName: "ExampleStop",
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{{
				Name: "kind",
				Attribute: &expr.AttributeExpr{
					Type:       expr.String,
					Validation: &expr.ValidationExpr{Values: []any{"stop"}},
				},
			}},
			Validation: &expr.ValidationExpr{Required: []string{"kind"}},
		},
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		TypeName: "ExampleCommand",
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "start", Attribute: &expr.AttributeExpr{Type: start}},
			{Name: "stop", Attribute: &expr.AttributeExpr{Type: stop}},
		},
	}}

	value, ok := OpenAPIExampleValue(attribute, map[string]any{"kind": "start"})
	require.True(t, ok)
	require.Equal(t, map[string]any{"kind": "start"}, value)
}

func TestOpenAPIExampleValueMatchesExactUnsignedUnionEnums(t *testing.T) {
	t.Parallel()

	maximum := ^uint64(0)
	branch := func(value uint64) *expr.AttributeExpr {
		return &expr.AttributeExpr{
			Type: &expr.Object{{
				Name: "kind",
				Attribute: &expr.AttributeExpr{
					Type:       expr.UInt64,
					Validation: &expr.ValidationExpr{Values: []any{value}},
				},
			}},
			Validation: &expr.ValidationExpr{Required: []string{"kind"}},
		}
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "lower", Attribute: branch(maximum - 1)},
			{Name: "upper", Attribute: branch(maximum)},
		},
	}}
	example := map[string]any{"kind": maximum}

	value, ok := OpenAPIExampleValue(attribute, example)
	require.True(t, ok)
	require.Equal(t, example, value)
}

func TestOpenAPIExampleValueMaterializesArbitraryJSONForAllFormats(t *testing.T) {
	t.Parallel()

	attribute := &expr.AttributeExpr{Type: expr.Any}
	raw := jsontext.Value(`{"flag":true,"nested":[false,9007199254740993]}`)

	rawValue, ok := OpenAPIExampleValue(attribute, raw)
	require.True(t, ok)
	require.Equal(t, map[string]any{
		"flag": true,
		"nested": []any{
			false,
			openAPIJSONNumber("9007199254740993"),
		},
	}, rawValue)
	jsonValue, err := jsonv2.Marshal(rawValue, jsonv2.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, `{"flag":true,"nested":[false,9007199254740993]}`, string(jsonValue))
	yamlValue, err := yaml.Marshal(rawValue)
	require.NoError(t, err)
	require.Equal(t, "flag: true\nnested:\n    - false\n    - 9007199254740993\n", string(yamlValue))

	bytesValue, ok := OpenAPIExampleValue(attribute, []byte{1, 2})
	require.True(t, ok)
	require.Equal(t, "AQI=", bytesValue)
}

func TestOpenAPIExampleValueRejectsTypedNilContainers(t *testing.T) {
	t.Parallel()

	array := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}
	_, ok := OpenAPIExampleValue(array, []string(nil))
	require.False(t, ok)

	objectMap := &expr.AttributeExpr{Type: &expr.Map{ElemType: &expr.AttributeExpr{Type: expr.String}}}
	_, ok = OpenAPIExampleValue(objectMap, map[string]string(nil))
	require.False(t, ok)
}

func TestOpenAPIExampleValuePreservesRawJSONNull(t *testing.T) {
	t.Parallel()

	value, ok := OpenAPIExampleValue(&expr.AttributeExpr{Type: expr.Any}, jsontext.Value(`null`))
	require.True(t, ok)
	require.IsType(t, NullExample{}, value)
	jsonValue, err := jsonv2.Marshal(value)
	require.NoError(t, err)
	require.Equal(t, "null", string(jsonValue))
	yamlValue, err := yaml.Marshal(value)
	require.NoError(t, err)
	require.Equal(t, "null\n", string(yamlValue))
}

func TestOpenAPIExampleValuePreservesNestedAnySemantics(t *testing.T) {
	t.Parallel()

	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "raw", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		{Name: "binary", Attribute: &expr.AttributeExpr{Type: expr.Any}},
	}}
	value, ok := OpenAPIExampleValue(attribute, map[string]any{
		"raw":    jsontext.Value(`9007199254740993`),
		"binary": []byte("abc"),
	})
	require.True(t, ok)
	object := value.(map[string]any)
	require.Equal(t, "YWJj", object["binary"])
	raw, err := jsonv2.Marshal(object["raw"])
	require.NoError(t, err)
	require.Equal(t, `9007199254740993`, string(raw))

	schema := NewAnalyzer(expr.NewRandom("raw-enum"), false).AnalyzeSchema(&expr.AttributeExpr{
		Type:       expr.Any,
		Validation: &expr.ValidationExpr{Values: []any{jsontext.Value(`9007199254740993`)}},
	})
	require.Len(t, schema.Enum, 1)
	raw, err = jsonv2.Marshal(schema.Enum[0])
	require.NoError(t, err)
	require.Equal(t, `9007199254740993`, string(raw))
}

func TestOpenAPIExampleValueProjectsOnlyMatchingUntaggedBranch(t *testing.T) {
	t.Parallel()

	branch := func(kind string, suppressPayload bool) *expr.AttributeExpr {
		payloadMeta := expr.MetaExpr(nil)
		if suppressPayload {
			payloadMeta = expr.MetaExpr{"openapi:generate": []string{"false"}}
		}
		return &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "kind", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{kind}}}},
				{Name: "payload", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: payloadMeta}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"kind", "payload"}},
		}
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "start", Attribute: branch("start", true)},
			{Name: "stop", Attribute: branch("stop", false)},
		},
	}}

	value, ok := OpenAPIExampleValue(attribute, map[string]any{"kind": "stop", "payload": "keep"})
	require.True(t, ok)
	require.Equal(t, map[string]any{"kind": "stop", "payload": "keep"}, value)

	value, ok = OpenAPIExampleValue(attribute, map[string]any{"kind": "start", "payload": "remove"})
	require.True(t, ok)
	require.Equal(t, map[string]any{"kind": "start"}, value)
}

func TestBuildDocumentComposesRepeatedHTTPBlocks(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("svc", func() {
			dsl.HTTP(func() {
				dsl.Path("/base")
			})
			dsl.Error("boom", dsl.ErrorResult, "boom")
			dsl.HTTP(func() {
				dsl.Response("boom", dsl.StatusConflict)
			})
			dsl.Method("get", func() {
				dsl.Result(dsl.String)
				dsl.HTTP(func() {
					dsl.GET("/thing")
				})
			})
		})
	})

	doc := BuildDocument(root.API, root.Types, root.ResultTypes)
	operation := doc.Paths["/base/thing"].Operations["GET"]
	require.NotNil(t, operation)
	require.Contains(t, operation.Responses, "409")
}

func TestOpenAPIExampleValueAcceptsNestedStructuredUnionFields(t *testing.T) {
	t.Parallel()

	branch := func(kind string) *expr.AttributeExpr {
		details := &expr.UserTypeExpr{
			TypeName: "Details" + kind,
			UID:      "Details" + kind,
			AttributeExpr: &expr.AttributeExpr{
				Type: &expr.Object{{
					Name: "kind",
					Attribute: &expr.AttributeExpr{
						Type:       expr.String,
						Validation: &expr.ValidationExpr{Values: []any{kind}},
					},
				}},
				Validation: &expr.ValidationExpr{Required: []string{"kind"}},
			},
		}
		return &expr.AttributeExpr{
			Type:       &expr.Object{{Name: "details", Attribute: &expr.AttributeExpr{Type: details}}},
			Validation: &expr.ValidationExpr{Required: []string{"details"}},
		}
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "start", Attribute: branch("start")},
			{Name: "stop", Attribute: branch("stop")},
		},
	}}
	example := map[string]any{"details": map[string]any{"kind": "start"}}

	value, ok := OpenAPIExampleValue(attribute, example)
	require.True(t, ok)
	require.Equal(t, example, value)
}
func TestOpenAPIExampleValueAcceptsNestedOpenObjectUnionFields(t *testing.T) {
	t.Parallel()

	details := &expr.AttributeExpr{
		Type: &expr.Object{{
			Name:      "kind",
			Attribute: &expr.AttributeExpr{Type: expr.String},
		}},
		Validation: &expr.ValidationExpr{Required: []string{"kind"}},
		Meta:       expr.MetaExpr{"openapi:additionalProperties": []string{"true"}},
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		TypeKey:  "type",
		ValueKey: "value",
		Values: []*expr.NamedAttributeExpr{
			{Name: "structured", Attribute: &expr.AttributeExpr{
				Type:       &expr.Object{{Name: "details", Attribute: details}},
				Validation: &expr.ValidationExpr{Required: []string{"details"}},
			}},
			{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
	}}
	example := map[string]any{"details": map[string]any{"kind": "event", "extra": true}}

	value, ok := OpenAPIExampleValue(attribute, example)
	require.True(t, ok)
	require.Equal(t, map[string]any{
		"type":  "structured",
		"value": example,
	}, value)
}

func TestOpenAPIExampleValueAcceptsNullableComplexUnionFields(t *testing.T) {
	t.Parallel()

	branch := func(kind string, payload *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "kind", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{kind}}}},
				{Name: "payload", Attribute: payload},
			},
			Validation: &expr.ValidationExpr{Required: []string{"kind", "payload"}},
		}
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "nullable", Attribute: branch("nullable", &expr.AttributeExpr{
				Type:     &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
				Nullable: true,
			})},
			{Name: "text", Attribute: branch("text", &expr.AttributeExpr{Type: expr.String})},
		},
	}}
	example := map[string]any{"kind": "nullable", "payload": nil}

	value, ok := OpenAPIExampleValue(attribute, example)
	require.True(t, ok)
	require.Equal(t, example, value)
}

func TestOpenAPIExampleValueMatchesRecursiveNumericEnums(t *testing.T) {
	t.Parallel()

	branch := func(payload *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{
			Type:       &expr.Object{{Name: "payload", Attribute: payload}},
			Validation: &expr.ValidationExpr{Required: []string{"payload"}},
		}
	}
	attribute := &expr.AttributeExpr{Type: &expr.Union{
		Untagged: true,
		Values: []*expr.NamedAttributeExpr{
			{Name: "structured", Attribute: branch(&expr.AttributeExpr{
				Type:       expr.Any,
				Validation: &expr.ValidationExpr{Values: []any{map[string]any{"n": int(1)}}},
			})},
			{Name: "text", Attribute: branch(&expr.AttributeExpr{Type: expr.String})},
		},
	}}
	example := map[string]any{"payload": map[string]any{"n": uint64(1)}}

	value, ok := OpenAPIExampleValue(attribute, example)
	require.True(t, ok)
	require.Equal(t, example, value)
	require.True(t, exampleValuesEqual(openAPIJSONNumber("18446744073709551615"), ^uint64(0)))
}

func TestBuildDocumentUsesExplicitRequestBodyDescriptionMeta(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "explicit_request_body_description"
	)

	root := codegen.RunDSL(t, func() {
		bodyType := dsl.Type("NamedBody", func() {
			dsl.Description("Type description that should not implicitly become the request body description.")
			dsl.Meta("openapi:description:requestBody", "Human-friendly request body description.")
			dsl.Attribute("name", dsl.String)
		})
		dsl.Service(serviceName, func() {
			dsl.Method(methodName, func() {
				dsl.Payload(bodyType)
				dsl.HTTP(func() {
					dsl.POST("/")
					dsl.Body(bodyType)
				})
			})
		})
	})
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	path := root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].FullPaths()[0]
	operation := doc.Paths[path].Operations["POST"]
	require.NotNil(t, operation)
	require.NotNil(t, operation.RequestBody)
	require.NotNil(t, operation.RequestBody.Value)
	require.Equal(t, "Human-friendly request body description.", operation.RequestBody.Value.Description)
}

func TestBuildDocumentPublishesDocumentedRawRequestBodies(t *testing.T) {
	root := codegen.RunDSL(t, testdata.RawRequestBodyOpenAPIDSL)
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	binary := doc.Paths["/uploads/{id}"].Operations["POST"]
	require.NotNil(t, binary)
	require.NotNil(t, binary.RequestBody)
	require.NotNil(t, binary.RequestBody.Value)
	require.True(t, binary.RequestBody.Value.Required)
	require.Equal(t, "Binary archive streamed directly to the service.", binary.RequestBody.Value.Description)
	binaryMedia := binary.RequestBody.Value.Content["application/octet-stream"]
	require.NotNil(t, binaryMedia)
	require.Equal(t, "string", binaryMedia.Schema.Type)
	require.Equal(t, "binary", binaryMedia.Schema.Format)
	require.Equal(t, "archive", binaryMedia.Example)
	require.Len(t, binary.Parameters, 4)
	require.Equal(t, []map[string][]string{{"upload_key": {}}}, binary.Security)

	text := doc.Paths["/imports"].Operations["POST"]
	require.NotNil(t, text)
	require.NotNil(t, text.RequestBody)
	require.NotNil(t, text.RequestBody.Value)
	require.False(t, text.RequestBody.Value.Required)
	require.Equal(t, "Optional newline-delimited import commands.", text.RequestBody.Value.Description)
	textMedia := text.RequestBody.Value.Content["text/plain; charset=utf-8"]
	require.NotNil(t, textMedia)
	require.Equal(t, "string", textMedia.Schema.Type)
	require.Empty(t, textMedia.Schema.Format)
	require.Equal(t, "create widget", textMedia.Example)

	manifest := doc.Paths["/manifests"].Operations["POST"]
	require.NotNil(t, manifest)
	require.NotNil(t, manifest.RequestBody)
	require.NotNil(t, manifest.RequestBody.Value)
	manifestMedia := manifest.RequestBody.Value.Content["application/vnd.loom.manifest+json"]
	require.NotNil(t, manifestMedia)
	require.Equal(t, "#/components/schemas/RawUploadManifest", manifestMedia.Schema.Ref)
}

func TestBuildDocumentPublishesMultipleRawRequestBodyMediaTypes(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		body := dsl.Type("FlexibleBody", func() {
			dsl.Attribute("name", dsl.String)
			dsl.Required("name")
		})
		dsl.Service("flexible", func() {
			dsl.Method("create", func() {
				dsl.HTTP(func() {
					dsl.POST("/flexible")
					dsl.SkipRequestBodyEncodeDecode()
					dsl.OpenAPIRequestBodyTypes(body, []string{
						"application/json",
						"application/x-www-form-urlencoded",
						"multipart/form-data",
					}, true)
				})
			})
		})
	})
	document := BuildDocument(root.API, root.Types, root.ResultTypes)

	body := document.Paths["/flexible"].Operations["POST"].RequestBody.Value
	require.Len(t, body.Content, 3)
	for _, contentType := range []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	} {
		media := body.Content[contentType]
		require.NotNil(t, media)
		require.Equal(t, "#/components/schemas/FlexibleBody", media.Schema.Ref)
	}
}

func TestBuildDocumentOmitsUndocumentedRawRequestBody(t *testing.T) {
	root := codegen.RunDSL(t, testdata.SkipRequestBodyEncodeDecodeDSL)
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	operation := doc.Paths["/"].Operations["POST"]
	require.NotNil(t, operation)
	require.Nil(t, operation.RequestBody)
}

func TestBuildDocumentCarriesErrorRemedyDescriptions(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "error_remedy"
	)

	root := codegen.RunDSL(t, dsls.ErrorRemedyResponseBodyDSL(serviceName, methodName))
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	path := root.API.HTTP.Services[0].HTTPEndpoints[0].Routes[0].FullPaths()[0]
	operation := doc.Paths[path].Operations["POST"]
	require.NotNil(t, operation)
	require.Contains(t, operation.Responses, "400")
	require.NotNil(t, operation.Responses["400"])
	require.NotNil(t, operation.Responses["400"].Value)
	require.Equal(t, "bad: Bad Request response. Remedy code: bad.fix. Safe message: Retry with a valid request. Retry hint: Correct the payload and retry.", operation.Responses["400"].Value.Description)
}

func TestBuildDocumentCanPreserveExactErrorResponseDescription(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("pets", func() {
			dsl.Method("show", func() {
				dsl.Result(dsl.String)
				dsl.Error("not_found", dsl.String)
				dsl.HTTP(func() {
					dsl.GET("/pets")
					dsl.Response(dsl.StatusOK)
					dsl.Response("not_found", dsl.StatusNotFound, func() {
						dsl.Description("Pet was not found.")
						dsl.Meta("openapi:description:errorName", "false")
					})
				})
			})
		})
	})

	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))
	operation := doc.Paths["/pets"].Operations["GET"]
	require.NotNil(t, operation)
	require.Equal(t, "Pet was not found.", operation.Responses["404"].Value.Description)
}

func TestBuildOperationAddsResponseCookieHeader(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "cookie_body"
	)

	root := codegen.RunDSL(t, dsls.MultiCookieResponseBodyDSL(serviceName, methodName))
	bodyTypes := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	var endpoint *expr.HTTPEndpointExpr
	for _, svc := range root.API.HTTP.Services {
		if svc.Name() != serviceName {
			continue
		}
		endpoint = svc.Endpoint("other")
		break
	}
	require.NotNil(t, endpoint)

	operation := buildOperation(transportir.BuildEndpoint(endpoint), bodyTypes.Services[serviceName]["other"], root.API.ExampleGenerator, false)
	require.NotNil(t, operation)
	require.Contains(t, operation.Responses, "200")
	require.NotNil(t, operation.Responses["200"])
	require.NotNil(t, operation.Responses["200"].Value)
	require.Contains(t, operation.Responses["200"].Value.Headers, "Set-Cookie")
	require.NotNil(t, operation.Responses["200"].Value.Headers["Set-Cookie"])
	require.NotNil(t, operation.Responses["200"].Value.Headers["Set-Cookie"].Value)
	require.Equal(t, "string", operation.Responses["200"].Value.Headers["Set-Cookie"].Value.Schema.Type)
}

func TestBuildOperationSuppressesStreamingResponseExamples(t *testing.T) {
	const (
		serviceName = "test service"
		methodName  = "streaming_object"
	)

	root := codegen.RunDSL(t, dsls.ObjectStreamingResponseBodyDSL(serviceName, methodName))
	bodyTypes := BuildBodyTypes(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	var endpoint *expr.HTTPEndpointExpr
	for _, svc := range root.API.HTTP.Services {
		if svc.Name() != serviceName {
			continue
		}
		endpoint = svc.Endpoint(methodName)
		break
	}
	require.NotNil(t, endpoint)

	operation := buildOperation(transportir.BuildEndpoint(endpoint), bodyTypes.Services[serviceName][methodName], root.API.ExampleGenerator, false)
	require.NotNil(t, operation)
	require.Contains(t, operation.Responses, "101")
	response := operation.Responses["101"]
	require.NotNil(t, response)
	require.NotNil(t, response.Value)
	require.Empty(t, response.Value.Content)
}

func TestBuildDocumentComponentizesRepeatedContractNodes(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIReusableComponentsDSL)

	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))
	require.NotNil(t, doc)
	require.NotNil(t, doc.Components)
	require.NotEmpty(t, doc.Components.RequestBodies)
	require.NotEmpty(t, doc.Components.Responses)
	require.NotEmpty(t, doc.Components.Headers)
	require.NotEmpty(t, doc.Components.Examples)

	signin := doc.Paths["/auth/signin"].Operations["POST"]
	refresh := doc.Paths["/auth/refresh"].Operations["POST"]
	require.NotNil(t, signin)
	require.NotNil(t, refresh)
	require.NotNil(t, signin.RequestBody)
	require.NotNil(t, refresh.RequestBody)
	require.NotEmpty(t, signin.RequestBody.Ref)
	require.Equal(t, signin.RequestBody.Ref, refresh.RequestBody.Ref)
}

func TestBuildExampleUsesAuthoredOpenAPISummary(t *testing.T) {
	example := &expr.ExampleExpr{
		Summary: "component-key",
		Value:   "value",
		Meta: expr.MetaExpr{
			"openapi:example:summary": {"Authored summary"},
		},
	}

	built := buildExample(example, example.Value)
	require.Equal(t, "Authored summary", built.Summary)
}

func TestBuildDocumentPublishesResponseLinksAndAsyncContracts(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIProblemLinksAsyncDSL)

	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))
	require.NotNil(t, doc)

	createThread := doc.Paths["/threads"].Operations["POST"]
	require.NotNil(t, createThread)
	require.Contains(t, createThread.Responses, "202")
	accepted := createThread.Responses["202"]
	require.NotNil(t, accepted)
	require.NotEmpty(t, accepted.Ref)
	componentName := strings.TrimPrefix(accepted.Ref, ResponseComponentRefPrefix)
	require.Contains(t, doc.Components.Responses, componentName)
	require.NotNil(t, doc.Components.Responses[componentName])
	require.NotNil(t, doc.Components.Responses[componentName].Value)
	require.Contains(t, doc.Components.Responses[componentName].Value.Links, "thread")
	require.Contains(t, doc.Components.Responses[componentName].Value.Links, "watch")
	require.Equal(t, "thread_ops.get_thread", doc.Components.Responses[componentName].Value.Links["thread"].Value.OperationID)
	require.Equal(t, "$response.body#/thread_id", doc.Components.Responses[componentName].Value.Links["thread"].Value.Parameters["thread_id"])

	watchThread := doc.Paths["/threads/{thread_id}/events"].Operations["GET"]
	require.NotNil(t, watchThread)
	async, ok := watchThread.Extensions[asyncContractExtensionName].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "sse", async["transport"])

	streamCommands := doc.Paths["/ws/ops/{channel}"].Operations["GET"]
	require.NotNil(t, streamCommands)
	async, ok = streamCommands.Extensions[asyncContractExtensionName].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "websocket", async["transport"])
	messages, ok := async["messages"].(map[string]any)
	require.True(t, ok)
	inbound, ok := messages["inbound"].(map[string]any)
	require.True(t, ok)
	schema, ok := inbound["schema"].(*openapi.Schema)
	require.True(t, ok)
	require.Empty(t, schema.Ref)
	require.Equal(t, "object", string(schema.Type))
	require.Contains(t, schema.Properties, "op")
	require.Contains(t, schema.Properties, "target")
}

func TestBuildDocumentPublishesSSEProjectionAlternatives(t *testing.T) {
	root := codegen.RunDSL(t, testdata.SSEVariantProjectionDSL)
	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))

	watch := doc.Paths["/events"].Operations["GET"]
	require.NotNil(t, watch)
	async, ok := watch.Extensions[asyncContractExtensionName].(map[string]any)
	require.True(t, ok)
	messages, ok := async["messages"].(map[string]any)
	require.True(t, ok)
	outbound, ok := messages["outbound"].(map[string]any)
	require.True(t, ok)
	schema, ok := outbound["schema"].(*openapi.Schema)
	require.True(t, ok)
	require.Len(t, schema.OneOf, 2)
	sse, ok := outbound["sse"].(map[string]any)
	require.True(t, ok)
	projections, ok := sse["projections"].([]map[string]string)
	require.True(t, ok)
	require.Equal(t, []map[string]string{
		{"event": "legacy", "view": "legacy"},
		{"event": "updated", "view": "updated"},
	}, projections)

	response := watch.Responses["200"]
	require.NotNil(t, response)
	if response.Ref != "" {
		response = doc.Components.Responses[strings.TrimPrefix(response.Ref, ResponseComponentRefPrefix)]
	}
	require.NotNil(t, response.Value)
	media := response.Value.Content["text/event-stream"]
	require.NotNil(t, media)
	require.Len(t, media.Schema.OneOf, 2)
}

func TestBuildDocumentMixedTransportContracts(t *testing.T) {
	root := codegen.RunDSL(t, mixedTransportDocumentDSL)

	doc := BuildDocument(root.API, root.Types, root.ResultTypes, WithExampleValue(openAPIExampleValueForTest))
	require.NotNil(t, doc)

	create := doc.Paths["/plain/{id}"].Operations["POST"]
	require.NotNil(t, create)
	require.NotNil(t, create.RequestBody)
	require.NotNil(t, create.RequestBody.Value)
	require.True(t, create.RequestBody.Value.Required)

	watch := doc.Paths["/events"].Operations["GET"]
	require.NotNil(t, watch)
	async, ok := watch.Extensions[asyncContractExtensionName].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "sse", async["transport"])

	socket := doc.Paths["/ws"].Operations["GET"]
	require.NotNil(t, socket)
	async, ok = socket.Extensions[asyncContractExtensionName].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "websocket", async["transport"])
}

func openAPIExampleValueForTest(attr *expr.AttributeExpr, raw any) (any, bool) {
	return OpenAPIExampleValue(attr, raw)
}

func mixedTransportDocumentDSL() {
	dsl.Service("MixedDocument", func() {
		dsl.Method("create", func() {
			dsl.Payload(func() {
				dsl.Attribute("id", dsl.String)
				dsl.Attribute("name", dsl.String)
				dsl.Required("id", "name")
			})
			dsl.Result(dsl.String)
			dsl.HTTP(func() {
				dsl.POST("/plain/{id}")
				dsl.Body("name")
			})
		})

		dsl.Method("watch", func() {
			dsl.StreamingResult(func() {
				dsl.Attribute("event", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.ServerSentEvents()
			})
		})

		dsl.Method("socket", func() {
			dsl.StreamingPayload(func() {
				dsl.Attribute("message", dsl.String)
			})
			dsl.StreamingResult(func() {
				dsl.Attribute("event", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/ws")
			})
		})
	})
}
