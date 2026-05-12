package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/testutil"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestHandlerInit(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"no payload no result", testdata.ServerNoPayloadNoResultDSL},
		{"no payload no result with a redirect", testdata.ServerNoPayloadNoResultWithRedirectDSL},
		{"payload no result", testdata.ServerPayloadNoResultDSL},
		{"payload no result with a redirect", testdata.ServerPayloadNoResultWithRedirectDSL},
		{"no payload result", testdata.ServerNoPayloadResultDSL},
		{"payload result", testdata.ServerPayloadResultDSL},
		{"payload result error", testdata.ServerPayloadResultErrorDSL},
		{"skip response body encode decode", testdata.ServerSkipResponseBodyEncodeDecodeDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ServerFiles(genpkg, services)
			sections := codegentest.Sections(fs, "server.go", "server-handler-init")
			require.Greater(t, len(sections), 0)
			code := codegen.SectionCode(t, sections[0])
			testutil.AssertGo(t, "testdata/golden/handler_"+c.Name+".go.golden", code)
		})
	}
}

// TestHandlerInitConstructorSignature pins the generated handler-constructor
// parameter list so observer wiring stays source-compatible: callers of
// `New...Handler` from prior Loom releases must continue to compile.
func TestHandlerInitConstructorSignature(t *testing.T) {
	const genpkg = "gen"
	cases := []struct {
		Name        string
		DSL         func()
		WantParams  []string
		WantImports []string
	}{
		{
			Name: "no payload no result",
			DSL:  testdata.ServerNoPayloadNoResultDSL,
			WantParams: []string{
				"endpoint loom.Endpoint",
				"mux loomhttp.Muxer",
				"decoder func(*http.Request) loomhttp.Decoder",
				"encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder",
				"errhandler func(context.Context, http.ResponseWriter, error)",
				"formatter func(ctx context.Context, err error) loomhttp.Statuser",
			},
			WantImports: []string{
				"github.com/CaliLuke/loom/observability/transport",
				"\"time\"",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, c.DSL)
			services := CreateHTTPServices(root)
			fs := ServerFiles(genpkg, services)
			sections := codegentest.Sections(fs, "server.go", "server-handler-init")
			require.Greater(t, len(sections), 0)
			code := codegen.SectionCode(t, sections[0])
			for _, p := range c.WantParams {
				require.Contains(t, code, p, "generated handler init must keep %q in its parameter list", p)
			}
			require.Contains(t, code, "loomtransport.BeginHTTPRequest(ctx, w,", "generated handler must wire the observer")
			require.Contains(t, code, "defer obs.End()", "generated handler must defer obs.End()")
		})
	}
}
