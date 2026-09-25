package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestOptionalObjectRequestBodyGeneratedIntegration generates services whose
// request body is an object payload attribute selected with Body: optional
// JSON bodies with and without required fields, the documented
// OptionalRequestBody example, an optional form body, and a required JSON
// body. It compiles and vets them in a temporary module and round-trips nil
// and non-nil objects through the generated client and server, checking that
// a nil object is sent as no body at all. It also sends empty, whitespace,
// null, empty-object, valid, invalid, malformed and truncated bodies to the
// generated servers and checks the status, the problem code and detail,
// whether the service is invoked and the decoded object.
func TestOptionalObjectRequestBodyGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/optobjectbody"

	root := RunHTTPDSL(t, optionalObjectRequestBodyIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_object_body_test.go"), []byte(optionalObjectRequestBodyHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func optionalObjectRequestBodyIntegrationDSL() {
	var Filters = Type("Filters", func() {
		Attribute("name", String)
		Attribute("limit", Int)
	})
	var Strict = Type("Strict", func() {
		Attribute("name", String)
		Attribute("limit", Int)
		Required("name")
	})
	var SearchFilters = Type("SearchFilters", func() {
		Attribute("tag", String)
	})
	Service("jsonfind", func() {
		Method("find", func() {
			Payload(func() {
				Attribute("q", String)
				Attribute("o", Filters)
			})
			HTTP(func() {
				POST("/json")
				Param("q")
				Body("o")
			})
		})
	})
	Service("strictfind", func() {
		Method("find", func() {
			Payload(func() {
				Attribute("q", String)
				Attribute("o", Strict)
			})
			HTTP(func() {
				POST("/strict")
				Param("q")
				Body("o")
			})
		})
	})
	Service("docfind", func() {
		Method("search", func() {
			Payload(func() {
				Attribute("query", String)
				Attribute("filters", SearchFilters)
			})
			HTTP(func() {
				POST("/search")
				Param("query")
				Body("filters")
				OptionalRequestBody()
			})
		})
	})
	Service("formfind", func() {
		Method("find", func() {
			Payload(func() {
				Attribute("q", String)
				Attribute("o", Filters)
			})
			HTTP(func() {
				POST("/form")
				Param("q")
				Body("o")
				FormRequest()
			})
		})
	})
	Service("reqfind", func() {
		Method("find", func() {
			Payload(func() {
				Attribute("q", String)
				Attribute("o", Strict)
				Required("o")
			})
			HTTP(func() {
				POST("/required")
				Param("q")
				Body("o")
			})
		})
	})
}

