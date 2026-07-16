package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	stest "github.com/CaliLuke/loom/codegen/service/testdata"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAnalyzeServiceErrorsCarryRemedyMetadataToErrorTypes(t *testing.T) {
	root := codegen.RunDSL(t, stest.ErrorRemedyMethodDSL)
	services := NewServicesData(root)
	svc := services.Get("ErrorRemedyMethod")
	require.NotNil(t, svc)
	require.NotEmpty(t, svc.errorTypes)

	var found bool
	for _, ut := range svc.errorTypes {
		if ut.RemedyCode == "bad_request.fix" {
			require.Equal(t, "The request is invalid.", ut.SafeMessage)
			require.Equal(t, "Correct the payload and retry.", ut.RetryHint)
			found = true
			break
		}
	}

	require.True(t, found)
}

func TestAnalyzeWrapsRawObjectPayloadsWithUniqueSyntheticTypeNames(t *testing.T) {
	root := codegen.RunDSL(t, stest.RawObjectPayloadTypeNameCollisionDSL)
	services := NewServicesData(root)
	svc := services.Get("RawObjectPayloadTypeNameCollision")
	require.NotNil(t, svc)

	method := svc.Method("Foo")
	require.NotNil(t, method)
	require.Equal(t, "FooPayload3", method.Payload)
}

func TestAnalyzeWrapsRawObjectResults(t *testing.T) {
	root := codegen.RunDSL(t, stest.WithDefaultDSL)
	services := NewServicesData(root)
	svc := services.Get("WithDefault")
	require.NotNil(t, svc)

	method := svc.Method("A")
	require.NotNil(t, method)
	require.Equal(t, "AResult", method.Result)
	require.Equal(t, "*AResult", method.ResultRef)
	_, ok := root.Services[0].Methods[0].Result.Type.(expr.UserType)
	require.True(t, ok)
}

func TestAnalyzeViewedResultsDeduplicateCanonicalTypeButPreserveMethodViews(t *testing.T) {
	root := codegen.RunDSL(t, stest.WithExplicitAndDefaultViewsDSL)
	services := NewServicesData(root)
	svc := services.Get("WithExplicitAndDefaultViews")
	require.NotNil(t, svc)
	require.Len(t, svc.viewedResultTypes, 1)
	require.Len(t, svc.Methods, 2)
	require.NotSame(t, svc.Methods[0].ViewedResult, svc.Methods[1].ViewedResult)
	require.Equal(t, "", svc.Methods[0].ViewedResult.ViewName)
	require.Equal(t, "tiny", svc.Methods[1].ViewedResult.ViewName)
	require.Equal(t, svc.Methods[0].ViewedResult.FullName, svc.Methods[1].ViewedResult.FullName)
}

func TestAnalyzeForceGeneratedTypesRespectServiceFilters(t *testing.T) {
	cases := []struct {
		name     string
		dsl      func()
		service  string
		expected bool
	}{
		{
			name:     "unfiltered force generate",
			dsl:      stest.ForceGenerateTypeDSL,
			service:  "ForceGenerateType",
			expected: true,
		},
		{
			name:     "matching explicit service filter",
			dsl:      stest.ForceGenerateTypeExplicitDSL,
			service:  "ForceGenerateTypeExplicit",
			expected: true,
		},
		{
			name:     "non matching explicit service filter",
			dsl:      forceGenerateTypeMismatchDSL,
			service:  "ForceGenerateTypeMismatch",
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.dsl)
			services := NewServicesData(root)
			svc := services.Get(c.service)
			require.NotNil(t, svc)
			require.Equal(t, c.expected, hasServiceUserType(svc.userTypes, "ForcedType"))
		})
	}
}

