package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
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

func TestStaticFileTargetPreservesHTTPContract(t *testing.T) {
	modified := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	fileSystem := http.FS(fstest.MapFS{
		"dist/index.html": &fstest.MapFile{Data: []byte("spa-index"), ModTime: modified},
	})
	tests := []struct {
		name       string
		method     string
		header     string
		value      string
		wantStatus int
		wantBody   string
	}{
		{"get", http.MethodGet, "", "", http.StatusOK, "spa-index"},
		{"head", http.MethodHead, "", "", http.StatusOK, ""},
		{"range", http.MethodGet, "Range", "bytes=0-2", http.StatusPartialContent, "spa"},
		{"invalid range", http.MethodGet, "Range", "bytes=99-100", http.StatusRequestedRangeNotSatisfiable, "invalid range: failed to overlap\n"},
		{"not modified", http.MethodGet, "If-Modified-Since", modified.Format(http.TimeFormat), http.StatusNotModified, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/nested/%69ndex.html?keep=1", nil)
			if test.header != "" {
				request.Header.Set(test.header, test.value)
			}
			originalURL := *request.URL
			response := httptest.NewRecorder()
			loomhttp.NewStaticFileServer(fileSystem, "./dist/index.html").ServeHTTP(response, request)
			require.Equal(t, test.wantStatus, response.Code)
			require.Equal(t, test.wantBody, response.Body.String())
			require.Equal(t, originalURL, *request.URL)
			require.Empty(t, response.Header().Get("Location"))
			if response.Code == http.StatusOK || response.Code == http.StatusPartialContent {
				require.Equal(t, "text/html; charset=utf-8", response.Header().Get("Content-Type"))
				require.Equal(t, modified.Format(http.TimeFormat), response.Header().Get("Last-Modified"))
			}
			if response.Code == http.StatusPartialContent {
				require.Equal(t, "bytes 0-2/9", response.Header().Get("Content-Range"))
			}
		})
	}
}
