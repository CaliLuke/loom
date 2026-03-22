package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func TestJSONRPCCodegenWithSynthesizedService(t *testing.T) {
	serviceExpr := &expr.ServiceExpr{
		Name: "calc",
		Methods: []*expr.MethodExpr{{
			Name:    "add",
			Payload: &expr.AttributeExpr{Type: expr.String},
			Result:  &expr.AttributeExpr{Type: expr.String},
		}},
	}
	httpService := expr.NewJSONRPCHTTPService(serviceExpr, "/rpc")
	root := &expr.RootExpr{
		Services: []*expr.ServiceExpr{serviceExpr},
		API: &expr.APIExpr{
			Name: "rpc",
			ExampleGenerator: &expr.ExampleGenerator{
				Randomizer: expr.NewFakerRandomizer("rpc"),
			},
			HTTP: &expr.HTTPExpr{},
			JSONRPC: &expr.JSONRPCExpr{
				HTTPExpr: expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{httpService}},
			},
			GRPC: &expr.GRPCExpr{},
			Servers: []*expr.ServerExpr{{
				Name:     "calc",
				Services: []string{"calc"},
			}},
		},
	}
	httpService.Root = &root.API.JSONRPC.HTTPExpr

	err := expr.PrepareValidateFinalize(root)
	require.NoError(t, err)

	services := service.NewServicesData(root)
	httpServices := httpcodegen.NewServicesData(services, &root.API.JSONRPC.HTTPExpr)
	files := ServerFiles("goa.design/example", httpServices)

	require.NotEmpty(t, files)
	require.NotNil(t, httpServices.Get("calc"))
}
