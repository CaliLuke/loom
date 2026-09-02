package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedHTTPServerRejectsOversizedRequestBodies(t *testing.T) {
	const modulePath = "example.com/bodylimitit"

	root := RunHTTPDSL(t, requestBodyLimitDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "request_body_limit_test.go"), []byte(requestBodyLimitHarness), 0o600); err != nil {
		t.Fatalf("write request body limit harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func requestBodyLimitDSL() {
	Service("limits", func() {
		Method("JSON", func() {
			Payload(func() {
				Attribute("value", String)
				Required("value")
			})
			HTTP(func() {
				POST("/json")
			})
		})
		Method("Text", func() {
			Payload(String)
			HTTP(func() {
				POST("/text")
			})
		})
		Method("Form", func() {
			Payload(func() {
				Attribute("value", String)
				Required("value")
			})
			HTTP(func() {
				POST("/form")
				FormRequest()
			})
		})
		Method("Multipart", func() {
			Payload(func() {
				Attribute("data", Bytes)
				Required("data")
			})
			HTTP(func() {
				POST("/multipart")
				MultipartRequest()
			})
		})
		Method("Raw", func() {
			HTTP(func() {
				POST("/raw")
				SkipRequestBodyEncodeDecode()
			})
		})
	})
}

const requestBodyLimitHarness = `package bodylimitit_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	limits "example.com/bodylimitit/gen/limits"
	limitsserver "example.com/bodylimitit/gen/http/limits/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type dispatchCounts struct {
	json      atomic.Int32
	text      atomic.Int32
	form      atomic.Int32
	multipart atomic.Int32
	raw       atomic.Int32
}

type countingDecoder struct {
	delegate loomhttp.Decoder
	calls    *atomic.Int32
}

func (d *countingDecoder) Decode(value any) error {
	d.calls.Add(1)
	return d.delegate.Decode(value)
}

func TestGeneratedRoutesEnforceRequestBodyLimit(t *testing.T) {
	counts := new(dispatchCounts)
	server := newBodyLimitServer(t, counts, false)

	exactJSON := exactLimitJSON(t)
	response := sendRequest(t, server, "/json", "application/json", bytes.NewReader(exactJSON), false)
	requireSuccess(t, response)
	if got := counts.json.Load(); got != 1 {
		t.Fatalf("exact-limit JSON dispatch count = %d, want 1", got)
	}

	oversizedJSON := append(append([]byte(nil), exactJSON...), ' ')
	tests := []struct {
		name        string
		path        string
		contentType string
		body        func(*testing.T) io.Reader
		chunked     bool
	}{
		{
			name:        "JSON with Content-Length",
			path:        "/json",
			contentType: "application/json",
			body:        func(*testing.T) io.Reader { return bytes.NewReader(oversizedJSON) },
		},
		{
			name:        "chunked JSON",
			path:        "/json",
			contentType: "application/json",
			body:        func(*testing.T) io.Reader { return bytes.NewReader(oversizedJSON) },
			chunked:     true,
		},
		{
			name:        "text",
			path:        "/text",
			contentType: "text/plain",
			body: func(*testing.T) io.Reader {
				return strings.NewReader(strings.Repeat("x", loomhttp.DefaultMaxRequestBodyBytes+1))
			},
		},
		{
			name:        "multipart",
			path:        "/multipart",
			contentType: "multipart/form-data",
			body:        oversizedMultipartBody,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bodyReader := test.body(t)
			contentType := test.contentType
			if multipartBody, ok := bodyReader.(*multipartRequestBody); ok {
				contentType = multipartBody.contentType
				bodyReader = multipartBody.Reader
			}
			response := sendRequest(t, server, test.path, contentType, bodyReader, test.chunked)
			requireRequestTooLarge(t, response)
		})
	}

	if got := counts.json.Load(); got != 1 {
		t.Errorf("JSON dispatch count after oversized requests = %d, want 1", got)
	}
	if got := counts.text.Load(); got != 0 {
		t.Errorf("text dispatch count = %d, want 0", got)
	}
	if got := counts.multipart.Load(); got != 0 {
		t.Errorf("multipart dispatch count = %d, want 0", got)
	}
}

func TestGeneratedRoutesHonorConfiguredRequestBodyLimits(t *testing.T) {
	counts := new(dispatchCounts)
	server := newBodyLimitServer(t, counts, true)

	tests := []struct {
		name        string
		path        string
		contentType string
		body        io.Reader
		wantSuccess bool
	}{
		{name: "exact JSON", path: "/json", contentType: "application/json", body: strings.NewReader(` + "`" + `{"value":"ok"}` + "`" + `), wantSuccess: true},
		{name: "oversized JSON", path: "/json", contentType: "application/json", body: strings.NewReader(` + "`" + `{"value":"ok"} ` + "`" + `)},
		{name: "exact form", path: "/form", contentType: "application/x-www-form-urlencoded", body: strings.NewReader("value=ok"), wantSuccess: true},
		{name: "oversized form", path: "/form", contentType: "application/x-www-form-urlencoded", body: strings.NewReader("value=too-long")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := sendRequest(t, server, test.path, test.contentType, test.body, false)
			if test.wantSuccess {
				requireSuccess(t, response)
				return
			}
			requireRequestTooLarge(t, response)
		})
	}

	multipartBody := sizedMultipartBody(t, 2048)
	response := sendRequest(t, server, "/multipart", multipartBody.contentType, multipartBody, true)
	requireRequestTooLarge(t, response)

	response = sendRequest(t, server, "/raw", "application/octet-stream", strings.NewReader("raw"), false)
	requireSuccess(t, response)
	response = sendRequest(t, server, "/raw", "application/octet-stream", strings.NewReader("four"), true)
	requireRequestTooLarge(t, response)

	if got := counts.json.Load(); got != 1 {
		t.Errorf("configured JSON dispatch count = %d, want 1", got)
	}
	if got := counts.form.Load(); got != 1 {
		t.Errorf("configured form dispatch count = %d, want 1", got)
	}
	if got := counts.multipart.Load(); got != 0 {
		t.Errorf("configured multipart dispatch count = %d, want 0", got)
	}
	if got := counts.raw.Load(); got != 2 {
		t.Errorf("configured raw dispatch count = %d, want 2", got)
	}
}

func TestGeneratedFormRouteHonorsLimitAboveNetHTTPDefault(t *testing.T) {
	const maxBytes = 11 << 20

	counts := new(dispatchCounts)
	policy, err := loomhttp.NewRequestBodyPolicy(maxBytes)
	if err != nil {
		t.Fatalf("create large form policy: %v", err)
	}
	server := newFormLimitServer(t, counts, policy)

	exact := "value=" + strings.Repeat("x", maxBytes-len("value="))
	response := sendRequest(t, server, "/form", "application/x-www-form-urlencoded", strings.NewReader(exact), false)
	requireSuccess(t, response)

	oversized := exact + "x"
	response = sendRequest(t, server, "/form", "application/x-www-form-urlencoded", strings.NewReader(oversized), false)
	requireRequestTooLarge(t, response)

	if got := counts.form.Load(); got != 1 {
		t.Errorf("large configured form dispatch count = %d, want 1", got)
	}
}

func TestGeneratedCustomDecoderHonorsConfiguredRequestBodyLimit(t *testing.T) {
	counts := new(dispatchCounts)
	var decoderCalls atomic.Int32
	server := newCustomDecoderLimitServer(t, counts, &decoderCalls)

	response := sendRequest(t, server, "/json", "application/json", strings.NewReader(` + "`" + `{"value":"ok"}` + "`" + `), false)
	requireSuccess(t, response)
	response = sendRequest(t, server, "/json", "application/json", strings.NewReader(` + "`" + `{"value":"ok"} ` + "`" + `), true)
	requireRequestTooLarge(t, response)

	if got := decoderCalls.Load(); got != 2 {
		t.Errorf("custom decoder calls = %d, want 2", got)
	}
	if got := counts.json.Load(); got != 1 {
		t.Errorf("custom decoder endpoint calls = %d, want 1", got)
	}
}

func TestGeneratedMultipartRouteHonorsExactConfiguredWireLimit(t *testing.T) {
	exactBody := sizedMultipartBody(t, 8)
	policy, err := loomhttp.NewRequestBodyPolicy(exactBody.size)
	if err != nil {
		t.Fatalf("create exact multipart policy: %v", err)
	}
	counts := new(dispatchCounts)
	server := newMultipartLimitServer(t, counts, policy)

	response := sendRequest(t, server, "/multipart", exactBody.contentType, exactBody, false)
	requireSuccess(t, response)

	oversizedBody := sizedMultipartBody(t, 9)
	if oversizedBody.size != exactBody.size+1 {
		t.Fatalf("multipart wire sizes = %d and %d, want one-byte difference", exactBody.size, oversizedBody.size)
	}
	response = sendRequest(t, server, "/multipart", oversizedBody.contentType, oversizedBody, true)
	requireRequestTooLarge(t, response)

	if got := counts.multipart.Load(); got != 1 {
		t.Errorf("exact multipart dispatch count = %d, want 1", got)
	}
}

func newBodyLimitServer(t *testing.T, counts *dispatchCounts, configured bool) *httptest.Server {
	t.Helper()
	endpoints := &limits.Endpoints{
		JSON: func(context.Context, any) (any, error) {
			counts.json.Add(1)
			return nil, nil
		},
		Text: func(context.Context, any) (any, error) {
			counts.text.Add(1)
			return nil, nil
		},
		Form: func(context.Context, any) (any, error) {
			counts.form.Add(1)
			return nil, nil
		},
		Multipart: func(context.Context, any) (any, error) {
			counts.multipart.Add(1)
			return nil, nil
		},
		Raw: func(_ context.Context, value any) (any, error) {
			counts.raw.Add(1)
			request, ok := value.(*limits.RawRequestData)
			if !ok {
				t.Fatalf("raw request type = %T, want *limits.RawRequestData", value)
			}
			_, err := io.ReadAll(request.Body)
			return nil, err
		},
	}
	mux := loomhttp.NewMuxer()
	generated := limitsserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	if configured {
		jsonPolicy, err := loomhttp.NewRequestBodyPolicy(14)
		if err != nil {
			t.Fatalf("create JSON request body policy: %v", err)
		}
		formPolicy, err := loomhttp.NewRequestBodyPolicy(8)
		if err != nil {
			t.Fatalf("create form request body policy: %v", err)
		}
		generated.JSON = jsonPolicy.Handler(generated.JSON)
		generated.Form = formPolicy.Handler(generated.Form)
		multipartPolicy, err := loomhttp.NewRequestBodyPolicy(1024)
		if err != nil {
			t.Fatalf("create multipart request body policy: %v", err)
		}
		generated.Multipart = multipartPolicy.Handler(generated.Multipart)
		rawPolicy, err := loomhttp.NewRequestBodyPolicy(3)
		if err != nil {
			t.Fatalf("create raw request body policy: %v", err)
		}
		generated.Raw = rawPolicy.Handler(generated.Raw)
	}
	limitsserver.Mount(mux, generated)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newFormLimitServer(
	t *testing.T,
	counts *dispatchCounts,
	policy loomhttp.RequestBodyPolicy,
) *httptest.Server {
	t.Helper()
	endpoints := &limits.Endpoints{
		Form: func(context.Context, any) (any, error) {
			counts.form.Add(1)
			return nil, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := limitsserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	generated.Form = policy.Handler(generated.Form)
	limitsserver.Mount(mux, generated)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newCustomDecoderLimitServer(
	t *testing.T,
	counts *dispatchCounts,
	decoderCalls *atomic.Int32,
) *httptest.Server {
	t.Helper()
	endpoints := &limits.Endpoints{
		JSON: func(context.Context, any) (any, error) {
			counts.json.Add(1)
			return nil, nil
		},
	}
	mux := loomhttp.NewMuxer()
	decoder := func(request *http.Request) loomhttp.Decoder {
		return &countingDecoder{delegate: loomhttp.RequestDecoder(request), calls: decoderCalls}
	}
	generated := limitsserver.New(endpoints, mux, decoder, loomhttp.ResponseEncoder, nil, nil)
	policy, err := loomhttp.NewRequestBodyPolicy(14)
	if err != nil {
		t.Fatalf("create custom decoder request body policy: %v", err)
	}
	generated.JSON = policy.Handler(generated.JSON)
	limitsserver.Mount(mux, generated)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newMultipartLimitServer(
	t *testing.T,
	counts *dispatchCounts,
	policy loomhttp.RequestBodyPolicy,
) *httptest.Server {
	t.Helper()
	endpoints := &limits.Endpoints{
		Multipart: func(context.Context, any) (any, error) {
			counts.multipart.Add(1)
			return nil, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := limitsserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	generated.Multipart = policy.Handler(generated.Multipart)
	limitsserver.Mount(mux, generated)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func exactLimitJSON(t *testing.T) []byte {
	t.Helper()
	body := []byte(` + "`" + `{"value":"ok"}` + "`" + `)
	if len(body) > loomhttp.DefaultMaxRequestBodyBytes {
		t.Fatalf("JSON prefix length %d exceeds request limit", len(body))
	}
	return append(body, bytes.Repeat([]byte{' '}, loomhttp.DefaultMaxRequestBodyBytes-len(body))...)
}

type multipartRequestBody struct {
	io.Reader
	contentType string
	size        int64
}

func oversizedMultipartBody(t *testing.T) io.Reader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("data", "payload.bin")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := io.CopyN(part, zeroReader{}, loomhttp.DefaultMaxRequestBodyBytes+1); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &multipartRequestBody{
		Reader:      bytes.NewReader(body.Bytes()),
		contentType: writer.FormDataContentType(),
		size:        int64(body.Len()),
	}
}

func sizedMultipartBody(t *testing.T, size int64) *multipartRequestBody {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("data", "payload.bin")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := io.CopyN(part, zeroReader{}, size); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &multipartRequestBody{
		Reader:      bytes.NewReader(body.Bytes()),
		contentType: writer.FormDataContentType(),
		size:        int64(body.Len()),
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func sendRequest(
	t *testing.T,
	server *httptest.Server,
	path string,
	contentType string,
	body io.Reader,
	chunked bool,
) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+path, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", contentType)
	if chunked {
		request.ContentLength = -1
		request.TransferEncoding = []string{"chunked"}
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	})
	return response
}

func requireSuccess(t *testing.T, response *http.Response) {
	t.Helper()
	if response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 204; body: %s", response.StatusCode, body)
	}
}

func requireRequestTooLarge(t *testing.T, response *http.Response) {
	t.Helper()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 413; body: %s", response.StatusCode, body)
	}
	if got := response.Header.Get("Content-Type"); got != loomhttp.ProblemJSONContentType {
		t.Errorf("Content-Type = %q, want %q", got, loomhttp.ProblemJSONContentType)
	}
	var problem loomhttp.ProblemResponse
	if err := json.UnmarshalRead(response.Body, &problem); err != nil {
		t.Fatalf("decode problem response: %v", err)
	}
	if problem.Type != "about:blank" {
		t.Errorf("problem type = %q, want about:blank", problem.Type)
	}
	if problem.Title != http.StatusText(http.StatusRequestEntityTooLarge) {
		t.Errorf("problem title = %q, want %q", problem.Title, http.StatusText(http.StatusRequestEntityTooLarge))
	}
	if problem.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("problem status = %d, want 413", problem.Status)
	}
	if problem.Detail != "request body too large" {
		t.Errorf("problem detail = %q, want request body too large", problem.Detail)
	}
	if problem.Code != loom.RequestBodyTooLarge {
		t.Errorf("problem code = %q, want %q", problem.Code, loom.RequestBodyTooLarge)
	}
	if problem.Instance != "" && !strings.HasPrefix(problem.Instance, "urn:loom:error:") {
		t.Errorf("problem instance = %q, want empty or Loom error URN", problem.Instance)
	}
}
`
