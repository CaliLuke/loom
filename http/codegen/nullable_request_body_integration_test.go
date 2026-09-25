package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestNullableRequestBodyGeneratedIntegration generates services whose request
// body is a nullable object, array or string payload attribute, or an Any
// attribute, selected with Body. It compiles and vets them in a temporary
// module and round-trips absent, null and concrete values through the
// generated clients and servers, checking that an absent optional value is
// sent as no body and a null value as JSON null. It also sends empty,
// whitespace, null, concrete, invalid, malformed and truncated bodies to the
// generated servers and checks the status, the problem code and detail,
// whether the service is invoked and the decoded absent, null or concrete
// state.
func TestNullableRequestBodyGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/nullbody"

	root := RunHTTPDSL(t, nullableRequestBodyIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullable_body_test.go"), []byte(nullableRequestBodyHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func nullableRequestBodyIntegrationDSL() {
	var Leaf = Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	services := []struct {
		name      string
		required  bool
		attribute func()
	}{
		{"optobj", false, func() { Attribute("b", Leaf, func() { Nullable() }) }},
		{"reqobj", true, func() { Attribute("b", Leaf, func() { Nullable() }) }},
		{"optarr", false, func() { Attribute("b", ArrayOf(Leaf), func() { Nullable() }) }},
		{"optstr", false, func() {
			Attribute("b", String, func() {
				Nullable()
				MinLength(2)
			})
		}},
		{"optany", false, func() { Attribute("b", Any) }},
		{"reqany", true, func() { Attribute("b", Any) }},
	}
	for _, s := range services {
		Service(s.name, func() {
			Method("pick", func() {
				Payload(func() {
					Attribute("q", String)
					s.attribute()
					if s.required {
						Required("b")
					}
				})
				HTTP(func() {
					POST("/" + s.name)
					Param("q")
					Body("b")
				})
			})
		})
	}
}

