package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedFileServerSupportsSPACatchAllAndDirectoryTargets(t *testing.T) {
	const modulePath = "example.com/fileserverit"

	root := RunHTTPDSL(t, fileServerIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "file_server_test.go"), []byte(fileServerIntegrationHarness), 0o600); err != nil {
		t.Fatalf("write file server harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func fileServerIntegrationDSL() {
	Service("spa", func() {
		HTTP(func() {
			Path("/one")
			Path("/two")
		})
		Method("GetData", func() {
			Result(String)
			HTTP(func() {
				GET("/api/data")
			})
		})
		Files("/index.html", "./dist/index.html")
		Files("/first/home", "./dist/first.html")
		Files("/second/home", "./dist/second.html")
		Files("/assets/{*path}", "./dist/assets")
		Files("/{*path}", "./dist/index.html")
	})
}

const fileServerIntegrationHarness = `package fileserverit_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	spa "example.com/fileserverit/gen/spa"
	spaserver "example.com/fileserverit/gen/http/spa/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestGeneratedFileServer(t *testing.T) {
	modified := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	fileSystem := http.FS(fstest.MapFS{
		"dist/index.html":    &fstest.MapFile{Data: []byte("spa-index"), ModTime: modified},
		"dist/first.html": &fstest.MapFile{Data: []byte("first")},
		"dist/second.html": &fstest.MapFile{Data: []byte("second")},
		"dist/assets/app.js": &fstest.MapFile{Data: []byte("asset")},
	})
	mux := loomhttp.NewMuxer()
	server := spaserver.New(
		&spa.Endpoints{},
		mux,
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		nil,
		nil,
		fileSystem,
		fileSystem,
		fileSystem,
		fileSystem,
		fileSystem,
	)
	spaserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	client := httpServer.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	tests := []struct {
		name string
		path string
		want string
		status int
		header string
		value string
	}{
		{name: "exact index", path: "/one/index.html", want: "spa-index"},
		{name: "second exact index", path: "/two/index.html", want: "spa-index"},
		{name: "first alias", path: "/one/first/home", want: "first"},
		{name: "second alias", path: "/one/second/home", want: "second"},
		{name: "exact range", path: "/one/index.html", want: "spa", status: http.StatusPartialContent, header: "Range", value: "bytes=0-2"},
		{name: "exact conditional", path: "/one/index.html", status: http.StatusNotModified, header: "If-Modified-Since", value: modified.Format(http.TimeFormat)},
		{name: "missing directory child", path: "/one/assets/missing", want: "404 page not found\n", status: http.StatusNotFound},
		{name: "root SPA route", path: "/one/", want: "spa-index"},
		{name: "nested SPA route", path: "/one/settings", want: "spa-index"},
		{name: "directory asset", path: "/one/assets/app.js", want: "asset"},
		{name: "second root", path: "/two/", want: "spa-index"},
		{name: "second nested", path: "/two/settings", want: "spa-index"},
		{name: "second directory", path: "/two/assets/app.js", want: "asset"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, httpServer.URL + test.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if test.header != "" {
				request.Header.Set(test.header, test.value)
			}
			response, err := client.Do(request)
			if err != nil {
				t.Fatalf("GET %s: %v", test.path, err)
			}
			body, err := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if err != nil {
				t.Fatalf("read %s response: %v", test.path, err)
			}
			if closeErr != nil {
				t.Fatalf("close %s response: %v", test.path, closeErr)
			}
			wantStatus := test.status
			if wantStatus == 0 {
				wantStatus = http.StatusOK
			}
			if response.StatusCode != wantStatus {
				t.Errorf("GET %s status = %d, want %d; body = %q", test.path, response.StatusCode, wantStatus, body)
			}
			if test.path == "/one/index.html" && wantStatus != http.StatusNotModified {
				if got := response.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
					t.Errorf("Content-Type = %q", got)
				}
			}
			if string(body) != test.want {
				t.Errorf("GET %s body = %q, want %q", test.path, body, test.want)
			}
		})
	}
}
`
