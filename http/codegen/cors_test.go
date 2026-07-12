package codegen

import (
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	"github.com/CaliLuke/loom/codegen/testutil"
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
	require.Contains(t, mount, `loomhttp.HandleCORSPreflight`)
	testutil.CompareOrUpdateGolden(t, mount, filepath.Join("testdata", "golden", "cors_server-mount.golden"))

	handlers := codegentest.Sections(fs, "server.go", "server-handler")
	require.NotEmpty(t, handlers)
	handler := codegen.SectionCode(t, handlers[0])
	require.Contains(t, handler, `loomhttp.CORSHandler`)
	testutil.CompareOrUpdateGolden(t, handler, filepath.Join("testdata", "golden", "cors_server-handler.golden"))
}
