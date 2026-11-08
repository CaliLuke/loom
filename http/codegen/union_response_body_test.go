package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUnionResponseBodyConstructors(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UnionResponseBodyDSL)
	services := CreateHTTPServices(root)
	for _, svc := range root.API.HTTP.Services {
		service := services.Get(svc.Name())
		endpoint := service.Endpoints[0]
		t.Run(service.Service.Name, func(t *testing.T) {
			wantValue := strings.HasPrefix(service.Service.Name, "required") && !strings.Contains(service.Service.Name, "nullable")
			for _, response := range endpoint.Result.Responses {
				want := wantValue
				require.Equal(t, want, response.ResultInit.ReturnIsUnionValue, "success constructor")
				code := codegen.SectionCode(t, constructorSection("test", response.ResultInit, true, false, service.Scope))
				if want {
					require.Contains(t, code, "Choice: *v,")
				} else {
					require.Contains(t, code, "Choice: v,")
				}
			}
			for _, group := range endpoint.Errors {
				for _, response := range group.Errors {
					require.Equal(t, wantValue, response.Response.ResultInit.ReturnIsUnionValue, "error constructor")
					require.False(t, response.Response.ResultInit.ReturnIsPrimitivePointer, "union transform already returns a pointer")
					code := codegen.SectionCode(t, constructorSection("test", response.Response.ResultInit, true, false, service.Scope))
					if wantValue {
						require.Contains(t, code, "Choice: *v,")
					} else {
						require.Contains(t, code, "Choice: v,")
					}
				}
			}
		})
	}
}

func TestUnionResponseBodyGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UnionResponseBodyDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/unionresponse", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(testdata.UnionResponseBodyHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}

func TestUnionResponseStatusNames(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UnionResponseStatusDSL)
	service := CreateHTTPServices(root).Get("required_named")
	responses := service.Endpoints[0].Result.Responses
	require.Len(t, responses, 2)
	for _, response := range responses {
		suffix := "OK"
		if response.LiteralCode() == 299 {
			suffix = "Status299"
		} else {
			require.Equal(t, 200, response.LiteralCode())
		}
		require.Equal(t, "Show"+suffix+"ResponseBody", response.ClientBody.VarName)
		require.Contains(t, response.ResultInit.Name, suffix)
		require.True(t, response.ResultInit.ReturnIsUnionValue)
	}
	require.Equal(t, "http.StatusBadRequest", service.Endpoints[0].Errors[0].Errors[0].Response.LiteralStatus())
}
