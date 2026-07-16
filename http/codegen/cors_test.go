package codegen

import (
	"path/filepath"
	"strings"
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
	testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
		StringContent(mount).
		Path("cors_server-mount.golden").
		CompareContent()

	handlers := codegentest.Sections(fs, "server.go", "server-handler")
	require.NotEmpty(t, handlers)
	handler := codegen.SectionCode(t, handlers[0])
	require.Contains(t, handler, `loomhttp.CORSHandler`)
	testutil.NewGoldenFile(t, filepath.Join("testdata", "golden")).
		StringContent(handler).
		Path("cors_server-handler.golden").
		CompareContent()
}

func TestServerRuntimeCORSOutput(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ServerRuntimeCORSPolicyDSL)
	fs := ServerFiles("gen", CreateHTTPServices(root))

	initSections := codegentest.Sections(fs, "server.go", "server-init")
	require.NotEmpty(t, initSections)
	initCode := codegen.SectionCode(t, initSections[0])
	require.Contains(t, initCode, "corsPolicy loomhttp.RuntimeCORSPolicy")

	mountSections := codegentest.Sections(fs, "server.go", "server-mount")
	require.NotEmpty(t, mountSections)
	mountCode := codegen.SectionCode(t, mountSections[0])
	require.Contains(t, mountCode, "corsPolicy.HandlePreflight")

	handlerSections := codegentest.Sections(fs, "server.go", "server-handler")
	require.NotEmpty(t, handlerSections)
	for _, section := range handlerSections {
		handlerCode := codegen.SectionCode(t, section)
		require.Contains(t, handlerCode, "corsPolicy.Handler")
		require.NotContains(t, handlerCode, "CORSPolicy{Origins:")
	}
}

func TestServerRuntimeCORSGeneratedModuleCompiles(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ServerRuntimeCORSPolicyDSL)
	dir := t.TempDir()
	repoRoot := runCommand(t, "", "git", "rev-parse", "--show-toplevel")
	t.Setenv("LOOM_DIR", filepath.Clean(strings.TrimSpace(repoRoot)))
	renderHTTPModule(t, dir, "example.com/runtimecors", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
