package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestServerCORSOutput(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ServerCORSPolicyDSL)
	services := CreateHTTPServices(root)
	fs := ServerFiles("gen", services)

	mounts := codegentest.Sections(fs, "server.go", "server-mount")
	require.NotEmpty(t, mounts)
	mount := codegen.SectionCode(t, mounts[0])
	require.Contains(t, mount, `mux.Handle("OPTIONS", "/items"`)
	require.Contains(t, mount, `loomhttp.HandleCORSPreflight`)
	require.Contains(t, mount, `[]string{"GET", "POST"}`)

	handlers := codegentest.Sections(fs, "server.go", "server-handler")
	require.NotEmpty(t, handlers)
	handler := codegen.SectionCode(t, handlers[0])
	require.Contains(t, handler, `loomhttp.CORSHandler`)
	require.Contains(t, handler, `https://app.example.com`)
}
