package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestNewStaticFileServer(t *testing.T) {
	fileSystem := http.FS(fstest.MapFS{
		"dist/index.html":    &fstest.MapFile{Data: []byte("spa-index")},
		"dist/assets/app.js": &fstest.MapFile{Data: []byte("asset")},
	})
	tests := []struct {
		name        string
		target      string
		requestPath string
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "single file at root",
			target:      "./dist/index.html",
			requestPath: "/",
			wantStatus:  http.StatusOK,
			wantBody:    "spa-index",
		},
		{
			name:        "single file at nested route",
			target:      "./dist/index.html",
			requestPath: "/settings",
			wantStatus:  http.StatusOK,
			wantBody:    "spa-index",
		},
		{
			name:        "single index file path does not redirect",
			target:      "./dist/index.html",
			requestPath: "/index.html",
			wantStatus:  http.StatusOK,
			wantBody:    "spa-index",
		},
		{
			name:        "directory child",
			target:      "./dist/assets",
			requestPath: "/app.js",
			wantStatus:  http.StatusOK,
			wantBody:    "asset",
		},
		{
			name:        "missing target",
			target:      "./dist/missing",
			requestPath: "/anything",
			wantStatus:  http.StatusNotFound,
			wantBody:    "404 page not found\n",
		},
		{
			name:        "missing directory child",
			target:      "./dist/assets",
			requestPath: "/missing.js",
			wantStatus:  http.StatusNotFound,
			wantBody:    "404 page not found\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := loomhttp.NewStaticFileServer(fileSystem, test.target)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.requestPath, nil)

			handler.ServeHTTP(response, request)

			require.Equal(t, test.wantStatus, response.Code)
			require.Equal(t, test.wantBody, response.Body.String())
		})
	}
}
