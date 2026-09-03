package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestGeneratedHTTPTransportPoliciesComposeAtRuntime(t *testing.T) {
	const modulePath = "example.com/transportpolicyit"

	root := RunHTTPDSL(t, transportPolicyIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	if err := os.WriteFile(filepath.Join(dir, "transport_policy_test.go"), []byte(transportPolicyHarness), 0o600); err != nil {
		t.Fatalf("write transport policy harness: %v", err)
	}

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}

func transportPolicyIntegrationDSL() {
	Service("transport_policy", func() {
		Method("read", func() {
			Result(func() {
				Attribute("message", String)
				Attribute("session", String)
				Attribute("refresh", String)
				Required("message", "session", "refresh")
			})
			HTTP(func() {
				GET("/session")
				Response(StatusOK, func() {
					Body("message")
					Cookie("session")
					CookieHTTPOnly()
					CookieSameSite(CookieSameSiteLax)
					Cookie("refresh")
					CookiePath("/refresh")
				})
			})
		})
		Method("raw", func() {
			Result(func() {
				Attribute("session", String)
				Attribute("refresh", String)
				Required("session", "refresh")
			})
			HTTP(func() {
				GET("/raw")
				SkipResponseBodyEncodeDecode()
				Response(StatusOK, func() {
					Cookie("session")
					Cookie("refresh")
				})
			})
		})
		Method("raw_implicit", func() {
			HTTP(func() {
				GET("/raw-implicit")
				SkipResponseBodyEncodeDecode()
			})
		})
		Method("raw_created", func() {
			HTTP(func() {
				GET("/raw-created")
				SkipResponseBodyEncodeDecode()
				Response(StatusCreated, func() {
					ContentType("application/json")
					OpenAPIBody(Any)
				})
			})
		})
		Method("raw_created_empty", func() {
			HTTP(func() {
				GET("/raw-created-empty")
				SkipResponseBodyEncodeDecode()
				Response(StatusCreated)
			})
		})
		Method("fail", func() {
			Error("denied", func() {
				Attribute("session", String)
				Attribute("refresh", String)
				Required("session", "refresh")
			})
			HTTP(func() {
				GET("/fail")
				Response("denied", StatusUnauthorized, func() {
					Cookie("session")
					Cookie("refresh")
				})
			})
		})
	})
	Service("raw_content_type", func() {
		Method("raw_media", func() {
			HTTP(func() {
				GET("/raw-media")
				SkipResponseBodyEncodeDecode()
				Response(StatusOK, func() {
					ContentType("application/json")
					OpenAPIBody(Any)
				})
			})
		})
		Method("raw_media_options", func() {
			HTTP(func() {
				OPTIONS("/raw-media")
				Response(StatusNoContent)
			})
		})
	})
}

const transportPolicyHarness = `package transportpolicyit_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	transportpolicy "example.com/transportpolicyit/gen/transport_policy"
	transportpolicyserver "example.com/transportpolicyit/gen/http/transport_policy/server"
	rawcontenttype "example.com/transportpolicyit/gen/raw_content_type"
	rawcontenttypeserver "example.com/transportpolicyit/gen/http/raw_content_type/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestPoliciesRunThroughGeneratedHandler(t *testing.T) {
	var calls atomic.Int32
	expires := time.Date(2031, time.February, 3, 4, 5, 6, 0, time.UTC)
	server := newServer(t, &calls, func(_ context.Context, cookie *http.Cookie) error {
		cookie.Domain = "example.test"
		cookie.Secure = true
		cookie.Expires = expires
		return nil
	})

	response := request(t, server, http.MethodGet, "application/json")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("GET status = %d, want 200; body: %s", response.StatusCode, body)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("GET endpoint calls = %d, want 1", got)
	}
	cookies := response.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("GET cookies = %d, want 2", len(cookies))
	}
	for _, cookie := range cookies {
		if cookie.Domain != "example.test" || !cookie.Secure || !cookie.Expires.Equal(expires) {
			t.Errorf("cookie %q runtime attributes = %#v", cookie.Name, cookie)
		}
	}
	if vary := response.Header.Get("Vary"); !strings.Contains(vary, "Accept") {
		t.Errorf("GET Vary = %q, want Accept", vary)
	}

	head := request(t, server, http.MethodHead, "application/json")
	defer head.Body.Close()
	if head.StatusCode != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", head.StatusCode)
	}
	if body, err := io.ReadAll(head.Body); err != nil || len(body) != 0 {
		t.Fatalf("HEAD body = %q, %v; want empty", body, err)
	}
	if head.Header.Get("Content-Length") == "" {
		t.Error("HEAD response omitted generated representation Content-Length")
	}
	if len(head.Cookies()) != 2 {
		t.Errorf("HEAD cookies = %d, want 2", len(head.Cookies()))
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("endpoint calls after HEAD = %d, want 2", got)
	}
}

func TestNegotiationRejectsBeforeGeneratedEndpoint(t *testing.T) {
	var calls atomic.Int32
	server := newServer(t, &calls, func(context.Context, *http.Cookie) error { return nil })

	response := request(t, server, http.MethodGet, "text/html")
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotAcceptable {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 406; body: %s", response.StatusCode, body)
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("endpoint calls = %d, want 0", got)
	}
	if len(response.Cookies()) != 0 {
		t.Errorf("rejection cookies = %d, want 0", len(response.Cookies()))
	}
}

func TestNegotiationCombinesFieldLinesForGeneratedEncoding(t *testing.T) {
	var calls atomic.Int32
	server := newServer(t, &calls, func(context.Context, *http.Cookie) error { return nil })

	response := requestWithAcceptValues(t, server, http.MethodGet, "text/plain", "application/json")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 200; body: %s", response.StatusCode, body)
	}
	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("endpoint calls = %d, want 1", got)
	}
}

func TestLateCookiePolicyFailureDoesNotLeakEarlierCookie(t *testing.T) {
	var calls atomic.Int32
	server := newServer(t, &calls, func(_ context.Context, cookie *http.Cookie) error {
		if cookie.Name == "refresh" {
			return errors.New("invalid refresh deployment policy")
		}
		return nil
	})

	response := request(t, server, http.MethodGet, "application/json")
	defer response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 500; body: %s", response.StatusCode, body)
	}
	if got := response.Header.Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("failed response leaked cookies: %v", got)
	}
	if got := response.Header.Get("Content-Type"); got != loomhttp.ProblemJSONContentType {
		t.Errorf("Content-Type = %q, want %q", got, loomhttp.ProblemJSONContentType)
	}
}

func TestRawResponseCookiePolicyFailureUsesGeneratedProblem(t *testing.T) {
	server := newFailureServer(t, false)

	response := requestPath(t, server, "/raw")
	defer response.Body.Close()
	requireInternalProblemWithoutCookies(t, response)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read raw failure body: %v", err)
	}
	if strings.Contains(string(body), "raw-success-body") {
		t.Fatalf("raw success body leaked into failure: %q", body)
	}
}

func TestRawWriterFailureDoesNotLeakGeneratedResponseCookies(t *testing.T) {
	server := newRawWriteFailureServer(t)

	response := requestPath(t, server, "/raw")
	defer response.Body.Close()
	requireInternalProblemWithoutCookies(t, response)
}

func TestRawSuccessPreservesImplicitHeadersAndAuthoredStatus(t *testing.T) {
	server := newRawSuccessServer(t)

	implicit := requestPath(t, server, "/raw-implicit")
	defer implicit.Body.Close()
	if implicit.StatusCode != http.StatusOK {
		t.Fatalf("implicit raw status = %d, want 200", implicit.StatusCode)
	}
	if got := implicit.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("implicit raw Content-Type = %q, want text/plain; charset=utf-8", got)
	}
	implicitBody, err := io.ReadAll(implicit.Body)
	if err != nil {
		t.Fatalf("read implicit raw body: %v", err)
	}
	if got := string(implicitBody); got != "plain body" {
		t.Errorf("implicit raw body = %q, want plain body", got)
	}

	for _, path := range []string{"/raw-created", "/raw-created-empty"} {
		response := requestPath(t, server, path)
		if response.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			t.Fatalf("GET %s status = %d, want 201; body: %s", path, response.StatusCode, body)
		}
		if path == "/raw-created" {
			if got := response.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("GET %s Content-Type = %q, want application/json", path, got)
			}
		}
		if err := response.Body.Close(); err != nil {
			t.Errorf("close GET %s response: %v", path, err)
		}
	}
}

func TestRawResponseUsesDesignedContentType(t *testing.T) {
	server := newRawSuccessServer(t)

	get := requestMethodPath(t, server, http.MethodGet, "/raw-media")
	defer get.Body.Close()
	if get.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(get.Body)
		t.Fatalf("GET status = %d, want 200; body: %s", get.StatusCode, body)
	}
	if got := get.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("GET Content-Type = %q, want application/json", got)
	}
	body, err := io.ReadAll(get.Body)
	if err != nil {
		t.Fatalf("read GET body: %v", err)
	}
	if got := string(body); got != "{\"message\":\"ok\"}" {
		t.Errorf("GET body = %q, want raw JSON", got)
	}

	head := requestMethodPath(t, server, http.MethodHead, "/raw-media")
	defer head.Body.Close()
	if head.StatusCode != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", head.StatusCode)
	}
	if got := head.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("HEAD Content-Type = %q, want application/json", got)
	}
	if head.Header.Get("Content-Length") == "" {
		t.Error("HEAD response omitted raw representation Content-Length")
	}
	if body, err := io.ReadAll(head.Body); err != nil || len(body) != 0 {
		t.Errorf("HEAD body = %q, %v; want empty", body, err)
	}

	options := requestMethodPath(t, server, http.MethodOptions, "/raw-media")
	defer options.Body.Close()
	if options.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(options.Body)
		t.Fatalf("OPTIONS status = %d, want 204; body: %s", options.StatusCode, body)
	}
}

func TestModeledErrorCookiePolicyFailureUsesGeneratedProblem(t *testing.T) {
	server := newFailureServer(t, true)

	response := requestPath(t, server, "/fail")
	defer response.Body.Close()
	requireInternalProblemWithoutCookies(t, response)
}

func newServer(
	t *testing.T,
	calls *atomic.Int32,
	cookiePolicy loomhttp.ResponseCookiePolicy,
) *httptest.Server {
	t.Helper()
	endpoints := &transportpolicy.Endpoints{
		Read: func(context.Context, any) (any, error) {
			calls.Add(1)
			return &transportpolicy.ReadResult{
				Message: "hello",
				Session: "session-value",
				Refresh: "refresh-value",
			}, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := transportpolicyserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	generated.Read = cookiePolicy.Handler(generated.Read)
	negotiation, err := loomhttp.NewResponseNegotiationPolicy("application/json")
	if err != nil {
		t.Fatalf("create response negotiation policy: %v", err)
	}
	generated.Read = negotiation.Handler(generated.Read)
	transportpolicyserver.Mount(mux, generated)
	loomhttp.MountDerivedHead(mux, "/session", generated.Read)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newFailureServer(t *testing.T, modeledError bool) *httptest.Server {
	t.Helper()
	endpoints := &transportpolicy.Endpoints{
		Raw: func(context.Context, any) (any, error) {
			return &transportpolicy.RawResponseData{
				Result: &transportpolicy.RawResult{Session: "session-value", Refresh: "refresh-value"},
				Body:   io.NopCloser(strings.NewReader("raw-success-body")),
			}, nil
		},
		Fail: func(context.Context, any) (any, error) {
			return nil, &transportpolicy.Denied{Session: "session-value", Refresh: "refresh-value"}
		},
	}
	mux := loomhttp.NewMuxer()
	generated := transportpolicyserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	policy := loomhttp.ResponseCookiePolicy(func(_ context.Context, cookie *http.Cookie) error {
		if cookie.Name == "refresh" {
			return errors.New("invalid refresh deployment policy")
		}
		return nil
	})
	if modeledError {
		generated.Fail = policy.Handler(generated.Fail)
	} else {
		generated.Raw = policy.Handler(generated.Raw)
	}
	transportpolicyserver.Mount(mux, generated)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

type failingRawBody struct {
	err error
}

func (*failingRawBody) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (*failingRawBody) Close() error {
	return nil
}

func (b *failingRawBody) WriteTo(io.Writer) (int64, error) {
	return 0, b.err
}

func newRawWriteFailureServer(t *testing.T) *httptest.Server {
	t.Helper()
	endpoints := &transportpolicy.Endpoints{
		Raw: func(context.Context, any) (any, error) {
			return &transportpolicy.RawResponseData{
				Result: &transportpolicy.RawResult{Session: "session-value", Refresh: "refresh-value"},
				Body:   &failingRawBody{err: errors.New("raw writer failed before commit")},
			}, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := transportpolicyserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	transportpolicyserver.Mount(mux, generated)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func newRawSuccessServer(t *testing.T) *httptest.Server {
	t.Helper()
	endpoints := &transportpolicy.Endpoints{
		RawImplicit: func(context.Context, any) (any, error) {
			return &transportpolicy.RawImplicitResponseData{
				Body: io.NopCloser(strings.NewReader("plain body")),
			}, nil
		},
		RawCreated: func(context.Context, any) (any, error) {
			return &transportpolicy.RawCreatedResponseData{
				Body: io.NopCloser(strings.NewReader("created body")),
			}, nil
		},
		RawCreatedEmpty: func(context.Context, any) (any, error) {
			return &transportpolicy.RawCreatedEmptyResponseData{
				Body: io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}
	rawEndpoints := &rawcontenttype.Endpoints{
		RawMedia: func(context.Context, any) (any, error) {
			return &rawcontenttype.RawMediaResponseData{
				Body: io.NopCloser(strings.NewReader("{\"message\":\"ok\"}")),
			}, nil
		},
		RawMediaOptions: func(context.Context, any) (any, error) {
			return nil, nil
		},
	}
	mux := loomhttp.NewMuxer()
	generated := transportpolicyserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	rawGenerated := rawcontenttypeserver.New(rawEndpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	transportpolicyserver.Mount(mux, generated)
	rawcontenttypeserver.Mount(mux, rawGenerated)
	loomhttp.MountDerivedHead(mux, "/raw-media", rawGenerated.RawMedia)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func request(t *testing.T, server *httptest.Server, method, accept string) *http.Response {
	return requestWithAcceptValues(t, server, method, accept)
}

func requestWithAcceptValues(
	t *testing.T,
	server *httptest.Server,
	method string,
	accept ...string,
) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, server.URL+"/session", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	for _, value := range accept {
		request.Header.Add("Accept", value)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return response
}

func requestPath(t *testing.T, server *httptest.Server, path string) *http.Response {
	t.Helper()
	return requestMethodPath(t, server, http.MethodGet, path)
}

func requestMethodPath(t *testing.T, server *httptest.Server, method, path string) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, server.URL+path, nil)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, path, err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return response
}

func requireInternalProblemWithoutCookies(t *testing.T, response *http.Response) {
	t.Helper()
	if response.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 500; body: %s", response.StatusCode, body)
	}
	if got := response.Header.Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("failed response leaked cookies: %v", got)
	}
	if got := response.Header.Get("Content-Type"); got != loomhttp.ProblemJSONContentType {
		t.Errorf("Content-Type = %q, want %q", got, loomhttp.ProblemJSONContentType)
	}
}
`
