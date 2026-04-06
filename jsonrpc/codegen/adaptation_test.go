package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
)

func TestJSONRPCAdaptationHelpers(t *testing.T) {
	t.Run("transport path rewrite", func(t *testing.T) {
		assert.Equal(t, "gen/jsonrpc/calc/client/encode_decode.go", rewriteJSONRPCTransportPath("gen/http/calc/client/encode_decode.go"))
	})

	t.Run("example cli path rewrite", func(t *testing.T) {
		assert.Equal(t, filepath.Join("cmd", "calc", "jsonrpc.go"), rewriteJSONRPCExampleCLIPath(filepath.Join("cmd", "calc", "http.go")))
	})

	t.Run("cli parse endpoint rewrite", func(t *testing.T) {
		source := "func parse({{ .VarName }}Configurer *{{ .PkgName }}.ConnConfigurer, x int) {}\ncall(x, {{ .VarName }}Configurer{{ end }})"
		rewritten := rewriteJSONRPCCLIParseEndpointSource(source)
		assert.Contains(t, rewritten, "{{ .VarName }}ConfigFn loomhttp.ConnConfigureFunc,")
		assert.Contains(t, rewritten, ", {{ .VarName }}ConfigFn{{ end }}")
		assert.NotContains(t, rewritten, "ConnConfigurer")
	})

	t.Run("example cli source rewrite", func(t *testing.T) {
		rewritten := rewriteJSONRPCExampleCLISource("doHTTP()\nhttpUsage()\n")
		assert.Contains(t, rewritten, "doJSONRPC()")
		assert.Contains(t, rewritten, "jsonrpcUsage()")
	})

	t.Run("section source rewrite keeps headers and rewrites others", func(t *testing.T) {
		sections := []codegen.Section{
			&codegen.SectionTemplate{Name: "source-header", Source: "header"},
			&codegen.SectionTemplate{Name: "parse-endpoint", Source: "doHTTP"},
			codegen.NewRenderSection("usage", func() string { return "httpUsage" }),
		}
		updated := rewriteJSONRPCSectionSources(sections, rewriteJSONRPCExampleCLISource)
		require.Len(t, updated, 3)

		header, ok := updated[0].(*codegen.SectionTemplate)
		require.True(t, ok)
		assert.Equal(t, "header", header.Source)

		parse, ok := updated[1].(*codegen.SectionTemplate)
		require.True(t, ok)
		assert.Equal(t, "doJSONRPC", parse.Source)

		assert.Equal(t, "usage", updated[2].SectionName())
		assert.Contains(t, renderSectionSource(updated[2]), "jsonrpcUsage")
	})
}
