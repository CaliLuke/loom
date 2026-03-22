package codegen

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/v3/codegen"
	"github.com/CaliLuke/loom/v3/codegen/testutil"
	"github.com/CaliLuke/loom/v3/grpc/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestParseEndpointWithInterceptors(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{
			Name: "endpoint-with-interceptors",
			DSL:  testdata.InterceptorsDSL,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			fs := ClientCLIFiles("", services)
			require.Greater(t, len(fs), 1, "expected at least 2 files")
			require.NotEmpty(t, fs[0].AllSections())
			var buf bytes.Buffer
			for _, s := range fs[0].AllSections() {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, buf.String())
			golden := filepath.Join("testdata", "endpoint-"+c.Name+".golden")
			testutil.AssertGo(t, golden, code)
		})
	}
}