const optionalObjectRequestBodyHarness = `package optobjectbody

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	json "encoding/json/v2"

	docclient "example.com/optobjectbody/gen/http/docfind/client"
	docserver "example.com/optobjectbody/gen/http/docfind/server"
	formclient "example.com/optobjectbody/gen/http/formfind/client"
	formserver "example.com/optobjectbody/gen/http/formfind/server"
	jsonclient "example.com/optobjectbody/gen/http/jsonfind/client"
	jsonserver "example.com/optobjectbody/gen/http/jsonfind/server"
	reqclient "example.com/optobjectbody/gen/http/reqfind/client"
	reqserver "example.com/optobjectbody/gen/http/reqfind/server"
	strictclient "example.com/optobjectbody/gen/http/strictfind/client"
	strictserver "example.com/optobjectbody/gen/http/strictfind/server"
	docfind "example.com/optobjectbody/gen/docfind"
	formfind "example.com/optobjectbody/gen/formfind"
	jsonfind "example.com/optobjectbody/gen/jsonfind"
	reqfind "example.com/optobjectbody/gen/reqfind"
	strictfind "example.com/optobjectbody/gen/strictfind"
	loomhttp "github.com/CaliLuke/loom/http"
)

type recorder[T any] struct {
	mu   sync.Mutex
	seen []*T
}

func (r *recorder[T]) record(v *T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen = append(r.seen, v)
}

func (r *recorder[T]) take() []*T {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := r.seen
	r.seen = nil
	return seen
}

type jsonService struct{ recorder[jsonfind.FindPayload] }

func (s *jsonService) Find(_ context.Context, p *jsonfind.FindPayload) error {
	s.record(p)
	return nil
}

type strictService struct{ recorder[strictfind.FindPayload] }

func (s *strictService) Find(_ context.Context, p *strictfind.FindPayload) error {
	s.record(p)
	return nil
}

type docService struct{ recorder[docfind.SearchPayload] }

func (s *docService) Search(_ context.Context, p *docfind.SearchPayload) error {
	s.record(p)
	return nil
}

type formService struct{ recorder[formfind.FindPayload] }

func (s *formService) Find(_ context.Context, p *formfind.FindPayload) error {
	s.record(p)
	return nil
}

type reqService struct{ recorder[reqfind.FindPayload] }

func (s *reqService) Find(_ context.Context, p *reqfind.FindPayload) error {
	s.record(p)
	return nil
}

func ptr[T any](v T) *T { return &v }

// roundTrip sends every payload through the generated client, checks that the
// server received it unchanged and that a nil object is sent as no body, not
// as JSON null, while a non-nil object is sent with the content type want.
func roundTrip[P any](t *testing.T, name string, call func(*P) error, r *recorder[P], wire *wireRecorder, payloads []P, isNil func(*P) bool, contentType string) {
	t.Helper()
	for i := range payloads {
		p := &payloads[i]
		err := func() (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					err = fmt.Errorf("client panicked: %v", rec)
				}
			}()
			return call(p)
		}()
		if err != nil {
			t.Errorf("%s %+v: %v", name, *p, err)
			continue
		}
		seen := r.take()
		if len(seen) != 1 || !reflect.DeepEqual(*seen[0], *p) {
			t.Errorf("%s: server received %+v, want %+v", name, seen, *p)
		}
		body, ct, length := wire.take()
		if isNil(p) {
			if len(body) != 0 || ct != "" || length != 0 {
				t.Errorf("%s nil object: sent body %q, Content-Type %q, Content-Length %d, want no body", name, body, ct, length)
			}
			continue
		}
		if len(body) == 0 || ct != contentType {
			t.Errorf("%s %+v: sent body %q, Content-Type %q, want a %s body", name, *p, body, ct, contentType)
		}
	}
}

func TestOptionalJSONObjectBody(t *testing.T) {
	svc := &jsonService{}
	mux := loomhttp.NewMuxer()
	jsonserver.Mount(mux, jsonserver.New(jsonfind.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := jsonclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	roundTrip(t, "json", func(p *jsonfind.FindPayload) error {
		_, err := c.Find()(context.Background(), p)
		return err
	}, &svc.recorder, wire, []jsonfind.FindPayload{
		{Q: ptr("a"), O: &jsonfind.Filters{Name: ptr("n"), Limit: ptr(2)}},
		{O: &jsonfind.Filters{}},
		{Q: ptr("b")},
		{},
	}, func(p *jsonfind.FindPayload) bool { return p.O == nil }, "application/json")

	cases := []bodyCase[jsonfind.Filters]{
		{"valid", ` + "`" + `{"name":"x","limit":3}` + "`" + `, http.StatusNoContent, "", "", &jsonfind.Filters{Name: ptr("x"), Limit: ptr(3)}},
		{"empty object", "{}", http.StatusNoContent, "", "", &jsonfind.Filters{}},
		{"null", "null", http.StatusNoContent, "", "", &jsonfind.Filters{}},
		{"empty", "", http.StatusNoContent, "", "", nil},
		{"whitespace", " \n\t", http.StatusNoContent, "", "", nil},
		{"null field", ` + "`" + `{"name":null}` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", nil},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", nil},
		{"truncated", ` + "`" + `{"name":"x"` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", nil},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/json", "application/json", tc, &svc.recorder, func(p *jsonfind.FindPayload) *jsonfind.Filters { return p.O })
	}
}

