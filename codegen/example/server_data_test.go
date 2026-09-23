package example

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestComputeHandlerArgsJSONRPCOrdering(t *testing.T) {
	method := &expr.MethodExpr{Name: "Run"}
	httpSvc := &expr.HTTPServiceExpr{
		ServiceExpr: &expr.ServiceExpr{
			Name:    "orchestrator",
			Methods: []*expr.MethodExpr{method},
		},
		HTTPEndpoints: []*expr.HTTPEndpointExpr{{MethodExpr: method}},
	}
	mcpMethod := &expr.MethodExpr{Name: "ListTools"}
	jsonrpcOrchestrator := &expr.HTTPServiceExpr{ServiceExpr: &expr.ServiceExpr{Name: "orchestrator"}}
	jsonrpcMCPAssistant := &expr.HTTPServiceExpr{
		ServiceExpr: &expr.ServiceExpr{
			Name:    "mcp_assistant",
			Methods: []*expr.MethodExpr{mcpMethod},
		},
		HTTPEndpoints: []*expr.HTTPEndpointExpr{{MethodExpr: mcpMethod}},
	}
	root := &expr.RootExpr{
		API: &expr.APIExpr{
			HTTP: &expr.HTTPExpr{
				Services: []*expr.HTTPServiceExpr{httpSvc},
			},
			JSONRPC: &expr.JSONRPCExpr{
				HTTPExpr: expr.HTTPExpr{
					Services: []*expr.HTTPServiceExpr{jsonrpcOrchestrator, jsonrpcMCPAssistant},
				},
			},
		},
		Services: []*expr.ServiceExpr{
			{Name: "orchestrator", Methods: []*expr.MethodExpr{method}},
			{Name: "mcp_assistant", Methods: []*expr.MethodExpr{mcpMethod}},
		},
	}
	server := &Data{
		Services: []string{"orchestrator", "mcp_assistant"},
		Transports: []*TransportData{
			{Type: TransportHTTP, Services: []string{"orchestrator", "mcp_assistant"}},
		},
	}
	args := computeHandlerArgs(TransportHTTP, server, root)

	want := []HandlerArg{
		{ServiceName: "orchestrator", Endpoint: "orchestratorEndpoints"},
		{ServiceName: "orchestrator", Service: "orchestratorSvc"},
		{ServiceName: "mcp_assistant", Service: "mcpAssistantSvc"},
		{ServiceName: "mcp_assistant", Endpoint: "mcpAssistantEndpoints"},
	}
	if len(args) != len(want) {
		t.Fatalf("expected %d handler args, got %d (%v)", len(want), len(args), args)
	}
	for i, arg := range want {
		if args[i] != arg {
			t.Fatalf("handler arg %d: expected %+v, got %+v", i, arg, args[i])
		}
	}
}

func TestComputeHandlerArgsSkipsJSONRPCServicesOutsideServer(t *testing.T) {
	method := &expr.MethodExpr{Name: "Run"}
	httpSvc := &expr.HTTPServiceExpr{
		ServiceExpr:   &expr.ServiceExpr{Name: "web", Methods: []*expr.MethodExpr{method}},
		HTTPEndpoints: []*expr.HTTPEndpointExpr{{MethodExpr: method}},
	}
	rpcMethod := &expr.MethodExpr{Name: "Echo"}
	rpcSvc := &expr.HTTPServiceExpr{
		ServiceExpr:   &expr.ServiceExpr{Name: "rpc", Methods: []*expr.MethodExpr{rpcMethod}},
		HTTPEndpoints: []*expr.HTTPEndpointExpr{{MethodExpr: rpcMethod}},
	}
	root := &expr.RootExpr{
		API: &expr.APIExpr{
			HTTP:    &expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{httpSvc}},
			JSONRPC: &expr.JSONRPCExpr{HTTPExpr: expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{rpcSvc}}},
			GRPC:    &expr.GRPCExpr{},
		},
		Services: []*expr.ServiceExpr{httpSvc.ServiceExpr, rpcSvc.ServiceExpr},
	}
	cases := []struct {
		Name     string
		Services []string
		Want     []HandlerArg
	}{
		{
			Name:     "hosts-both",
			Services: []string{"web", "rpc"},
			Want: []HandlerArg{
				{ServiceName: "web", Endpoint: "webEndpoints"},
				{ServiceName: "rpc", Service: "rpcSvc"},
				{ServiceName: "rpc", Endpoint: "rpcEndpoints"},
			},
		},
		{
			Name:     "hosts-http-only",
			Services: []string{"web"},
			Want:     []HandlerArg{{ServiceName: "web", Endpoint: "webEndpoints"}},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			svr := &expr.ServerExpr{
				Name:     c.Name,
				Services: c.Services,
				Hosts: []*expr.HostExpr{{
					Name:      "local",
					URIs:      []expr.URIExpr{"http://localhost:80", "https://localhost:443"},
					Variables: &expr.AttributeExpr{Type: &expr.Object{}},
				}},
			}
			data := buildServerData(svr, root)
			require.Equal(t, c.Want, data.HTTPHandlerArgs)
			for _, uri := range data.Hosts[0].URIs {
				require.Equal(t, c.Want, uri.HandlerArgs, uri.URL)
			}
		})
	}
}
