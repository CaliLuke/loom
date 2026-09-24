package codegen

import (
	"fmt"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/codegen/service"
)

// TestServicePackageAliasRenderedVerbatim checks that the gRPC example server
// and CLI code referencing a service package qualify identifiers with the
// service package import alias exactly as computed. The alias "my_svc"
// contains an underscore, which jen.Qual would strip when deriving a
// qualifier from an import path, rendering "mysvc".
func TestServicePackageAliasRenderedVerbatim(t *testing.T) {
	const alias = "my_svc"
	cases := []struct {
		Name   string
		Render func() string
		Want   string
	}{
		{
			Name: "example-server-params",
			Render: func() string {
				services := []*ServiceData{{
					Service: &service.Data{
						VarName: "mySvc",
						PkgName: alias,
						Methods: []*service.MethodData{{Name: "get"}},
					},
				}}
				return fmt.Sprintf("%#v", jen.Func().Id("handleGRPCServer").Params(grpcExampleServerParams(services)...).Block())
			},
			Want: "mySvcEndpoints *my_svc.Endpoints",
		},
		{
			Name: "parse-endpoint-interceptors",
			Render: func() string {
				commands := []*cli.CommandData{{
					Interceptors: &cli.InterceptorData{VarName: "mySvcInterceptors", PkgName: alias},
				}}
				return fmt.Sprintf("%#v", jen.Func().Id("ParseEndpoint").Params(grpcParseEndpointParams(commands)...).Block())
			},
			Want: "mySvcInterceptors my_svc.ClientInterceptors",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			code := c.Render()
			require.Contains(t, code, c.Want)
			require.NotContains(t, code, "mysvc.")
		})
	}
}
