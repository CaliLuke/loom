package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestFileResponseDelegatesHTTPContentSemantics(t *testing.T) {
	modTime := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name              string
		method            string
		headers           map[string]string
		wantStatus        int
		wantBody          string
		checkBody         bool
		wantRanges        bool
		wantContentRange  string
		checkContentRange bool
		wantContentType   string
		wantTypePrefix    string
	}{
		{name: "full content", method: http.MethodGet, wantStatus: http.StatusOK, wantBody: "abcdef", checkBody: true, wantRanges: true, wantContentType: "text/plain; charset=utf-8"},
		{
			name:       "range",
			method:     http.MethodGet,
			headers:    map[string]string{"Range": "bytes=1-3"},
			wantStatus: http.StatusPartialContent,
			wantBody:   "bcd",
			checkBody:  true,
			wantRanges: true,
		},
		{
			name:              "multiple ranges",
			method:            http.MethodGet,
			headers:           map[string]string{"Range": "bytes=0-0,2-3"},
			wantStatus:        http.StatusPartialContent,
			wantRanges:        true,
			checkContentRange: true,
			wantTypePrefix:    "multipart/byteranges; boundary=",
		},
		{name: "head", method: http.MethodHead, wantStatus: http.StatusOK, checkBody: true, wantRanges: true},
		{
			name:       "not modified",
			method:     http.MethodGet,
			headers:    map[string]string{"If-Modified-Since": modTime.Format(http.TimeFormat)},
			wantStatus: http.StatusNotModified,
		},
		{
			name:       "precondition failed",
			method:     http.MethodGet,
			headers:    map[string]string{"If-Match": `"different"`},
			wantStatus: http.StatusPreconditionFailed,
		},
		{
			name:              "range does not overlap",
			method:            http.MethodGet,
			headers:           map[string]string{"Range": "bytes=99-100"},
			wantStatus:        http.StatusRequestedRangeNotSatisfiable,
			wantContentRange:  "bytes */6",
			checkContentRange: true,
		},
		{
			name:              "malformed range",
			method:            http.MethodGet,
			headers:           map[string]string{"Range": "bytes=wat"},
			wantStatus:        http.StatusRequestedRangeNotSatisfiable,
			checkContentRange: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/file", nil)
			for name, value := range test.headers {
				request.Header.Set(name, value)
			}
			response := httptest.NewRecorder()
			response.Header().Set("ETag", `"sha256-example"`)
			file := &loomhttp.FileResponse{
				Name:    "sample.txt",
				ModTime: modTime,
				Content: strings.NewReader("abcdef"),
			}

			file.ServeHTTP(response, request)

			require.Equal(t, test.wantStatus, response.Code)
			if test.checkBody {
				require.Equal(t, test.wantBody, response.Body.String())
			}
			if test.wantRanges {
				require.Equal(t, "bytes", response.Header().Get("Accept-Ranges"))
			}
			if test.checkContentRange {
				require.Equal(t, test.wantContentRange, response.Header().Get("Content-Range"))
			}
			if test.wantContentType != "" {
				require.Equal(t, test.wantContentType, response.Header().Get("Content-Type"))
			}
			if test.wantTypePrefix != "" {
				require.True(t, strings.HasPrefix(response.Header().Get("Content-Type"), test.wantTypePrefix))
			}
		})
	}
}

func TestFileResponsePreservesHeadersSetBeforeServing(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	response := httptest.NewRecorder()
	response.Header().Set("ETag", `"sha256-example"`)
	response.Header().Set("Content-Disposition", `attachment; filename="sample.txt"`)
	file := &loomhttp.FileResponse{
		Name:    "sample.txt",
		Content: strings.NewReader("abcdef"),
	}

	file.ServeHTTP(response, request)

	require.Equal(t, `"sha256-example"`, response.Header().Get("ETag"))
	require.Equal(t, `attachment; filename="sample.txt"`, response.Header().Get("Content-Disposition"))
}
