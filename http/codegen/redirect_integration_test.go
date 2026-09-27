package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/codegentest"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const mixedRedirectHarness = `package redirect_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	loomhttp "github.com/CaliLuke/loom/http"
	svc "example.com/redirect/gen/server_mixed"
	server "example.com/redirect/gen/http/server_mixed/server"
)

func TestRoutes(t *testing.T) {
	fileSystem := http.FS(fstest.MapFS{
		"path/to/file1.json": &fstest.MapFile{Data: []byte("regular-file")},
	})
	mux := loomhttp.NewMuxer()
	var normalCalls, redirectCalls int
	h := server.New(&svc.Endpoints{
		MethodMixed1: func(context.Context, any) (any, error) {
			normalCalls++
			return nil, nil
		},
		MethodMixed2: func(context.Context, any) (any, error) {
			redirectCalls++
			return nil, nil
		},
	}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, fileSystem, fileSystem)
	server.Mount(mux, h)
	for _, tc := range []struct {
		path string
		status int
		location string
		body string
	}{
		{"/resources1/123", http.StatusNoContent, "", ""},
		{"/resources2/123", http.StatusMovedPermanently, "/redirect/dest1", ""},
		{"/file1.json", http.StatusOK, "", "regular-file"},
		{"/file2.json", http.StatusMovedPermanently, "/redirect/dest2", ""},
	} {
		t.Run(tc.path, func(t *testing.T) {
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equal(t, tc.status, r.Code)
			require.Equal(t, tc.location, r.Header().Get("Location"))
			if tc.body != "" {
				require.Equal(t, tc.body, r.Body.String())
			}
		})
	}
	require.Equal(t, 1, normalCalls)
	require.Zero(t, redirectCalls)
}
`

const payloadRedirectHarness = `package redirect_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	loomhttp "github.com/CaliLuke/loom/http"
	server "example.com/redirect/gen/http/service_payload_no_result/server"
)

func TestDecodeBeforeRedirect(t *testing.T) {
	var calls int
	h := server.NewMethodPayloadNoResultHandler(func(context.Context, any) (any, error) {
		calls++
		return nil, nil
	}, loomhttp.NewMuxer(), loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	for _, tc := range []struct {
		body string
		status int
		location string
	}{
		{"", http.StatusBadRequest, ""},
		{"{}", http.StatusMovedPermanently, "/redirect/dest"},
		{"{\"a\":true}", http.StatusMovedPermanently, "/redirect/dest"},
		{"{\"a\":", http.StatusBadRequest, ""},
	} {
		t.Run(tc.body, func(t *testing.T) {
			r := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			h.ServeHTTP(r, req)
			require.Equal(t, tc.status, r.Code)
			require.Equal(t, tc.location, r.Header().Get("Location"))
		})
	}
	require.Zero(t, calls)
}
`

func TestRedirectHandlerContext(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"without payload", testdata.ServerNoPayloadNoResultWithRedirectDSL},
		{"with payload", testdata.ServerPayloadNoResultWithRedirectDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			files := ServerFiles("gen", CreateHTTPServices(root))
			sections := codegentest.Sections(files, "server.go", "server-handler-init")
			require.Len(t, sections, 1)
			code := codegen.SectionCode(t, sections[0])
			require.NotContains(t, code, "ctx := lifecycle.Context()")
			require.Contains(t, code, "defer lifecycle.End()")
			require.Contains(t, code, "w = lifecycle.Writer()")
			require.Contains(t, code, "http.Redirect(w, r,")
		})
	}
}

func TestGeneratedRedirectHandlers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		design  func()
		harness string
	}{
		{"mixed routes and files", testdata.ServerMixedDSL, mixedRedirectHarness},
		{"payload decoding", testdata.ServerPayloadNoResultWithRedirectDSL, payloadRedirectHarness},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/redirect", RunHTTPDSL(t, tc.design))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "redirect_test.go"), []byte(tc.harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}
