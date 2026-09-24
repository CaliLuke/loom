package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestUnionRequestBodyGeneratedIntegration generates services whose payload
// or streaming payload is a constructor OneOf union carried in a required
// JSON body, an optional JSON body, a form body and WebSocket messages,
// compiles and vets them in a temporary module, and round-trips every branch
// through the generated client and server, including a nil optional union,
// which the client sends as no body at all. It also sends empty, whitespace,
// malformed, truncated and invalid JSON bodies to the generated server and
// checks the status, the problem code and detail, whether the service is
// invoked and the decoded payload.
func TestUnionRequestBodyGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/unionrequestbody"

	root := RunHTTPDSL(t, unionRequestBodyIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_request_body_test.go"), []byte(unionRequestBodyHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func unionRequestBodyIntegrationDSL() {
	var Leaf = Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	var Other = Type("Other", func() {
		Attribute("count", Int)
	})
	Service("jsonpick", func() {
		Method("pick", func() {
			Payload(OneOf(Leaf, Other))
			HTTP(func() {
				POST("/json")
			})
		})
	})
	Service("formpick", func() {
		Method("pick", func() {
			Payload(OneOf(Leaf, Other))
			HTTP(func() {
				POST("/form")
				FormRequest()
			})
		})
	})
	Service("optpick", func() {
		Method("pick", func() {
			Payload(func() {
				Attribute("q", String)
				Attribute("u", OneOf(Leaf, Other))
			})
			HTTP(func() {
				POST("/opt")
				Param("q")
				Body("u")
			})
		})
	})
	Service("streampick", func() {
		Method("pick", func() {
			StreamingPayload(OneOf(Leaf, Other))
			HTTP(func() {
				GET("/stream")
			})
		})
	})
}

const unionRequestBodyHarness = `package unionrequestbody

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	json "encoding/json/v2"

	formpick "example.com/unionrequestbody/gen/formpick"
	formclient "example.com/unionrequestbody/gen/http/formpick/client"
	formserver "example.com/unionrequestbody/gen/http/formpick/server"
	jsonclient "example.com/unionrequestbody/gen/http/jsonpick/client"
	jsonserver "example.com/unionrequestbody/gen/http/jsonpick/server"
	streamclient "example.com/unionrequestbody/gen/http/streampick/client"
	streamserver "example.com/unionrequestbody/gen/http/streampick/server"
	jsonpick "example.com/unionrequestbody/gen/jsonpick"
	optclient "example.com/unionrequestbody/gen/http/optpick/client"
	optserver "example.com/unionrequestbody/gen/http/optpick/server"
	optpick "example.com/unionrequestbody/gen/optpick"
	streampick "example.com/unionrequestbody/gen/streampick"
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

type jsonService struct{ recorder[jsonpick.LeafOrOther] }

func (s *jsonService) Pick(_ context.Context, p *jsonpick.LeafOrOther) error {
	s.record(p)
	return nil
}

type formService struct{ recorder[formpick.LeafOrOther] }

func (s *formService) Pick(_ context.Context, p *formpick.LeafOrOther) error {
	s.record(p)
	return nil
}

type optService struct{ recorder[optpick.PickPayload] }

func (s *optService) Pick(_ context.Context, p *optpick.PickPayload) error {
	s.record(p)
	return nil
}

type streamService struct {
	recorder[streampick.LeafOrOther]
	done chan error
}

func (s *streamService) Pick(ctx context.Context, stream streampick.PickServerStream) error {
	err := s.recv(ctx, stream)
	s.done <- err
	return err
}

func (s *streamService) recv(ctx context.Context, stream streampick.PickServerStream) error {
	for {
		msg, err := stream.RecvWithContext(ctx)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		s.record(msg)
	}
}

func ptr[T any](v T) *T { return &v }

func TestJSONBodyRoundTrip(t *testing.T) {
	svc := &jsonService{}
	mux := loomhttp.NewMuxer()
	jsonserver.Mount(mux, jsonserver.New(jsonpick.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := jsonclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []jsonpick.LeafOrOther{
		jsonpick.NewLeafOrOtherLeaf(&jsonpick.Leaf{Name: "a"}),
		jsonpick.NewLeafOrOtherOther(&jsonpick.Other{Count: ptr(3)}),
		jsonpick.NewLeafOrOtherOther(&jsonpick.Other{}),
	}
	for _, p := range payloads {
		if _, err := c.Pick()(context.Background(), &p); err != nil {
			t.Fatalf("pick %s: %v", p.Kind(), err)
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(*seen[0], p) {
			t.Errorf("pick %s: server received %+v, want %+v", p.Kind(), seen, p)
		}
	}

	cases := []bodyCase[jsonpick.LeafOrOther]{
		{"leaf", ` + "`" + `{"type":"Leaf","value":{"name":"b"}}` + "`" + `, http.StatusNoContent, "", "", ptr(jsonpick.NewLeafOrOtherLeaf(&jsonpick.Leaf{Name: "b"}))},
		{"other", ` + "`" + `{"type":"Other","value":{"count":7}}` + "`" + `, http.StatusNoContent, "", "", ptr(jsonpick.NewLeafOrOtherOther(&jsonpick.Other{Count: ptr(7)}))},
		{"empty", "", http.StatusBadRequest, "missing_payload", "validation error", nil},
		{"whitespace", " \n\t", http.StatusBadRequest, "missing_payload", "validation error", nil},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", nil},
		{"truncated", ` + "`" + `{"type":"Leaf"` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", nil},
		{"unknown branch", ` + "`" + `{"type":"Nope","value":{}}` + "`" + `, http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "Nope", expected one of "Leaf", "Other"` + "`" + `, nil},
		{"missing value", ` + "`" + `{"type":"Leaf"}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: value", nil},
		{"null value", ` + "`" + `{"type":"Leaf","value":null}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: value", nil},
		{"invalid branch", ` + "`" + `{"type":"Leaf","value":{}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", nil},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/json", tc, &svc.recorder, func(p *jsonpick.LeafOrOther) *jsonpick.LeafOrOther { return p })
	}
}