func TestBuildMethodDataClassifiesJSONRPCStreamingTransports(t *testing.T) {
	t.Run("jsonrpc sse classification", func(t *testing.T) {
		root := codegen.RunDSL(t, jsonrpcSSEClassificationDSL)
		services := NewServicesData(root)
		svc := services.Get("JSONRPCSSEClassification")
		require.NotNil(t, svc)

		method := svc.Method("Watch")
		require.NotNil(t, method)
		require.True(t, method.IsJSONRPC)
		require.True(t, method.IsJSONRPCSSE)
		require.False(t, method.IsJSONRPCWebSocket)
	})

	t.Run("jsonrpc websocket classification", func(t *testing.T) {
		root := codegen.RunDSL(t, jsonrpcWebSocketClassificationDSL)
		services := NewServicesData(root)
		svc := services.Get("JSONRPCWebSocketClassification")
		require.NotNil(t, svc)

		method := svc.Method("Watch")
		require.NotNil(t, method)
		require.True(t, method.IsJSONRPC)
		require.False(t, method.IsJSONRPCSSE)
		require.True(t, method.IsJSONRPCWebSocket)
	})
}

func TestBuildMethodDataPropagatesHTTPSkipBodyFlags(t *testing.T) {
	t.Run("skip request body", func(t *testing.T) {
		root := codegen.RunDSL(t, httpSkipRequestBodyDSL)
		services := NewServicesData(root)
		svc := services.Get("HTTPSkipRequestBody")
		require.NotNil(t, svc)

		method := svc.Method("Upload")
		require.NotNil(t, method)
		require.True(t, method.SkipRequestBodyEncodeDecode)
		require.False(t, method.SkipResponseBodyEncodeDecode)
	})

	t.Run("skip response body", func(t *testing.T) {
		root := codegen.RunDSL(t, httpSkipResponseBodyDSL)
		services := NewServicesData(root)
		svc := services.Get("HTTPSkipResponseBody")
		require.NotNil(t, svc)

		method := svc.Method("Download")
		require.NotNil(t, method)
		require.False(t, method.SkipRequestBodyEncodeDecode)
		require.True(t, method.SkipResponseBodyEncodeDecode)
	})
}

func TestBuildMethodDataMixedResultsStreamMetadata(t *testing.T) {
	root := codegen.RunDSL(t, mixedResultsStreamMetadataDSL)
	services := NewServicesData(root)
	svc := services.Get("MixedResultsStreamMetadata")
	require.NotNil(t, svc)

	method := svc.Method("Watch")
	require.NotNil(t, method)
	require.True(t, method.HasMixedResults)
	require.Equal(t, expr.ServerStreamKind, method.ServerStream.Kind)
	require.Equal(t, "WatchEndpointInput", method.ServerStream.EndpointStruct)
	require.Equal(t, "WatchServerStream", method.ServerStream.Interface)
	require.Equal(t, "WatchClientStream", method.ClientStream.Interface)
	require.Equal(t, "WatchEvent", method.StreamingResult)
	require.Equal(t, "WatchEvent", method.ServerStream.SendTypeName)
	require.Equal(t, "WatchEvent", method.ClientStream.RecvTypeName)
}

func TestBuildMethodDataPartitionsConcernFields(t *testing.T) {
	root := codegen.RunDSL(t, serviceDataRefactorRegressionDSL)
	services := NewServicesData(root)
	svc := services.Get("ServiceDataRefactor")
	require.NotNil(t, svc)

	method := svc.Method("Watch")
	require.NotNil(t, method)

	require.Equal(t, "UnionPayload", method.MethodPayloadData.Payload)
	require.Equal(t, "*UnionPayload", method.MethodPayloadData.PayloadRef)
	require.Equal(t, "WatchResult", method.MethodResultData.Result)
	require.Equal(t, "*WatchResult", method.MethodResultData.ResultRef)
	require.False(t, method.MethodTransportData.IsJSONRPC)
	require.False(t, method.MethodTransportData.IsJSONRPCSSE)
	require.Equal(t, "WatchRequestData", method.MethodTransportData.RequestStruct)
	require.Equal(t, "WatchResponseData", method.MethodTransportData.ResponseStruct)
	require.True(t, method.MethodStreamingData.HasMixedResults)
	require.Equal(t, expr.ServerStreamKind, method.MethodStreamingData.StreamKind)
	require.NotNil(t, method.MethodStreamingData.ServerStream)
	require.NotNil(t, method.MethodStreamingData.ClientStream)
	require.Len(t, method.MethodSecurityData.Errors, 1)
	require.Empty(t, method.MethodSecurityData.Schemes)
	require.NotEmpty(t, method.MethodTransportData.EndpointField)
	require.NotEmpty(t, method.MethodTransportData.StreamEndpointField)
	require.NotNil(t, method.MethodResultData.ViewedResult)
}

