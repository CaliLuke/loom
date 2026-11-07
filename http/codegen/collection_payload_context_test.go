package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestCollectionPayloadConstructorUsesBoundElement(t *testing.T) {
	for _, test := range []struct {
		name    string
		service string
		design  func()
	}{
		{"headers and queries", "namedarrays", testdata.NamedStringArraysDSL},
		{"paths", "namedarrays", testdata.NamedStringArrayPathsDSL},
		{"map values", "ServiceMapQueryPrimitivePrimitive", testdata.PayloadMapQueryPrimitivePrimitiveDSL},
		{"map array values", "ServiceMapQueryPrimitiveArray", testdata.PayloadMapQueryPrimitiveArrayDSL},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := CreateHTTPServices(RunHTTPDSL(t, test.design)).Get(test.service)
			checked := 0
			for _, endpoint := range data.Endpoints {
				request := endpoint.Payload.Request
				init := request.PayloadInit
				if init != nil && !init.ReturnIsStruct {
					checked++
					// These maintained fixtures declare native string/uint
					// collections in parameter storage, not JSON Optional values.
					require.NotContains(t, init.ServerCode, ".Value()", endpoint.Method.Name)
					require.NotContains(t, init.ClientCode, ".Value()", endpoint.Method.Name)
				}
				for _, param := range request.QueryParams {
					if param.MapQueryParams == nil {
						continue
					}
					checked++
					require.NotNil(t, param.attribute)
					require.NotNil(t, param.context)
					require.True(t, expr.IsMap(param.attribute.Type))
					require.False(t, param.context.CollectionElementPresence)
				}
			}
			require.Positive(t, checked)
		})
	}
}

func TestNamedMapParamsConstructorUsesBoundElement(t *testing.T) {
	root := RunHTTPDSL(t, parameterRootCollectionsDSL)
	logicalType := root.UserType("RootMap")
	originalShape := logicalType.Attribute().Type
	data := CreateHTTPServices(root).Get("collections")
	request := data.Endpoint("mapParams").Payload.Request
	require.NotNil(t, request.PayloadInit, "named MapParams must exercise whole-payload construction")
	require.NotContains(t, request.PayloadInit.ServerCode, ".Value()")
	require.NotContains(t, request.PayloadInit.ClientCode, ".Value()")
	require.Len(t, request.QueryParams, 1)
	param := request.QueryParams[0]
	require.NotNil(t, param.MapQueryParams)
	require.NotNil(t, param.attribute)
	require.NotNil(t, param.context)
	require.True(t, expr.IsMap(param.attribute.Type))
	require.False(t, param.context.CollectionElementPresence)
	require.IsType(t, &expr.Map{}, param.Type, "query codecs consume a concrete map carrier")
	require.Same(t, param.Type, param.attribute.Type)
	require.Implements(t, (*expr.UserType)(nil), param.FieldType, "service type identity remains named")
	require.Equal(t, "RootMap", param.FieldType.Name())
	require.Same(t, originalShape, logicalType.Attribute().Type, "transport normalization must not mutate the source alias")
}

func TestNamedMapParamsFieldUsesBoundElement(t *testing.T) {
	services := CreateHTTPServices(RunHTTPDSL(t, parameterRootCollectionsDSL))
	request := services.Get("collections").Endpoint("mapField").Payload.Request
	require.NotNil(t, request.PayloadInit)
	require.True(t, request.PayloadInit.ReturnIsStruct, "field mapping exercises struct initialization")
	require.Len(t, request.QueryParams, 1)
	param := request.QueryParams[0]
	require.NotNil(t, param.MapQueryParams)
	require.Equal(t, "filters", *param.MapQueryParams)
	require.Equal(t, "Filters", param.FieldName)
	require.IsType(t, &expr.Map{}, param.Type)
	require.Implements(t, (*expr.UserType)(nil), param.FieldType)
	require.Equal(t, "RootMap", param.FieldType.Name())
	rendered := filesCode(t, ServerTypeFiles("gen", services))
	require.Contains(t, rendered, "func "+request.PayloadInit.Name+"(")
	require.Contains(t, rendered, "v.Filters = make(map[string]string, len(filters))")
}
