package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

// TestMultipartRequiredFileMissingFieldError characterizes the wire behavior
// of the generated multipart request decoder when a required file field is
// missing from the request. The decoder accumulates a MissingFieldError for
// the absent file part and then unconditionally overwrites that error with
// the result of the body-level Validate call; the test asserts that the
// caller-visible outcome (a "missing_field" problem naming the file
// attribute) is unaffected by that internal redundancy, so the pair can be
// simplified without changing behavior.
func TestMultipartRequiredFileMissingFieldError(t *testing.T) {
	const modulePath = "example.com/multipartrequiredfileit"

	root := RunHTTPDSL(t, multipartRequiredFileIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "integration_test.go"), []byte(multipartRequiredFileIntegrationHarness), 0o644); err != nil {
		t.Fatalf("write integration harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func multipartRequiredFileIntegrationDSL() {
	Service("uploads", func() {
		Method("upload", func() {
			Payload(func() {
				Attribute("file", Bytes)
				Attribute("label", String)
				Required("file")
			})
			Result(func() {
				Attribute("label", String)
				Required("label")
			})
			HTTP(func() {
				POST("/")
				MultipartRequest()
			})
		})
	})
}

const multipartRequiredFileIntegrationHarness = `package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	uploads "example.com/multipartrequiredfileit/gen/uploads"
	uploadsserver "example.com/multipartrequiredfileit/gen/http/uploads/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type uploadService struct{}

func (s *uploadService) Upload(_ context.Context, payload *uploads.UploadPayload) (*uploads.UploadResult, error) {
	label := ""
	if payload != nil && payload.Label != nil {
		label = *payload.Label
	}
	return &uploads.UploadResult{Label: label}, nil
}

func newUploadServer(t *testing.T) *httptest.Server {
	t.Helper()
	service := &uploadService{}
	endpoints := uploads.NewEndpoints(service)
	mux := loomhttp.NewMuxer()
	transport := uploadsserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	uploadsserver.Mount(mux, transport)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestGeneratedMultipartRequiredFileMissingProducesMissingFieldError(t *testing.T) {
	server := newUploadServer(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("label", "example"); err != nil {
		t.Fatalf("write label field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, http.StatusBadRequest, body)
	}

	var problem struct {
		Code   string ` + "`json:\"code\"`" + `
		Detail string ` + "`json:\"detail\"`" + `
	}
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("unmarshal problem: %v; body=%s", err, body)
	}
	if problem.Code != "missing_field" {
		t.Errorf("code = %q, want %q; body=%s", problem.Code, "missing_field", body)
	}
	if !strings.Contains(problem.Detail, "file") {
		t.Errorf("detail = %q, want mention of %q", problem.Detail, "file")
	}
}

func TestGeneratedMultipartRequiredFilePresentSucceeds(t *testing.T) {
	server := newUploadServer(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("label", "example"); err != nil {
		t.Fatalf("write label field: %v", err)
	}
	part, err := writer.CreateFormFile("file", "sample.bin")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("payload")); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/", &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, http.StatusOK, body)
	}
}
`
