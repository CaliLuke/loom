package codegen

import (
	"fmt"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// TestServicePackageAliasRenderedVerbatim checks that the JSON-RPC sections
// referencing the service package qualify identifiers with the service
// package import alias exactly as computed. The alias "my_svc" contains an
// underscore, which jen.Qual would strip when deriving a qualifier from an
// import path, rendering "mysvc".
func TestServicePackageAliasRenderedVerbatim(t *testing.T) {
	const alias = "my_svc"
	cases := []struct {
		Name   string
		Design func()
		Render func(*httpcodegen.ServiceData) jen.Code
		Want   []string
	}{
		{
			Name:   "server-init-params",
			Design: serviceAliasSSEDSL,
			Render: func(data *httpcodegen.ServiceData) jen.Code {
				return jen.Func().Id("New").Params(jsonrpcServerInitParams(data)...).Block()
			},
			Want: []string{"endpoints *my_svc.Endpoints"},
		},
		{
			Name:   "sse-handler-init",
			Design: serviceAliasSSEDSL,
			Render: func(data *httpcodegen.ServiceData) jen.Code {
				e := serviceAliasEndpoint(t, data, "watch")
				return jen.Func().Id("f").Params().BlockFunc(func(g *jen.Group) {
					writeSSEHandlerInitBody(g, e)
				})
			},
			Want: []string{"v := &my_svc.WatchEndpointInput{"},
		},
		{
			Name:   "websocket-streaming-request",
			Design: serviceAliasWebSocketDSL,
			Render: func(data *httpcodegen.ServiceData) jen.Code {
				e := serviceAliasEndpoint(t, data, "exchange")
				return jen.Switch().BlockFunc(func(g *jen.Group) {
					writeWebSocketRequestCase(g, e)
				})
			},
			Want: []string{"endpointInput := &my_svc.ExchangeEndpointInput{"},
		},
		{
			Name:   "websocket-client-streaming-request",
			Design: serviceAliasWebSocketDSL,
			Render: func(data *httpcodegen.ServiceData) jen.Code {
				e := serviceAliasEndpoint(t, data, "upload")
				return jen.Switch().BlockFunc(func(g *jen.Group) {
					writeWebSocketRequestCase(g, e)
				})
			},
			Want: []string{"res.(*my_svc.UploadResult)"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, c.Design)
			data := CreateJSONRPCServices(root).Get("aliased")
			require.NotNil(t, data)
			svc := *data.Service
			svc.PkgName = alias
			data.Service = &svc
			for _, e := range data.Endpoints {
				e.ServicePkgName = alias
			}
			code := fmt.Sprintf("%#v", c.Render(data))
			for _, want := range c.Want {
				require.Contains(t, code, want)
			}
			require.NotContains(t, code, "mysvc.")
		})
	}
}

// serviceAliasEndpoint returns the endpoint of data for the named method.
func serviceAliasEndpoint(t *testing.T, data *httpcodegen.ServiceData, method string) *httpcodegen.EndpointData {
	t.Helper()
	for _, e := range data.Endpoints {
		if e.Method.Name == method {
			return e
		}
	}
	require.Failf(t, "missing endpoint", "method %q", method)
	return nil
}

// serviceAliasSSEDSL declares a JSON-RPC service with a server-sent events
// method.
func serviceAliasSSEDSL() {
	dsl.API("alias", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("aliased", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("watch", func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
			})
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}

// serviceAliasWebSocketDSL declares a JSON-RPC service served over a
// WebSocket with a bidirectional and a client streaming method.
func serviceAliasWebSocketDSL() {
	var Frame = dsl.Type("Frame", func() {
		dsl.Attribute("value", dsl.String)
	})
	dsl.API("alias", func() {
		dsl.JSONRPC(func() {})
	})
	dsl.Service("aliased", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/rpc")
		})
		dsl.Method("exchange", func() {
			dsl.StreamingPayload(Frame)
			dsl.StreamingResult(Frame)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("upload", func() {
			dsl.StreamingPayload(Frame)
			dsl.Result(Frame)
			dsl.JSONRPC(func() {})
		})
	})
}
