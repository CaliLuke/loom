package generator

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
)

// TestTransportImportsImplicitHandshakeCustomDefaults renders the server file
// through the transport pipeline: implicit handshake defaults with custom Go
// representations, including an unmapped member, use the shared import owner.
func TestTransportImportsImplicitHandshakeCustomDefaults(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.API("probe", func() {})
		opening := dsl.Type("UnmappedOpening", func() {
			dsl.Attribute("trace", dsl.String, func() {
				dsl.Default("abc")
			})
			dsl.Attribute("token", dsl.String, func() {
				dsl.Meta("struct:field:type", "scalar.Token", "example.com/scalar")
				dsl.Default("token:default")
			})
			dsl.Attribute("hint", dsl.String, func() {
				dsl.Meta("struct:field:type", "scalar.Token", "example.com/scalar")
				dsl.Default("token:hint")
			})
		})
		dsl.Service("probe", func() {
			dsl.Method("watch", func() {
				dsl.StreamingResult(dsl.String)
				dsl.HTTP(func() {
					dsl.GET("/watch")
					dsl.ServerSentEvents()
					dsl.Response(dsl.StatusOK, opening, func() {
						dsl.Header("trace:X-Trace")
						dsl.Header("token:X-Token")
					})
				})
			})
		})
	})
	files, err := Transport("example.com/probe/gen", []eval.Root{root})
	require.NoError(t, err)
	dir := t.TempDir()
	rendered := 0
	for _, file := range files {
		if !strings.HasSuffix(file.Path, "http/probe/server/server.go") {
			continue
		}
		path, err := file.Render(dir)
		require.NoError(t, err)
		source, err := os.ReadFile(path)
		require.NoError(t, err)
		code := string(source)
		require.Contains(t, code, `"example.com/scalar"`)
		require.Contains(t, code, `loom.ParseStringValue[scalar.Token]("token:default", "default.token")`)
		require.Contains(t, code, `loom.ParseStringValue[scalar.Token]("token:hint", "default.hint")`)
		rendered++
	}
	require.Equal(t, 1, rendered)
}
