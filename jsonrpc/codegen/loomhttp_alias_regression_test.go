package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestJSONRPCRenderedFilesUseLoomHTTPAlias(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMixedInitializeAndEventsStreamDSL)
	services := CreateJSONRPCServices(root)
	dir := t.TempDir()

	renderCodegenFiles(t, dir, ClientFiles("example.com/repro/gen", services))
	renderCodegenFiles(t, dir, ServerFiles("example.com/repro/gen", services))
	renderCodegenFiles(t, dir, SSEServerFiles("example.com/repro/gen", services))

	clientPath := filepath.Join(dir, "gen/jsonrpc/jsonrpc_mixed_initialize_events_stream_service/client/client.go")
	clientCode, err := os.ReadFile(clientPath)
	require.NoError(t, err)
	require.Contains(t, string(clientCode), "loomhttp.ErrRequestError(")
	require.NotContains(t, string(clientCode), "return nil, http.ErrRequestError(")

	encodeDecodePath := filepath.Join(dir, "gen/jsonrpc/jsonrpc_mixed_initialize_events_stream_service/client/encode_decode.go")
	encodeDecodeCode, err := os.ReadFile(encodeDecodePath)
	require.NoError(t, err)
	require.Contains(t, string(encodeDecodeCode), "loomhttp.ErrInvalidResponse(")
	require.NotContains(t, string(encodeDecodeCode), "http1.")

	ssePath := filepath.Join(dir, "gen/jsonrpc/jsonrpc_mixed_initialize_events_stream_service/server/stream.go")
	sseCode, err := os.ReadFile(ssePath)
	require.NoError(t, err)
	require.Contains(t, string(sseCode), "loomhttp.WriteJSONSSEEvent")
	require.NotContains(t, string(sseCode), "http1.")
}

func TestJSONRPCMixedSSEGeneratedModuleCompiles(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcEncodeDecodeDSL)
	dir := t.TempDir()

	renderJSONRPCModule(t, dir, "example.com/jsonrpcaliasit", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

func renderJSONRPCModule(t *testing.T, dir, modulePath string, root *expr.RootExpr) {
	t.Helper()

	genpkg := modulePath + "/gen"
	serviceData := servicecodegen.NewServicesData(root)
	jsonrpcData := CreateJSONRPCServices(root)

	files := make([]*cg.File, 0, len(root.Services)*2+5)
	userTypePkgs := make(map[string][]string)
	for _, svc := range root.Services {
		files = append(files, servicecodegen.Files(genpkg, svc, serviceData, userTypePkgs)...)
		files = append(files, servicecodegen.EndpointFile(genpkg, svc, serviceData))
	}
	files = append(files, PathFiles(jsonrpcData)...)
	files = append(files, ClientTypeFiles(genpkg, jsonrpcData)...)
	files = append(files, ClientFiles(genpkg, jsonrpcData)...)
	files = append(files, ServerTypeFiles(genpkg, jsonrpcData)...)
	files = append(files, ServerFiles(genpkg, jsonrpcData)...)

	renderCodegenFiles(t, dir, files)

	goMod := fmt.Sprintf(`module %s

go 1.27

require github.com/CaliLuke/loom v1.0.0

replace github.com/CaliLuke/loom => %s
`, modulePath, resolveJSONRPCTestLoomModulePath(t, dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644))
}

func renderCodegenFiles(t *testing.T, dir string, files []*cg.File) {
	t.Helper()

	for _, file := range files {
		_, err := file.Render(dir)
		require.NoErrorf(t, err, "render %s", file.Path)
	}
}

func runGoJSONRPCTestCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	out, err := runJSONRPCTestCommandAllowFailure(dir, "go", args...)
	require.NoErrorf(t, err, "go %v failed:\n%s", args, out)
}

func resolveJSONRPCTestLoomModulePath(t *testing.T, parentDir string) string {
	t.Helper()

	root, err := runJSONRPCTestCommandAllowFailure("", "git", "rev-parse", "--show-toplevel")
	require.NoError(t, err)
	repoRoot := strings.TrimSpace(root)
	if repoRoot != "" {
		return repoRoot
	}

	commitOut, err := runJSONRPCTestCommandAllowFailure("", "git", "rev-parse", "HEAD")
	require.NoError(t, err)
	commit := strings.TrimSpace(commitOut)
	remote := resolveJSONRPCTestRemoteURL(t)
	dest := filepath.Join(parentDir, "loom-pinned")

	_, err = runJSONRPCTestCommandAllowFailure("", "git", "init", dest)
	require.NoError(t, err)
	_, err = runJSONRPCTestCommandAllowFailure(dest, "git", "remote", "add", "origin", remote)
	require.NoError(t, err)
	_, err = runJSONRPCTestCommandAllowFailure(dest, "git", "fetch", "--depth", "1", "origin", commit)
	require.NoError(t, err)
	_, err = runJSONRPCTestCommandAllowFailure(dest, "git", "checkout", "--detach", "FETCH_HEAD")
	require.NoError(t, err)

	return dest
}

func resolveJSONRPCTestRemoteURL(t *testing.T) string {
	t.Helper()

	for _, name := range []string{"origin", "fork"} {
		out, err := runJSONRPCTestCommandAllowFailure("", "git", "remote", "get-url", name)
		if err == nil {
			url := strings.TrimSpace(out)
			if url != "" {
				return url
			}
		}
	}

	t.Fatal("could not resolve git remote URL from fork or origin")
	return ""
}

func runJSONRPCTestCommandAllowFailure(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...) // #nosec G204 -- test executes fixed toolchain commands with controlled args
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
