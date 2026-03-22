package codegen

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	ctestdata "github.com/CaliLuke/loom/codegen/example/testdata"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestExampleCLIFiles(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"no-server", ctestdata.NoServerDSL},
		{"server-hosting-service-subset", ctestdata.ServerHostingServiceSubsetDSL},
		{"server-hosting-multiple-services", ctestdata.ServerHostingMultipleServicesDSL},
		{"streaming", testdata.StreamingResultDSL},
		{"streaming-multiple-services", testdata.StreamingMultipleServicesDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// reset global variable
			example.Servers = make(example.ServersData)
			root := codegen.RunDSL(t, c.DSL)
			httpServices := NewServicesData(service.NewServicesData(root), root.API.HTTP)
			fs := ExampleCLIFiles("", httpServices)
			require.Len(t, fs, 1)
			sections := fs[0].AllSections()
			require.Greater(t, len(sections), 0)
			var buf bytes.Buffer
			for _, s := range sections[1:] {
				require.NoError(t, s.Write(&buf))
			}
			code := codegen.FormatTestCode(t, "package foo\n"+buf.String())
			golden := filepath.Join("testdata", "golden", "client-"+c.Name+".golden")
			testutil.CompareOrUpdateGolden(t, code, golden)
		})
	}
}