func TestOptionalJSONObjectBodyWithRequiredFields(t *testing.T) {
	svc := &strictService{}
	mux := loomhttp.NewMuxer()
	strictserver.Mount(mux, strictserver.New(strictfind.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := strictclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	roundTrip(t, "strict", func(p *strictfind.FindPayload) error {
		_, err := c.Find()(context.Background(), p)
		return err
	}, &svc.recorder, wire, []strictfind.FindPayload{
		{Q: ptr("a"), O: &strictfind.Strict{Name: "n", Limit: ptr(2)}},
		{Q: ptr("b")},
		{},
	}, func(p *strictfind.FindPayload) bool { return p.O == nil }, "application/json")

	cases := []bodyCase[strictfind.Strict]{
		{"valid", ` + "`" + `{"name":"x"}` + "`" + `, http.StatusNoContent, "", "", &strictfind.Strict{Name: "x"}},
		{"empty", "", http.StatusNoContent, "", "", nil},
		{"whitespace", " \n\t", http.StatusNoContent, "", "", nil},
		{"empty object", "{}", http.StatusBadRequest, "missing_field", "Missing required field: name", nil},
		{"null", "null", http.StatusBadRequest, "missing_field", "Missing required field: name", nil},
		{"missing required field", ` + "`" + `{"limit":1}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", nil},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", nil},
		{"truncated", ` + "`" + `{"name":"x"` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", nil},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/strict", "application/json", tc, &svc.recorder, func(p *strictfind.FindPayload) *strictfind.Strict { return p.O })
	}
}

func TestDocumentedOptionalRequestBody(t *testing.T) {
	svc := &docService{}
	mux := loomhttp.NewMuxer()
	docserver.Mount(mux, docserver.New(docfind.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := docclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	roundTrip(t, "doc", func(p *docfind.SearchPayload) error {
		_, err := c.Search()(context.Background(), p)
		return err
	}, &svc.recorder, wire, []docfind.SearchPayload{
		{Query: ptr("a"), Filters: &docfind.SearchFilters{Tag: ptr("t")}},
		{Query: ptr("b")},
	}, func(p *docfind.SearchPayload) bool { return p.Filters == nil }, "application/json")

	cases := []bodyCase[docfind.SearchFilters]{
		{"valid", ` + "`" + `{"tag":"x"}` + "`" + `, http.StatusNoContent, "", "", &docfind.SearchFilters{Tag: ptr("x")}},
		{"empty", "", http.StatusNoContent, "", "", nil},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", nil},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/search", "application/json", tc, &svc.recorder, func(p *docfind.SearchPayload) *docfind.SearchFilters { return p.Filters })
	}
}

func TestOptionalFormObjectBody(t *testing.T) {
	svc := &formService{}
	mux := loomhttp.NewMuxer()
	formserver.Mount(mux, formserver.New(formfind.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := formclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	roundTrip(t, "form", func(p *formfind.FindPayload) error {
		_, err := c.Find()(context.Background(), p)
		return err
	}, &svc.recorder, wire, []formfind.FindPayload{
		{Q: ptr("a"), O: &formfind.Filters{Name: ptr("n"), Limit: ptr(2)}},
		{Q: ptr("b")},
		{},
	}, func(p *formfind.FindPayload) bool { return p.O == nil }, "application/x-www-form-urlencoded")

	const form = "application/x-www-form-urlencoded"
	cases := []bodyCase[formfind.Filters]{
		{"valid", "name=x&limit=3", http.StatusNoContent, "", "", &formfind.Filters{Name: ptr("x"), Limit: ptr(3)}},
		{"empty", "", http.StatusNoContent, "", "", nil},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/form", form, tc, &svc.recorder, func(p *formfind.FindPayload) *formfind.Filters { return p.O })
	}
}

func TestRequiredJSONObjectBody(t *testing.T) {
	svc := &reqService{}
	mux := loomhttp.NewMuxer()
	reqserver.Mount(mux, reqserver.New(reqfind.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := reqclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	roundTrip(t, "required", func(p *reqfind.FindPayload) error {
		_, err := c.Find()(context.Background(), p)
		return err
	}, &svc.recorder, wire, []reqfind.FindPayload{
		{Q: ptr("a"), O: &reqfind.Strict{Name: "n"}},
	}, func(p *reqfind.FindPayload) bool { return p.O == nil }, "application/json")

	cases := []bodyCase[reqfind.Strict]{
		{"valid", ` + "`" + `{"name":"x"}` + "`" + `, http.StatusNoContent, "", "", &reqfind.Strict{Name: "x"}},
		{"empty", "", http.StatusBadRequest, "missing_payload", "validation error", nil},
		{"whitespace", " \n\t", http.StatusBadRequest, "missing_payload", "validation error", nil},
		{"empty object", "{}", http.StatusBadRequest, "missing_field", "Missing required field: name", nil},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/required", "application/json", tc, &svc.recorder, func(p *reqfind.FindPayload) *reqfind.Strict { return p.O })
	}
}

// wireRecorder records the body, Content-Type and Content-Length of the last
// request before passing it on to next.
type wireRecorder struct {
	next          http.Handler
	mu            sync.Mutex
	body          []byte
	contentType   string
	contentLength int64
}

func (w *wireRecorder) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	w.mu.Lock()
	w.body, w.contentType, w.contentLength = body, r.Header.Get("Content-Type"), r.ContentLength
	w.mu.Unlock()
	r.Body = io.NopCloser(bytes.NewReader(body))
	w.next.ServeHTTP(rw, r)
}

func (w *wireRecorder) take() ([]byte, string, int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body, w.contentType, w.contentLength
}

// bodyCase is a raw request body sent to a generated server with the expected
// response status, problem code and detail, and the object the service
// receives when the request succeeds. A nil want on success means the service
// receives a nil object because the optional body is absent.
type bodyCase[O any] struct {
	name   string
	body   string
	status int
	code   string
	detail string
	want   *O
}

// checkBody posts the body of tc with the content type ct and asserts the
// status and, for an error, the problem code and detail and that the service
// was not invoked. For a success it asserts that the service was invoked once
// with the object want.
func checkBody[O, P any](t *testing.T, hs *httptest.Server, path, ct string, tc bodyCase[O], r *recorder[P], object func(*P) *O) {
	t.Helper()
	resp, err := hs.Client().Post(hs.URL+path, ct, strings.NewReader(tc.body))
	if err != nil {
		t.Fatalf("%s: %v", tc.name, err)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s: read body: %v", tc.name, err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("%s: close body: %v", tc.name, err)
	}
	if resp.StatusCode != tc.status {
		t.Errorf("%s %s: status %d, want %d (%s)", path, tc.name, resp.StatusCode, tc.status, raw)
	}
	seen := r.take()
	if tc.status != http.StatusNoContent {
		var problem loomhttp.ProblemResponse
		if err := json.Unmarshal(raw, &problem); err != nil {
			t.Errorf("%s %s: decode problem %q: %v", path, tc.name, raw, err)
		}
		if problem.Code != tc.code || problem.Detail != tc.detail {
			t.Errorf("%s %s: problem (%q, %q), want (%q, %q)", path, tc.name, problem.Code, problem.Detail, tc.code, tc.detail)
		}
		if len(seen) != 0 {
			t.Errorf("%s %s: service invoked with %+v", path, tc.name, seen)
		}
		return
	}
	if len(seen) != 1 {
		t.Errorf("%s %s: service invoked %d times, want 1", path, tc.name, len(seen))
		return
	}
	if got := object(seen[0]); !reflect.DeepEqual(got, tc.want) {
		t.Errorf("%s %s: service received %+v, want %+v", path, tc.name, got, tc.want)
	}
}
`
