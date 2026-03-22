package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/v3/codegen"
	stest "github.com/CaliLuke/loom/v3/codegen/service/testdata"
	dsl "github.com/CaliLuke/loom/v3/dsl"
	"github.com/CaliLuke/loom/v3/expr"
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
			dsl.POST("/rpc")
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
