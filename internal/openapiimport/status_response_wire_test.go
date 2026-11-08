package openapiimport

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const rawSuccessMediaErrorSpec = `openapi: 3.2.0
info: {title: Body roles, version: '1'}
paths:
  /value:
    post:
      operationId: value
      responses:
        '200':
          description: raw value
          content:
            application/octet-stream:
              schema: {type: string, format: binary}
        '400':
          description: failure
          content:
            application/json:
              schema: {type: string}
            application/octet-stream:
              schema: {type: string, format: binary}
`

// This separate authored streaming control exercises the same encoded error
// owner before streaming begins. It does not claim SSE importer support.
const mediaErrorStreamControl = `
var _ = Service("streamcontrol", func() {
 Method("watch", func() {
  StreamingResult(String)
  Error("failed", func() {
   OneOf("body", func() {
    Attribute("json", String)
    Attribute("binary", Bytes)
   })
   Required("body")
  })
  HTTP(func() {
   GET("/watch")
   ServerSentEvents()
   Response(StatusOK)
   Response("failed", StatusBadRequest, func() {
    Body("body")
    MediaType("application/json", "json")
    MediaType("application/octet-stream", "binary")
   })
  })
 })
})
`

func TestImportRawSuccessMediaErrorWire(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(rawSuccessMediaErrorSpec))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{})
	require.NoError(t, err)
	design = append(design, []byte(mediaErrorStreamControl)...)
	directory := requireRenderedDesignGenerates(t, design)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "body_role_wire_test.go"), []byte(rawSuccessMediaErrorHarness), 0o600))
	command := exec.Command("go", "test", "-race", "-v", "-count=1", ".")
	command.Dir = directory
	output, err := command.CombinedOutput()
	t.Log(string(output))
	require.NoError(t, err, string(output))
}

const rawSuccessMediaErrorHarness = `package imported_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	raw "example.com/imported/gen/body_roles"
	rawserver "example.com/imported/gen/http/body_roles/server"
	streamserver "example.com/imported/gen/http/streamcontrol/server"
	stream "example.com/imported/gen/streamcontrol"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestRawSuccessAndIndependentErrors(t *testing.T) {
	for _, tc := range []struct {
		name, path, method, media, wire string
		status                          int
		failure                         error
	}{
		{"raw success", "/value", http.MethodPost, "application/octet-stream", "raw-result", 200, nil},
		{"raw JSON error", "/value", http.MethodPost, "application/json", "\"json-error\"", 400, &raw.Status400{Body: raw.NewValueResponseMediaOrValueResponseMedia2ValueResponseMedia("json-error")}},
		{"raw binary error", "/value", http.MethodPost, "application/octet-stream", "binary-error", 400, &raw.Status400{Body: raw.NewValueResponseMediaOrValueResponseMedia2ValueResponseMedia2([]byte("binary-error"))}},
		{"pre-stream JSON error", "/watch", http.MethodGet, "application/json", "\"json-error\"", 400, &stream.Failed{Body: stream.NewBodyJSON("json-error")}},
		{"pre-stream binary error", "/watch", http.MethodGet, "application/octet-stream", "binary-error", 400, &stream.Failed{Body: stream.NewBodyBinary([]byte("binary-error"))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			mux := loomhttp.NewMuxer()
			endpoint := func(context.Context, any) (any, error) {
				calls.Add(1)
				if tc.failure != nil {
					return nil, tc.failure
				}
				return &raw.ValueResponseData{Body: io.NopCloser(strings.NewReader("raw-result"))}, nil
			}
			rawserver.Mount(mux, rawserver.New(&raw.Endpoints{Value: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			streamserver.Mount(mux, streamserver.New(&stream.Endpoints{Watch: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			host := httptest.NewServer(mux)
			defer host.Close()
			request, err := http.NewRequest(tc.method, host.URL+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Accept", tc.media)
			response, err := host.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			wire, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil {
				t.Fatalf("read: %v, close: %v", readErr, closeErr)
			}
			if response.StatusCode != tc.status || response.Header.Get("Content-Type") != tc.media || strings.TrimSpace(string(wire)) != tc.wire {
				t.Errorf("status/media/body: %d/%s/%q, want %d/%s/%q", response.StatusCode, response.Header.Get("Content-Type"), wire, tc.status, tc.media, tc.wire)
			}
			if calls.Load() != 1 {
				t.Errorf("service calls: %d, want 1", calls.Load())
			}
		})
	}
}
`