const nullableRequestBodyHarness = `package nullbody

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	json "encoding/json/v2"

	optanyclient "example.com/nullbody/gen/http/optany/client"
	optanyserver "example.com/nullbody/gen/http/optany/server"
	optarrclient "example.com/nullbody/gen/http/optarr/client"
	optarrserver "example.com/nullbody/gen/http/optarr/server"
	optobjclient "example.com/nullbody/gen/http/optobj/client"
	optobjserver "example.com/nullbody/gen/http/optobj/server"
	optstrclient "example.com/nullbody/gen/http/optstr/client"
	optstrserver "example.com/nullbody/gen/http/optstr/server"
	reqanyclient "example.com/nullbody/gen/http/reqany/client"
	reqanyserver "example.com/nullbody/gen/http/reqany/server"
	reqobjclient "example.com/nullbody/gen/http/reqobj/client"
	reqobjserver "example.com/nullbody/gen/http/reqobj/server"
	optany "example.com/nullbody/gen/optany"
	optarr "example.com/nullbody/gen/optarr"
	optobj "example.com/nullbody/gen/optobj"
	optstr "example.com/nullbody/gen/optstr"
	reqany "example.com/nullbody/gen/reqany"
	reqobj "example.com/nullbody/gen/reqobj"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
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

type optobjService struct{ recorder[optobj.PickPayload] }

func (s *optobjService) Pick(_ context.Context, p *optobj.PickPayload) error {
	s.record(p)
	return nil
}

type reqobjService struct{ recorder[reqobj.PickPayload] }

func (s *reqobjService) Pick(_ context.Context, p *reqobj.PickPayload) error {
	s.record(p)
	return nil
}

type optarrService struct{ recorder[optarr.PickPayload] }

func (s *optarrService) Pick(_ context.Context, p *optarr.PickPayload) error {
	s.record(p)
	return nil
}

type optstrService struct{ recorder[optstr.PickPayload] }

func (s *optstrService) Pick(_ context.Context, p *optstr.PickPayload) error {
	s.record(p)
	return nil
}

type optanyService struct{ recorder[optany.PickPayload] }

func (s *optanyService) Pick(_ context.Context, p *optany.PickPayload) error {
	s.record(p)
	return nil
}

type reqanyService struct{ recorder[reqany.PickPayload] }

func (s *reqanyService) Pick(_ context.Context, p *reqany.PickPayload) error {
	s.record(p)
	return nil
}

// state describes an absent, null or concrete nullable value.
func state[T any](n loom.Nullable[T]) string {
	switch {
	case !n.Present():
		return "absent"
	case n.IsNull():
		return "null"
	}
	v, _ := n.Value()
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("marshal error: %v", err)
	}
	return string(b)
}

// anyState describes an absent or present Any value.
func anyState(v loom.JSONValue) string {
	if v == nil {
		return "absent"
	}
	return string(v)
}

// roundTrip is a payload sent by a generated client with the state the
// service must receive and the request body that must be sent: no body when
// ct is empty, and a body of Content-Type ct equal to body otherwise.
type roundTrip[P any] struct {
	p    P
	want string
	body string
	ct   string
}

// bodyCase is a raw JSON request body sent to a generated server with the
// expected response status, problem code and detail, and the state of the
// body attribute the service receives when the request succeeds.
type bodyCase struct {
	name   string
	body   string
	status int
	code   string
	detail string
	want   string
}

// serve mounts a generated server on a test HTTP server that records the
// last request body.
func serve(t *testing.T, mount func(loomhttp.Muxer)) (*httptest.Server, *wireRecorder) {
	t.Helper()
	mux := loomhttp.NewMuxer()
	mount(mux)
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	t.Cleanup(hs.Close)
	return hs, wire
}

func host(hs *httptest.Server) string { return strings.TrimPrefix(hs.URL, "http://") }

// checkRoundTrips sends each payload through the generated client endpoint
// pick and asserts the service state and the request body.
func checkRoundTrips[P any](t *testing.T, pick func(context.Context, any) (any, error), r *recorder[P], wire *wireRecorder, cases []roundTrip[P], attr func(*P) string) {
	t.Helper()
	for _, tc := range cases {
		if _, err := pick(context.Background(), &tc.p); err != nil {
			t.Errorf("pick %s: %v", tc.want, err)
			continue
		}
		seen := r.take()
		if len(seen) != 1 || attr(seen[0]) != tc.want {
			t.Errorf("pick %s: server received %d payloads (%v)", tc.want, len(seen), seen)
		}
		body, ct, length := wire.take()
		if tc.ct == "" {
			if len(body) != 0 || ct != "" || length != 0 {
				t.Errorf("pick %s: sent body %q, Content-Type %q, Content-Length %d, want no body", tc.want, body, ct, length)
			}
			continue
		}
		if ct != tc.ct || strings.TrimSpace(string(body)) != tc.body {
			t.Errorf("pick %s: sent body %q, Content-Type %q, want %q with %s", tc.want, body, ct, tc.body, tc.ct)
		}
	}
}

// checkAbsentRequired asserts that the generated client endpoint pick fails
// to send the payload p whose required body attribute is absent.
func checkAbsentRequired[P any](t *testing.T, pick func(context.Context, any) (any, error), r *recorder[P], p *P) {
	t.Helper()
	if _, err := pick(context.Background(), p); err == nil {
		t.Errorf("pick absent required body: no error")
	}
	if seen := r.take(); len(seen) != 0 {
		t.Errorf("pick absent required body: service invoked with %+v", seen)
	}
}

func TestOptionalNullableObjectBody(t *testing.T) {
	svc := &optobjService{}
	hs, wire := serve(t, func(mux loomhttp.Muxer) {
		optobjserver.Mount(mux, optobjserver.New(optobj.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := optobjclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	attr := func(p *optobj.PickPayload) string { return state(p.B) }
	checkRoundTrips(t, c.Pick(), &svc.recorder, wire, []roundTrip[optobj.PickPayload]{
		{p: optobj.PickPayload{}, want: "absent"},
		{p: optobj.PickPayload{B: loom.NullValue[optobj.Leaf]()}, want: "null", body: "null", ct: "application/json"},
		{p: optobj.PickPayload{B: loom.NullableValue(optobj.Leaf{Name: "a"})}, want: ` + "`" + `{"name":"a"}` + "`" + `, body: ` + "`" + `{"name":"a"}` + "`" + `, ct: "application/json"},
	}, attr)
	for _, tc := range []bodyCase{
		{"empty", "", http.StatusNoContent, "", "", "absent"},
		{"whitespace", " \n\t", http.StatusNoContent, "", "", "absent"},
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"concrete", ` + "`" + `{"name":"b"}` + "`" + `, http.StatusNoContent, "", "", ` + "`" + `{"name":"b"}` + "`" + `},
		{"empty object", "{}", http.StatusBadRequest, "missing_field", "Missing required field: name", ""},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
		{"truncated", ` + "`" + `{"name":"b"` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	} {
		checkBody(t, hs, "/optobj", tc, &svc.recorder, attr)
	}
}

