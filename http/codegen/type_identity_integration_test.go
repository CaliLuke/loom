package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseTypesDoNotInheritRequestValidators(t *testing.T) {
	root := RunHTTPDSL(t, testdata.TypeIdentityDSL)
	data := CreateHTTPServices(root).Get("probe")
	for _, endpoint := range data.Endpoints {
		for _, response := range endpoint.Result.Responses {
			for _, body := range response.ServerBody {
				require.Empty(t, body.ValidateDef, "response body %s", body.Name)
			}
		}
		for _, group := range endpoint.Errors {
			for _, response := range group.Errors {
				for _, body := range response.Response.ServerBody {
					require.Empty(t, body.ValidateDef, "error body %s", body.Name)
				}
			}
		}
	}
}

func TestTypeIdentityGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.TypeIdentityDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/identity", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "identity_test.go"), []byte(testdata.TypeIdentityHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}
