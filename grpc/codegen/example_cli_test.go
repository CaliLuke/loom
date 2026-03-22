package codegen

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	ctestdata "github.com/CaliLuke/loom/codegen/example/testdata"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestExampleCLIFiles(t *testing.T) {
	cases := []struct {
		Name    string
		DSL     func()
		PkgPath string
	}{
		{"no-server", ctestdata.NoServerDSL, ""},
		{"server-hosting-service-subset", ctestdata.ServerHostingServiceSubsetDSL, ""},
		{"server-hosting-multiple-services", ctestdata.ServerHostingMultipleServicesDSL, ""},
		{"no-server-pkgpath", ctestdata.NoServerDSL, "my/pkg/path"},
		{"server-hosting-service-subset-pkgpath", ctestdata.ServerHostingServiceSubsetDSL, "my/pkg/path"},
		{"server-hosting-multiple-services-pkgpath", ctestdata.ServerHostingMultipleServicesDSL, "my/pkg/path"},
		{"interceptors", testdata.InterceptorsDSL, ""},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// reset global variable
			example.Servers = make(example.ServersData)
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(service.NewServicesData(root))
			fs := ExampleCLIFiles(c.PkgPath, services)
			require.Greater(t, len(fs), 0)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), 0)
			var buf bytes.Buffer
			for _, s := range sections {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, buf.String())
			golden := filepath.Join("testdata", "client-"+c.Name+".golden")
			testutil.AssertGo(t, golden, code)
		})
	}
}
