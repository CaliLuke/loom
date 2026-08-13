package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestFileResponseGeneratedServerClientIntegration(t *testing.T) {
	const modulePath = "example.com/fileresponseit"

	root := RunHTTPDSL(t, fileResponseIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	err := os.WriteFile(filepath.Join(dir, "integration_test.go"), []byte(fileResponseIntegrationHarness), 0o644)
	if err != nil {
		t.Fatalf("write integration harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func fileResponseIntegrationDSL() {
	Service("files", func() {
		Method("download", func() {
			Payload(func() {
				Attribute("mode", String)
			})
			Result(func() {
				Attribute("etag", String)
				Attribute("disposition", String)
				Required("etag", "disposition")
			})
			Error("not_found")
			HTTP(func() {
				GET("/download")
				HEAD("/download")
				Param("mode")
				FileResponse()
				Response(func() {
					ContentType("application/octet-stream")
					Header("etag:ETag")
					Header("disposition:Content-Disposition")
				})
				Response("not_found", StatusNotFound)
			})
		})
	})
}

const fileResponseIntegrationHarness = `package integration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	files "example.com/fileresponseit/gen/files"
	filesclient "example.com/fileresponseit/gen/http/files/client"
	filesserver "example.com/fileresponseit/gen/http/files/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

const (
	fileBody = "abcdef"
	fileETag = "\"sha256-example\""
)

var fileModTime = time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

type fileService struct {
	mu   sync.Mutex
	last *trackingSeeker
}

func (s *fileService) Download(_ context.Context, payload *files.DownloadPayload) (*files.DownloadResult, *loomhttp.FileResponse, error) {
	mode := ""
	if payload != nil && payload.Mode != nil {
		mode = *payload.Mode
	}
	if mode == "error" {
		return nil, nil, files.MakeNotFound(errors.New("missing file"))
	}

	result := &files.DownloadResult{
		Etag:        fileETag,
		Disposition: ` + "`attachment; filename=\"sample.bin\"`" + `,
	}
	if mode == "nil" {
		return result, nil, nil
	}

	content := &trackingSeeker{Reader: strings.NewReader(fileBody)}
	s.mu.Lock()
	s.last = content
	s.mu.Unlock()
	return result, &loomhttp.FileResponse{
		Name:    "sample.bin",
		ModTime: fileModTime,
		Content: content,
	}, nil
}

func (s *fileService) closeCount() int32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		return 0
	}
	return s.last.closed.Load()
}

type trackingSeeker struct {
	*strings.Reader
	closed atomic.Int32
}

func (s *trackingSeeker) Close() error {
	s.closed.Add(1)
	return nil
}

type closeTrackingBody struct {
	io.ReadCloser
	closed atomic.Bool
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)
	return b.ReadCloser.Close()
}

type observingDoer struct {
	base    loomhttp.Doer
	headers http.Header
	body    *closeTrackingBody
}

func (d *observingDoer) Do(request *http.Request) (*http.Response, error) {
	for name, values := range d.headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	response, err := d.base.Do(request)
	if err != nil {
		return nil, err
	}
	d.body = &closeTrackingBody{ReadCloser: response.Body}
	response.Body = d.body
	return response, nil
}

func TestGeneratedFileResponseServerProtocol(t *testing.T) {
	service, server := newFileServer(t)
	tests := []struct {
		name        string
		method      string
		headers     http.Header
		wantStatus  int
		wantBody    string
		wantRange   string
		wantType    string
		wantETag    string
	}{
		{name: "GET", method: http.MethodGet, wantStatus: http.StatusOK, wantBody: fileBody, wantType: "application/octet-stream", wantETag: fileETag},
		{name: "HEAD", method: http.MethodHead, wantStatus: http.StatusOK, wantType: "application/octet-stream", wantETag: fileETag},
		{name: "range", method: http.MethodGet, headers: http.Header{"Range": {"bytes=1-3"}}, wantStatus: http.StatusPartialContent, wantBody: "bcd", wantRange: "bytes 1-3/6", wantType: "application/octet-stream", wantETag: fileETag},
		{name: "not modified", method: http.MethodGet, headers: http.Header{"If-None-Match": {fileETag}}, wantStatus: http.StatusNotModified, wantETag: fileETag},
		{name: "precondition failed", method: http.MethodGet, headers: http.Header{"If-Match": {"\"different\""}}, wantStatus: http.StatusPreconditionFailed, wantType: "application/octet-stream", wantETag: fileETag},
		{name: "range not satisfiable", method: http.MethodGet, headers: http.Header{"Range": {"bytes=99-100"}}, wantStatus: http.StatusRequestedRangeNotSatisfiable, wantRange: "bytes */6", wantBody: "invalid range: failed to overlap\n", wantType: "text/plain; charset=utf-8"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(test.method, server.URL+"/download", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			request.Header = test.headers.Clone()
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("consume response: read=%v close=%v", readErr, closeErr)
			}
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", response.StatusCode, test.wantStatus, body)
			}
			if string(body) != test.wantBody {
				t.Errorf("body = %q, want %q", body, test.wantBody)
			}
			if got := response.Header.Get("Content-Range"); got != test.wantRange {
				t.Errorf("Content-Range = %q, want %q", got, test.wantRange)
			}
			if got := response.Header.Get("Content-Type"); got != test.wantType {
				t.Errorf("Content-Type = %q, want %q", got, test.wantType)
			}
			if got := response.Header.Get("ETag"); got != test.wantETag {
				t.Errorf("ETag = %q, want %q", got, test.wantETag)
			}
			if test.wantStatus != http.StatusRequestedRangeNotSatisfiable {
				if got := response.Header.Get("Content-Disposition"); got != ` + "`attachment; filename=\"sample.bin\"`" + ` {
					t.Errorf("Content-Disposition = %q", got)
				}
			}
			if got := service.closeCount(); got != 1 {
				t.Errorf("content Close calls = %d, want 1", got)
			}
		})
	}
}

func TestGeneratedFileResponseErrorsCommitBeforeFileMetadata(t *testing.T) {
	_, server := newFileServer(t)
	for _, test := range []struct {
		name       string
		mode       string
		wantStatus int
	}{
		{name: "service error", mode: "error", wantStatus: http.StatusNotFound},
		{name: "nil content", mode: "nil", wantStatus: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			response, err := server.Client().Get(server.URL + "/download?mode=" + test.mode)
			if err != nil {
				t.Fatalf("GET: %v", err)
			}
			_, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("consume response: read=%v close=%v", readErr, closeErr)
			}
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
			for _, name := range []string{"ETag", "Content-Disposition", "Accept-Ranges", "Content-Range"} {
				if got := response.Header.Get(name); got != "" {
					t.Errorf("%s committed before error: %q", name, got)
				}
			}
		})
	}
}

func TestGeneratedFileResponseClientOwnershipAndStatuses(t *testing.T) {
	_, server := newFileServer(t)
	tests := []struct {
		name     string
		headers  http.Header
		wantBody string
	}{
		{name: "OK", wantBody: fileBody},
		{name: "partial content", headers: http.Header{"Range": {"bytes=1-3"}}, wantBody: "bcd"},
		{name: "not modified", headers: http.Header{"If-None-Match": {fileETag}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doer := &observingDoer{base: server.Client(), headers: test.headers}
			client := newGeneratedFileClient(t, server.URL, doer)
			value, err := client(context.Background(), &files.DownloadPayload{})
			if err != nil {
				t.Fatalf("Download: %v", err)
			}
			response := value.(*files.DownloadResponseData)
			result, body := response.Result, response.Body
			if result.Etag != fileETag || result.Disposition == "" {
				t.Errorf("metadata = %#v", result)
			}
			if doer.body.closed.Load() {
				t.Fatal("successful response body closed before caller ownership")
			}
			got, readErr := io.ReadAll(body)
			closeErr := body.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("consume body: read=%v close=%v", readErr, closeErr)
			}
			if string(got) != test.wantBody {
				t.Errorf("body = %q, want %q", got, test.wantBody)
			}
			if !doer.body.closed.Load() {
				t.Fatal("caller Close did not close HTTP response body")
			}
		})
	}

	t.Run("error body closed by generated client", func(t *testing.T) {
		doer := &observingDoer{base: server.Client()}
		client := newGeneratedFileClient(t, server.URL, doer)
		mode := "nil"
		value, err := client(context.Background(), &files.DownloadPayload{Mode: &mode})
		if err == nil {
			t.Fatal("Download succeeded, want error")
		}
		if value != nil {
			t.Fatalf("value = %T, want nil", value)
		}
		if !doer.body.closed.Load() {
			t.Fatal("generated client did not close error response body")
		}
	})
}

func newFileServer(t *testing.T) (*fileService, *httptest.Server) {
	t.Helper()
	service := &fileService{}
	endpoints := files.NewEndpoints(service)
	mux := loomhttp.NewMuxer()
	transport := filesserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	filesserver.Mount(mux, transport)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return service, server
}

func newGeneratedFileClient(t *testing.T, rawURL string, doer loomhttp.Doer) loom.Endpoint {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	transport := filesclient.NewClient(u.Scheme, u.Host, doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return transport.Download()
}
`