func TestAnalyzeServiceDataRefactorRegression(t *testing.T) {
	root := codegen.RunDSL(t, serviceDataRefactorRegressionDSL)
	services := NewServicesData(root)
	svc := services.Get("ServiceDataRefactor")
	require.NotNil(t, svc)

	require.ElementsMatch(t, []string{
		"ErrorChoice",
		"PayloadChoice",
		"ResultChoice",
		"StreamChoice",
		"Value",
	}, serviceUnionTypeNames(svc.unions))
	require.Len(t, svc.ServerInterceptors, 1)
	require.Len(t, svc.ClientInterceptors, 1)
	require.Len(t, svc.viewedResultTypes, 1)

	var remedyType *UserTypeData
	for _, ut := range svc.errorTypes {
		if ut.RemedyCode == "service_bad.fix" {
			remedyType = ut
			break
		}
	}
	require.NotNil(t, remedyType)
	require.Equal(t, "The service request is invalid.", remedyType.SafeMessage)
	require.Equal(t, "Correct the payload and retry.", remedyType.RetryHint)

	method := svc.Method("Watch")
	require.NotNil(t, method)
	require.True(t, method.HasMixedResults)
	require.Len(t, method.ServerInterceptors, 1)
	require.Len(t, method.ClientInterceptors, 1)
	require.NotEmpty(t, method.EndpointField)
	require.NotEmpty(t, method.StreamEndpointField)
	require.NotNil(t, method.ViewedResult)

	viewed := svc.viewedResultTypes[0]
	require.Len(t, viewed.Views, 2)
	require.Equal(t, "default", viewed.Views[0].Name)
	require.Equal(t, "tiny", viewed.Views[1].Name)
}

func serviceUnionTypeNames(types []*UnionTypeData) []string {
	names := make([]string, len(types))
	for i, union := range types {
		names[i] = union.Name
	}
	return names
}

func hasServiceUserType(types []*UserTypeData, name string) bool {
	for _, ut := range types {
		if ut.Name == name {
			return true
		}
	}
	return false
}

func forceGenerateTypeMismatchDSL() {
	var _ = dsl.Type("ForcedType", func() {
		dsl.Attribute("a", dsl.String)
		dsl.Meta("type:generate:force", "OtherService")
	})

	dsl.Service("ForceGenerateTypeMismatch", func() {
		dsl.Method("A", func() {})
	})
}

