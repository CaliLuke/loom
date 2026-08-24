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
		Method("GetData", func() {
			Result(String)
			HTTP(func() {
				GET("/api/data")
			})
		})
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

	spa "example.com/fileserverit/gen/spa"
	spaserver "example.com/fileserverit/gen/http/spa/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestGeneratedFileServer(t *testing.T) {
	fileSystem := http.FS(fstest.MapFS{
		"dist/index.html":    &fstest.MapFile{Data: []byte("spa-index")},
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
	)
	spaserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "root SPA route", path: "/", want: "spa-index"},
		{name: "nested SPA route", path: "/settings", want: "spa-index"},
		{name: "directory asset", path: "/assets/app.js", want: "asset"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := httpServer.Client().Get(httpServer.URL + test.path)
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
			if response.StatusCode != http.StatusOK {
				t.Errorf("GET %s status = %d, want %d; body = %q", test.path, response.StatusCode, http.StatusOK, body)
			}
			if string(body) != test.want {
				t.Errorf("GET %s body = %q, want %q", test.path, body, test.want)
			}
		})
	}
}
`