func TestOptionalJSONBodyRoundTrip(t *testing.T) {
	svc := &optService{}
	mux := loomhttp.NewMuxer()
	optserver.Mount(mux, optserver.New(optpick.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := optclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []optpick.PickPayload{
		{Q: ptr("a"), U: ptr(optpick.NewLeafOrOtherLeaf(&optpick.Leaf{Name: "a"}))},
		{U: ptr(optpick.NewLeafOrOtherOther(&optpick.Other{Count: ptr(3)}))},
		{Q: ptr("b")},
		{},
	}
	for _, p := range payloads {
		if _, err := c.Pick()(context.Background(), &p); err != nil {
			t.Fatalf("pick %+v: %v", p, err)
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(*seen[0], p) {
			t.Errorf("pick: server received %+v, want %+v", seen, p)
		}
		body, contentType, contentLength := wire.take()
		if p.U == nil {
			// No body at all, not JSON null: the server decodes an empty
			// body as a nil union and rejects null.
			if len(body) != 0 || contentType != "" || contentLength != 0 {
				t.Errorf("pick nil union: sent body %q, Content-Type %q, Content-Length %d, want no body", body, contentType, contentLength)
			}
			continue
		}
		if len(body) == 0 || contentType != "application/json" {
			t.Errorf("pick %s: sent body %q, Content-Type %q, want a JSON body", p.U.Kind(), body, contentType)
		}
	}

	cases := []bodyCase[optpick.LeafOrOther]{
		{"leaf", ` + "`" + `{"type":"Leaf","value":{"name":"b"}}` + "`" + `, http.StatusNoContent, "", "", ptr(optpick.NewLeafOrOtherLeaf(&optpick.Leaf{Name: "b"}))},
		{"other", ` + "`" + `{"type":"Other","value":{}}` + "`" + `, http.StatusNoContent, "", "", ptr(optpick.NewLeafOrOtherOther(&optpick.Other{}))},
		{"empty", "", http.StatusNoContent, "", "", nil},
		{"whitespace", " \n\t", http.StatusNoContent, "", "", nil},
		{"null", "null", http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, nil},
		{"empty object", "{}", http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, nil},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", nil},
		{"truncated", ` + "`" + `{"type":"Leaf"` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", nil},
		{"unknown branch", ` + "`" + `{"type":"Nope","value":{}}` + "`" + `, http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "Nope", expected one of "Leaf", "Other"` + "`" + `, nil},
		{"invalid branch", ` + "`" + `{"type":"Leaf","value":{}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", nil},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/opt", tc, &svc.recorder, func(p *optpick.PickPayload) *optpick.LeafOrOther { return p.U })
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
// response status, problem code and detail, and the union the service
// receives when the request succeeds. A nil want on success means the service
// receives no union because the optional body is absent.
type bodyCase[U any] struct {
	name   string
	body   string
	status int
	code   string
	detail string
	want   *U
}

// checkBody posts the body of tc and asserts the status and, for an error,
// the problem code and detail and that the service was not invoked. For a
// success it asserts that the service was invoked once with the union want.
func checkBody[U, P any](t *testing.T, hs *httptest.Server, path string, tc bodyCase[U], r *recorder[P], union func(*P) *U) {
	t.Helper()
	resp, err := hs.Client().Post(hs.URL+path, "application/json", strings.NewReader(tc.body))
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
		t.Errorf("%s: status %d, want %d (%s)", tc.name, resp.StatusCode, tc.status, raw)
	}
	seen := r.take()
	if tc.status != http.StatusNoContent {
		var problem loomhttp.ProblemResponse
		if err := json.Unmarshal(raw, &problem); err != nil {
			t.Errorf("%s: decode problem %q: %v", tc.name, raw, err)
		}
		if problem.Code != tc.code || problem.Detail != tc.detail {
			t.Errorf("%s: problem (%q, %q), want (%q, %q)", tc.name, problem.Code, problem.Detail, tc.code, tc.detail)
		}
		if len(seen) != 0 {
			t.Errorf("%s: service invoked with %+v", tc.name, seen)
		}
		return
	}
	if len(seen) != 1 {
		t.Errorf("%s: service invoked %d times, want 1", tc.name, len(seen))
		return
	}
	if got := union(seen[0]); !reflect.DeepEqual(got, tc.want) {
		t.Errorf("%s: service received %+v, want %+v", tc.name, got, tc.want)
	}
}

func TestFormBodyRoundTrip(t *testing.T) {
	svc := &formService{}
	mux := loomhttp.NewMuxer()
	formserver.Mount(mux, formserver.New(formpick.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := formclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []formpick.LeafOrOther{
		formpick.NewLeafOrOtherLeaf(&formpick.Leaf{Name: "a"}),
		formpick.NewLeafOrOtherOther(&formpick.Other{Count: ptr(3)}),
	}
	for _, p := range payloads {
		if _, err := c.Pick()(context.Background(), &p); err != nil {
			t.Fatalf("pick %s: %v", p.Kind(), err)
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(*seen[0], p) {
			t.Errorf("pick %s: server received %+v, want %+v", p.Kind(), seen, p)
		}
	}
}

func TestWebSocketStreamingPayloadRoundTrip(t *testing.T) {
	svc := &streamService{done: make(chan error, 1)}
	mux := loomhttp.NewMuxer()
	streamserver.Mount(mux, streamserver.New(streampick.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := streamclient.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)

	raw, err := c.Pick()(context.Background(), nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(streampick.PickClientStream)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}
	payloads := []streampick.LeafOrOther{
		streampick.NewLeafOrOtherLeaf(&streampick.Leaf{Name: "a"}),
		streampick.NewLeafOrOtherOther(&streampick.Other{Count: ptr(3)}),
	}
	for _, p := range payloads {
		if err := stream.Send(&p); err != nil {
			t.Fatalf("send %s: %v", p.Kind(), err)
		}
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	select {
	case err := <-svc.done:
		if err != nil {
			t.Fatalf("server recv: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not reach the end of the stream")
	}
	seen := svc.take()
	if len(seen) != len(payloads) {
		t.Fatalf("server received %d messages, want %d", len(seen), len(payloads))
	}
	for i, p := range payloads {
		if !reflect.DeepEqual(*seen[i], p) {
			t.Errorf("message %d: server received %+v, want %+v", i, seen[i], p)
		}
	}
}
`
