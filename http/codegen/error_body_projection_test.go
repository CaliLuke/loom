package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestErrorBodyProjectionAnalysis(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ErrorBodyProjectionDSL)
	data := CreateHTTPServices(root).Get("projection")
	for _, endpoint := range data.Endpoints {
		t.Run(endpoint.Method.Name, func(t *testing.T) {
			response := endpoint.Errors[0].Errors[0].Response
			field := "Body"
			if endpoint.Method.Name == "renamed" {
				field = "Content"
			} else if endpoint.Method.Name == "explicit" {
				field = ""
			}
			require.Equal(t, field, response.ResultAttr)
			require.Equal(t, field, response.ResultInit.ReturnTypeAttribute)
			require.Equal(t, endpoint.Method.Name == "optional", response.ResultInit.ReturnIsPrimitivePointer)
		})
	}
}

func TestErrorBodyProjectionEncoders(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ErrorBodyProjectionDSL)
	files := ServerFiles("example.com/projection/gen", CreateHTTPServices(root))
	sections := files[1].Section("error-encoder")
	require.Len(t, sections, 8)
	testutil.AssertGo(t, "testdata/golden/error_body_projection.go.golden", codegen.SectionsCode(t, sections))
}

func TestErrorBodyProjectionGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ErrorBodyProjectionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/projection", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "projection_test.go"), []byte(errorBodyProjectionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}
