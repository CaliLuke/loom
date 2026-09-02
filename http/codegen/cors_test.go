package codegen

import (
	"os"
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

func TestServerCORSWithAuthoredOptionsOutput(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ServerCORSOptionsPolicyDSL)
	fs := ServerFiles("gen", CreateHTTPServices(root))

	mountSections := codegentest.Sections(fs, "server.go", "server-mount")
	require.NotEmpty(t, mountSections)
	mountCode := codegen.SectionCode(t, mountSections[0])
	require.Contains(t, mountCode, "loomhttp.CORSOptionsHandler")
	require.Equal(t, 1, strings.Count(mountCode, `mux.Handle("OPTIONS", "/items",`))
	require.NotContains(t, mountCode, "MountOptionsHandler(mux")
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
	require.Contains(t, mountCode, "corsPolicy.OptionsHandler")
	require.Equal(t, 1, strings.Count(mountCode, `mux.Handle("OPTIONS", "/items",`))
	require.NotContains(t, mountCode, "MountOptionsHandler(mux")

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
	if err := os.WriteFile(filepath.Join(dir, "runtime_cors_test.go"), []byte(runtimeCORSOptionsHarness), 0o600); err != nil {
		t.Fatalf("write runtime CORS harness: %v", err)
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

const runtimeCORSOptionsHarness = `package runtimecors_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	runtimecorsservice "example.com/runtimecors/gen/runtime_cors_service"
	runtimecorsserver "example.com/runtimecors/gen/http/runtime_cors_service/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestGeneratedRuntimeCORSDispatchesOPTIONSRequests(t *testing.T) {
	var calls atomic.Int32
	var middlewareCalls atomic.Int32
	endpoints := &runtimecorsservice.Endpoints{
		Options: func(context.Context, any) (any, error) {
			calls.Add(1)
			return "metadata", nil
		},
	}
	policy, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{{
		Pattern: "https://app.example.com",
	}}})
	if err != nil {
		t.Fatalf("create CORS policy: %v", err)
	}
	mux := loomhttp.NewMuxer()
	generated := runtimecorsserver.New(
		endpoints,
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
		policy,
	)
	generated.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalls.Add(1)
			next.ServeHTTP(w, r)
		})
	})
	runtimecorsserver.Mount(mux, generated)
	server := httptest.NewServer(mux)
	defer server.Close()

	ordinary := sendOPTIONS(t, server, false)
	defer ordinary.Body.Close()
	if ordinary.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(ordinary.Body)
		t.Fatalf("ordinary OPTIONS status = %d, want 200; body: %s", ordinary.StatusCode, body)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("ordinary OPTIONS endpoint calls = %d, want 1", got)
	}
	if got := middlewareCalls.Load(); got != 1 {
		t.Fatalf("ordinary OPTIONS middleware calls = %d, want 1", got)
	}
	if got := ordinary.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("ordinary OPTIONS Allow-Origin = %q", got)
	}

	preflight := sendOPTIONS(t, server, true)
	defer preflight.Body.Close()
	if preflight.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(preflight.Body)
		t.Fatalf("preflight status = %d, want 204; body: %s", preflight.StatusCode, body)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("endpoint calls after preflight = %d, want 1", got)
	}
	if got := middlewareCalls.Load(); got != 1 {
		t.Fatalf("middleware calls after preflight = %d, want 1", got)
	}
	if got := preflight.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
		t.Errorf("preflight Allow-Methods = %q, want GET", got)
	}
}

func sendOPTIONS(t *testing.T, server *httptest.Server, preflight bool) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodOptions, server.URL+"/items", nil)
	if err != nil {
		t.Fatalf("create OPTIONS request: %v", err)
	}
	request.Header.Set("Origin", "https://app.example.com")
	if preflight {
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("send OPTIONS request: %v", err)
	}
	return response
}
`
