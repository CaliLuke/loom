package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/testutil"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestServerHandler(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"server simple routing", testdata.ServerSimpleRoutingDSL},
		{"server trailing slash routing", testdata.ServerTrailingSlashRoutingDSL},
		{"server simple routing with a redirect", testdata.ServerSimpleRoutingWithRedirectDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ServerFiles(genpkg, services)
			sections := codegentest.Sections(fs, "server.go", "server-handler")
			require.Greater(t, len(sections), 0)
			code := codegen.SectionCode(t, sections[0])
			testutil.AssertGo(t, "testdata/golden/server_handler_"+c.Name+".go.golden", code)
		})
	}
}

func TestServerHandlerDelegatesMountingToRuntime(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ServerSimpleRoutingDSL)
	services := CreateHTTPServices(root)
	files := ServerFiles("gen", services)
	sections := codegentest.Sections(files, "server.go", "server-handler")
	require.NotEmpty(t, sections)
	code := codegen.SectionCode(t, sections[0])

	require.Contains(t, code, `loomhttp.MountHandler(mux, "GET", "/simple/routing", h)`)
	require.NotContains(t, code, `h.(http.HandlerFunc)`)
}
