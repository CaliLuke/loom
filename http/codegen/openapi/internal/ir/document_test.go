package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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
	return openAPIExampleValue(attr, raw)
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
