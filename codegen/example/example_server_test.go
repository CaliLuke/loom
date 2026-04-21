package example

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example/testdata"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
)

// compareOrUpdateGolden delegates to the shared testutil helper. Pass -update
// (or -u) on the go-test command line to rewrite golden files.
func compareOrUpdateGolden(t *testing.T, code, golden string) {
	t.Helper()
	testutil.CompareOrUpdateGolden(t, code, golden)
}

func TestExampleServerFiles(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"no-server", testdata.NoServerDSL},
		{"same-api-service-name", testdata.SameAPIServiceNameDSL},
		{"single-server-single-host", testdata.SingleServerSingleHostDSL},
		{"single-server-single-host-with-variables", testdata.SingleServerSingleHostWithVariablesDSL},
		{"server-hosting-service-with-file-server", testdata.ServerHostingServiceWithFileServerDSL},
		{"server-hosting-service-subset", testdata.ServerHostingServiceSubsetDSL},
		{"server-hosting-multiple-services", testdata.ServerHostingMultipleServicesDSL},
		{"single-server-multiple-hosts", testdata.SingleServerMultipleHostsDSL},
		{"single-server-multiple-hosts-with-variables", testdata.SingleServerMultipleHostsWithVariablesDSL},
		{"service-name-with-spaces", testdata.NamesWithSpacesDSL},
		{"service-for-only-http", testdata.ServiceForOnlyHTTPDSL},
		{"sercice-for-only-grpc", testdata.ServiceForOnlyGRPCDSL},
		{"service-for-http-and-part-of-grpc", testdata.ServiceForHTTPAndPartOfGRPCDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := service.NewServicesData(root)
			fs := ServerFiles("", root, services)
			require.Len(t, fs, 1)
			require.Greater(t, len(fs[0].AllSections()), 0)
			var buf bytes.Buffer
			for _, s := range fs[0].AllSections()[1:] {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
			golden := filepath.Join("testdata", "server-"+c.Name+".golden")
			compareOrUpdateGolden(t, code, golden)
		})
	}
}