func jsonrpcSSEClassificationDSL() {
	dsl.API("jsonrpc-sse-classification-api", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("JSONRPCSSEClassification", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("Watch", func() {
			dsl.StreamingResult(func() {
				dsl.Attribute("event", dsl.String)
			})
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}

func jsonrpcWebSocketClassificationDSL() {
	dsl.API("jsonrpc-websocket-classification-api", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("JSONRPCWebSocketClassification", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/rpc")
		})
		dsl.Method("Watch", func() {
			dsl.StreamingResult(func() {
				dsl.Attribute("event", dsl.String)
			})
			dsl.JSONRPC(func() {})
		})
	})
}

func httpSkipRequestBodyDSL() {
	dsl.Service("HTTPSkipRequestBody", func() {
		dsl.Method("Upload", func() {
			dsl.Payload(func() {
				dsl.Attribute("name", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.POST("/")
				dsl.Header("name")
				dsl.SkipRequestBodyEncodeDecode()
			})
		})
	})
}

func httpSkipResponseBodyDSL() {
	dsl.Service("HTTPSkipResponseBody", func() {
		dsl.Method("Download", func() {
			dsl.Result(func() {
				dsl.Attribute("name", dsl.String)
			})
			dsl.HTTP(func() {
				dsl.GET("/")
				dsl.SkipResponseBodyEncodeDecode()
				dsl.Response(func() {
					dsl.Header("name")
				})
			})
		})
	})
}

func mixedResultsStreamMetadataDSL() {
	var watchEvent = dsl.Type("WatchEvent", func() {
		dsl.Attribute("event", dsl.String)
	})

	dsl.Service("MixedResultsStreamMetadata", func() {
		dsl.Method("Watch", func() {
			dsl.Result(func() {
				dsl.Attribute("done", dsl.Boolean)
			})
			dsl.StreamingResult(watchEvent)
			dsl.HTTP(func() {
				dsl.GET("/")
				dsl.ServerSentEvents()
			})
		})
	})
}

func serviceDataRefactorRegressionDSL() {
	var unionPayload = dsl.Type("UnionPayload", func() {
		dsl.OneOf("value", func() {
			dsl.Attribute("id", dsl.Int)
			dsl.Attribute("name", dsl.String)
		})
	})
	var watchResult = dsl.ResultType("application/vnd.service-data-refactor.watch", func() {
		dsl.TypeName("WatchResult")
		dsl.Attributes(func() {
			dsl.Attribute("name", dsl.String)
			dsl.Attribute("count", dsl.Int)
			dsl.Required("name", "count")
		})
		dsl.View("default", func() {
			dsl.Attribute("name")
			dsl.Attribute("count")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("name")
		})
	})
	var watchEvent = dsl.Type("WatchEvent", func() {
		dsl.Attribute("event", dsl.String)
	})
	var payloadEnvelope = dsl.Type("ServiceDataPayloadEnvelope", func() {
		dsl.OneOf("payload_choice", func() {
			dsl.Attribute("name", dsl.String)
			dsl.Attribute("count", dsl.Int)
		})
	})
	var streamEnvelope = dsl.Type("ServiceDataStreamEnvelope", func() {
		dsl.OneOf("stream_choice", func() {
			dsl.Attribute("message", dsl.String)
			dsl.Attribute("sequence", dsl.Int)
		})
	})
	var resultEnvelope = dsl.Type("ServiceDataResultEnvelope", func() {
		dsl.OneOf("result_choice", func() {
			dsl.Attribute("accepted", dsl.Boolean)
			dsl.Attribute("location", dsl.String)
		})
	})
	var errorEnvelope = dsl.Type("ServiceDataErrorEnvelope", func() {
		dsl.OneOf("error_choice", func() {
			dsl.Attribute("field", dsl.String)
			dsl.Attribute("retry_after", dsl.Int)
		})
	})

	dsl.Interceptor("logging")
	dsl.Interceptor("tracing")
	dsl.Service("ServiceDataRefactor", func() {
		dsl.Error("service_bad", func() {
			dsl.Remedy(func() {
				dsl.RemedyCode("service_bad.fix")
				dsl.SafeMessage("The service request is invalid.")
				dsl.RetryHint("Correct the payload and retry.")
			})
		})
		dsl.ServerInterceptor("logging")
		dsl.ClientInterceptor("tracing")

		dsl.Method("Watch", func() {
			dsl.Payload(unionPayload)
			dsl.Result(watchResult)
			dsl.StreamingResult(watchEvent)
			dsl.HTTP(func() {
				dsl.GET("/")
				dsl.ServerSentEvents()
			})
		})
		dsl.Method("Submit", func() {
			dsl.Payload(dsl.MapOf(dsl.String, payloadEnvelope))
		})
		dsl.Method("Upload", func() {
			dsl.StreamingPayload(dsl.ArrayOf(streamEnvelope))
			dsl.Result(dsl.String)
		})
		dsl.Method("List", func() {
			dsl.Result(dsl.MapOf(dsl.String, resultEnvelope))
		})
		dsl.Method("Fail", func() {
			dsl.Error("invalid", dsl.ArrayOf(errorEnvelope))
		})
	})
}
