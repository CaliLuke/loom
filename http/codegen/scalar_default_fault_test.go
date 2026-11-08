package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestScalarTextDefaultFault requires an authored default of a string-backed
// custom text parameter to be materialized by the shared parameter default
// owner, in server request decoders and client response decoders alike. A
// default that cannot be parsed is a server fault, not the client-input
// invalid_field_type error reserved for supplied text, and its raw text keeps
// the authored validation.
func TestScalarTextDefaultFault(t *testing.T) {
	root := RunHTTPDSL(t, scalarTextDefaultDSL)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/scalars/gen", root)
	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dir, "gen", "http", "scalars", name))
		require.NoError(t, err)
		return string(data)
	}
	server, client := read("server/encode_decode.go"), read("client/encode_decode.go")

	for _, tc := range []struct {
		code, side, name, wire string
	}{
		{server, "server", "q", "q"},
		{server, "server", "h", "X-H"},
		{server, "server", "c", "c"},
		{client, "client", "h", "X-H"},
		{client, "client", "c", "c"},
	} {
		require.Contains(t, tc.code, `loom.Fault("%s", "parameter `+tc.wire+`: invalid default")`, "%s %s default is a fault of the shared owner", tc.side, tc.wire)
		require.Equal(t, 1, strings.Count(tc.code, `loom.InvalidFieldTypeError("`+tc.name+`"`), "%s %s: only supplied text is a client input error", tc.side, tc.name)
	}
	require.Contains(t, server, `"label:query"`, "the authored default text is retained")
	require.Equal(t, 2, strings.Count(server, `loom.InvalidLengthError("q"`), "raw-text validation applies to supplied text and to the default")
	for _, name := range []string{"server/encode_decode.go", "client/encode_decode.go"} {
		require.Empty(t, returnArityMismatches(t, filepath.Join(dir, "gen", "http", "scalars", name)), name)
	}
}

func scalarTextDefaultDSL() {
	label := func() {
		Meta("struct:field:type", "label.Label", "example.com/scalars/label")
	}
	Service("scalars", func() {
		Method("request", func() {
			Payload(func() {
				Attribute("q", String, func() {
					label()
					MinLength(7)
					Default("label:query")
				})
				Attribute("h", String, func() {
					label()
					Default("label:header")
				})
				Attribute("c", String, func() {
					label()
					Default("label:cookie")
				})
				Attribute("p", String, label)
			})
			HTTP(func() {
				GET("/request")
				Param("q")
				Param("p")
				Header("h:X-H")
				Cookie("c")
				Response(StatusNoContent)
			})
		})
		Method("response", func() {
			Result(func() {
				Attribute("h", String, func() {
					label()
					Default("label:header")
				})
				Attribute("c", String, func() {
					label()
					Default("label:cookie")
				})
			})
			HTTP(func() {
				GET("/response")
				Response(StatusOK, func() {
					Header("h:X-H")
					Cookie("c")
				})
			})
		})
	})
}