func TestRequiredNullableObjectBody(t *testing.T) {
	svc := &reqobjService{}
	hs, wire := serve(t, func(mux loomhttp.Muxer) {
		reqobjserver.Mount(mux, reqobjserver.New(reqobj.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := reqobjclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	attr := func(p *reqobj.PickPayload) string { return state(p.B) }
	checkRoundTrips(t, c.Pick(), &svc.recorder, wire, []roundTrip[reqobj.PickPayload]{
		{p: reqobj.PickPayload{B: loom.NullValue[reqobj.Leaf]()}, want: "null", body: "null", ct: "application/json"},
		{p: reqobj.PickPayload{B: loom.NullableValue(reqobj.Leaf{Name: "a"})}, want: ` + "`" + `{"name":"a"}` + "`" + `, body: ` + "`" + `{"name":"a"}` + "`" + `, ct: "application/json"},
	}, attr)
	checkAbsentRequired(t, c.Pick(), &svc.recorder, &reqobj.PickPayload{})
	for _, tc := range []bodyCase{
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"concrete", ` + "`" + `{"name":"b"}` + "`" + `, http.StatusNoContent, "", "", ` + "`" + `{"name":"b"}` + "`" + `},
		{"empty", "", http.StatusBadRequest, "missing_payload", "validation error", ""},
		{"whitespace", " \n\t", http.StatusBadRequest, "missing_payload", "validation error", ""},
		{"empty object", "{}", http.StatusBadRequest, "missing_field", "Missing required field: name", ""},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	} {
		checkBody(t, hs, "/reqobj", tc, &svc.recorder, attr)
	}
}

func TestOptionalNullableArrayBody(t *testing.T) {
	svc := &optarrService{}
	hs, wire := serve(t, func(mux loomhttp.Muxer) {
		optarrserver.Mount(mux, optarrserver.New(optarr.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := optarrclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	attr := func(p *optarr.PickPayload) string { return state(p.B) }
	checkRoundTrips(t, c.Pick(), &svc.recorder, wire, []roundTrip[optarr.PickPayload]{
		{p: optarr.PickPayload{}, want: "absent"},
		{p: optarr.PickPayload{B: loom.NullValue[[]*optarr.Leaf]()}, want: "null", body: "null", ct: "application/json"},
		{p: optarr.PickPayload{B: loom.NullableValue([]*optarr.Leaf{})}, want: "[]", body: "[]", ct: "application/json"},
		{p: optarr.PickPayload{B: loom.NullableValue([]*optarr.Leaf{{Name: "a"}})}, want: ` + "`" + `[{"name":"a"}]` + "`" + `, body: ` + "`" + `[{"name":"a"}]` + "`" + `, ct: "application/json"},
	}, attr)
	for _, tc := range []bodyCase{
		{"empty", "", http.StatusNoContent, "", "", "absent"},
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"empty array", "[]", http.StatusNoContent, "", "", "[]"},
		{"concrete", ` + "`" + `[{"name":"b"}]` + "`" + `, http.StatusNoContent, "", "", ` + "`" + `[{"name":"b"}]` + "`" + `},
		{"null element", "[null]", http.StatusBadRequest, "invalid_field_type", ` + "`" + `invalid null value for "body[0]"; array element must be non-null` + "`" + `, ""},
		{"invalid element", "[{}]", http.StatusBadRequest, "missing_field", "Missing required field: name", ""},
		{"malformed", "[x]", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
		{"truncated", "[", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	} {
		checkBody(t, hs, "/optarr", tc, &svc.recorder, attr)
	}
}

func TestOptionalNullableStringBody(t *testing.T) {
	svc := &optstrService{}
	hs, wire := serve(t, func(mux loomhttp.Muxer) {
		optstrserver.Mount(mux, optstrserver.New(optstr.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := optstrclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	attr := func(p *optstr.PickPayload) string { return state(p.B) }
	checkRoundTrips(t, c.Pick(), &svc.recorder, wire, []roundTrip[optstr.PickPayload]{
		{p: optstr.PickPayload{}, want: "absent"},
		{p: optstr.PickPayload{B: loom.NullValue[string]()}, want: "null", body: "null", ct: "application/json"},
		{p: optstr.PickPayload{B: loom.NullableValue("ab")}, want: ` + "`" + `"ab"` + "`" + `, body: ` + "`" + `"ab"` + "`" + `, ct: "application/json"},
	}, attr)
	for _, tc := range []bodyCase{
		{"empty", "", http.StatusNoContent, "", "", "absent"},
		{"whitespace", " \n\t", http.StatusNoContent, "", "", "absent"},
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"concrete", ` + "`" + `"cd"` + "`" + `, http.StatusNoContent, "", "", ` + "`" + `"cd"` + "`" + `},
		{"too short", ` + "`" + `"c"` + "`" + `, http.StatusBadRequest, "invalid_length", "validation error", ""},
		{"wrong type", "1", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
		{"truncated", ` + "`" + `"cd` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	} {
		checkBody(t, hs, "/optstr", tc, &svc.recorder, attr)
	}
}

func TestOptionalAnyBody(t *testing.T) {
	svc := &optanyService{}
	hs, wire := serve(t, func(mux loomhttp.Muxer) {
		optanyserver.Mount(mux, optanyserver.New(optany.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := optanyclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	attr := func(p *optany.PickPayload) string { return anyState(p.B) }
	checkRoundTrips(t, c.Pick(), &svc.recorder, wire, []roundTrip[optany.PickPayload]{
		{p: optany.PickPayload{}, want: "absent"},
		{p: optany.PickPayload{B: loom.JSONValue("null")}, want: "null", body: "null", ct: "application/json"},
		{p: optany.PickPayload{B: loom.JSONValue(` + "`" + `{"k":[1,2.50]}` + "`" + `)}, want: ` + "`" + `{"k":[1,2.50]}` + "`" + `, body: ` + "`" + `{"k":[1,2.50]}` + "`" + `, ct: "application/json"},
	}, attr)
	for _, tc := range []bodyCase{
		{"empty", "", http.StatusNoContent, "", "", "absent"},
		{"whitespace", " \n\t", http.StatusNoContent, "", "", "absent"},
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"concrete", ` + "`" + `{"k":[1,2.50]}` + "`" + `, http.StatusNoContent, "", "", ` + "`" + `{"k":[1,2.50]}` + "`" + `},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
		{"truncated", ` + "`" + `{"k":` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	} {
		checkBody(t, hs, "/optany", tc, &svc.recorder, attr)
	}
}

func TestRequiredAnyBody(t *testing.T) {
	svc := &reqanyService{}
	hs, wire := serve(t, func(mux loomhttp.Muxer) {
		reqanyserver.Mount(mux, reqanyserver.New(reqany.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := reqanyclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	attr := func(p *reqany.PickPayload) string { return anyState(p.B) }
	checkRoundTrips(t, c.Pick(), &svc.recorder, wire, []roundTrip[reqany.PickPayload]{
		{p: reqany.PickPayload{B: loom.JSONValue("null")}, want: "null", body: "null", ct: "application/json"},
		{p: reqany.PickPayload{B: loom.JSONValue("3")}, want: "3", body: "3", ct: "application/json"},
		// A required Any has no absent state: a nil value is sent as null.
		{p: reqany.PickPayload{}, want: "null", body: "null", ct: "application/json"},
	}, attr)
	for _, tc := range []bodyCase{
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"concrete", "3", http.StatusNoContent, "", "", "3"},
		{"empty", "", http.StatusBadRequest, "missing_payload", "validation error", ""},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	} {
		checkBody(t, hs, "/reqany", tc, &svc.recorder, attr)
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

// checkBody posts the body of tc and asserts the status and, for an error,
// the problem code and detail and that the service was not invoked. For a
// success it asserts that the service was invoked once with a body attribute
// in the state want.
func checkBody[P any](t *testing.T, hs *httptest.Server, path string, tc bodyCase, r *recorder[P], attr func(*P) string) {
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
	if got := attr(seen[0]); got != tc.want {
		t.Errorf("%s %s: service received %s, want %s", path, tc.name, got, tc.want)
	}
}
`
