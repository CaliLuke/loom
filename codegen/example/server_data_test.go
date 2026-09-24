package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestServerDataHostedTransports checks the per-server transport presence
// that the example generators share: a server hosts a transport only when one
// of its own services uses it, whatever the other servers host.
func TestServerDataHostedTransports(t *testing.T) {
	web := &expr.ServiceExpr{Name: "web"}
	store := &expr.ServiceExpr{Name: "store"}
	rpc := &expr.ServiceExpr{Name: "rpc"}
	dual := &expr.ServiceExpr{Name: "dual"}
	both := &expr.ServiceExpr{Name: "both"}
	root := &expr.RootExpr{
		API: &expr.APIExpr{
			HTTP: &expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{
				{ServiceExpr: web}, {ServiceExpr: dual}, {ServiceExpr: both},
			}},
			JSONRPC: &expr.JSONRPCExpr{HTTPExpr: expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{
				{ServiceExpr: rpc}, {ServiceExpr: both},
			}}},
			GRPC: &expr.GRPCExpr{Services: []*expr.GRPCServiceExpr{
				{ServiceExpr: store}, {ServiceExpr: dual},
			}},
		},
		Services: []*expr.ServiceExpr{web, store, rpc, dual, both},
	}
	cases := []struct {
		Name        string
		Services    []string
		WantHTTP    bool
		WantJSONRPC bool
		WantGRPC    bool
	}{
		{Name: "http-only", Services: []string{"web"}, WantHTTP: true},
		{Name: "grpc-only", Services: []string{"store"}, WantGRPC: true},
		{Name: "jsonrpc-only", Services: []string{"rpc"}, WantJSONRPC: true},
		{Name: "http-and-grpc-services", Services: []string{"web", "store"}, WantHTTP: true, WantGRPC: true},
		{Name: "http-and-grpc-service", Services: []string{"dual"}, WantHTTP: true, WantGRPC: true},
		{Name: "http-and-jsonrpc-service", Services: []string{"both"}, WantHTTP: true, WantJSONRPC: true},
		{Name: "jsonrpc-and-grpc-services", Services: []string{"rpc", "store"}, WantJSONRPC: true, WantGRPC: true},
		{Name: "all", Services: []string{"web", "rpc", "store"}, WantHTTP: true, WantJSONRPC: true, WantGRPC: true},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			svr := &expr.ServerExpr{
				Name:     c.Name,
				Services: c.Services,
				Hosts: []*expr.HostExpr{{
					Name:      "local",
					URIs:      []expr.URIExpr{"http://localhost:80", "grpc://localhost:8080"},
					Variables: &expr.AttributeExpr{Type: &expr.Object{}},
				}},
			}
			data := buildServerData(svr, root)
			assert.Equal(t, c.WantHTTP, data.HostsHTTP(), "HostsHTTP")
			assert.Equal(t, c.WantJSONRPC, data.HostsJSONRPC(), "HostsJSONRPC")
			assert.Equal(t, c.WantGRPC, data.HostsGRPC(), "HostsGRPC")
		})
	}
}
