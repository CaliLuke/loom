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
	"github.com/stretchr/testify/require"
)

func TestExampleServerFiles(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"no-server", ctestdata.NoServerDSL},
		{"server-hosting-service-subset", ctestdata.ServerHostingServiceSubsetDSL},
		{"server-hosting-multiple-services", ctestdata.ServerHostingMultipleServicesDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// reset global variable
			example.Servers = make(example.ServersData)
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(service.NewServicesData(root))
			fs := ExampleServerFiles("", services)
			require.Greater(t, len(fs), 0)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), 0)
			var buf bytes.Buffer
			for _, s := range sections[1:] {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
			golden := filepath.Join("testdata", "server-"+c.Name+".golden")
			testutil.AssertGo(t, golden, code)
		})
	}
}
