package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONRPCHTTPService(t *testing.T) {
	t.Run("builds JSON-RPC service endpoints for all methods", func(t *testing.T) {
		service := &ServiceExpr{
			Name: "calc",
			Meta: MetaExpr{"existing": []string{"value"}},
			Methods: []*MethodExpr{
				{
					Name:    "add",
					Service: nil,
					Payload: &AttributeExpr{Type: String},
					Result:  &AttributeExpr{Type: String},
				},
				{
					Name:    "watch",
					Payload: &AttributeExpr{Type: String},
					Result:  &AttributeExpr{Type: String},
					Stream:  ServerStreamKind,
				},
			},
		}

		httpService := NewJSONRPCHTTPService(service, "/rpc")

		require.NotNil(t, httpService)
		require.Equal(t, service, httpService.ServiceExpr)
		require.NotNil(t, service.Meta["jsonrpc:service"])
		require.Equal(t, []string{"value"}, service.Meta["existing"])
		require.NotNil(t, httpService.JSONRPCRoute)
		require.Equal(t, "POST", httpService.JSONRPCRoute.Method)
		require.Equal(t, "/rpc", httpService.JSONRPCRoute.Path)
		require.NotNil(t, httpService.JSONRPCRoute.Endpoint)
		require.Len(t, httpService.HTTPEndpoints, 2)
		require.Same(t, service, service.Methods[0].Service)
		require.Same(t, service, service.Methods[1].Service)

		add := httpService.HTTPEndpoints[0]
		require.Equal(t, service.Methods[0], add.MethodExpr)
		require.NotNil(t, add.Meta["jsonrpc"])
		require.Same(t, service.Methods[0].Payload, add.Body)
		require.NotNil(t, add.Params)
		require.NotNil(t, add.Headers)
		require.NotNil(t, add.Cookies)
		require.Len(t, add.Routes, 1)
		require.Equal(t, "/rpc", add.Routes[0].Path)
		require.Equal(t, "POST", add.Routes[0].Method)
		require.Same(t, add, add.Routes[0].Endpoint)
		assert.Nil(t, add.SSE)

		stream := httpService.HTTPEndpoints[1]
		require.NotNil(t, stream.SSE)
		require.Same(t, service.Methods[1].Payload, stream.Body)
	})

	t.Run("normalizes empty path to root", func(t *testing.T) {
		service := &ServiceExpr{
			Name: "calc",
			Methods: []*MethodExpr{{
				Name:    "add",
				Payload: &AttributeExpr{Type: String},
				Result:  &AttributeExpr{Type: String},
			}},
		}

		httpService := NewJSONRPCHTTPService(service, "")

		require.Equal(t, "/", httpService.JSONRPCRoute.Path)
		require.Equal(t, "/", httpService.HTTPEndpoints[0].Routes[0].Path)
	})

	t.Run("panics on nil service", func(t *testing.T) {
		require.PanicsWithValue(t, "nil service", func() {
			NewJSONRPCHTTPService(nil, "/rpc")
		})
	})

	t.Run("works with temporary root finalization", func(t *testing.T) {
		service := &ServiceExpr{
			Name: "calc",
			Methods: []*MethodExpr{{
				Name:    "add",
				Payload: &AttributeExpr{Type: String},
				Result:  &AttributeExpr{Type: String},
			}},
		}
		httpService := NewJSONRPCHTTPService(service, "/rpc")
		root := &RootExpr{
			Services: []*ServiceExpr{service},
			API: &APIExpr{
				Name: "rpc",
				HTTP: &HTTPExpr{},
				JSONRPC: &JSONRPCExpr{
					HTTPExpr: HTTPExpr{Services: []*HTTPServiceExpr{httpService}},
				},
				GRPC: &GRPCExpr{},
			},
		}
		httpService.Root = &root.API.JSONRPC.HTTPExpr

		err := PrepareValidateFinalize(root)

		require.NoError(t, err)
		require.Len(t, httpService.HTTPEndpoints[0].Routes, 1)
		require.Equal(t, "/rpc", httpService.HTTPEndpoints[0].Routes[0].Path)
	})
}
