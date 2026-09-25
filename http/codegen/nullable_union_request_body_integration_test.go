package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestNullableUnionRequestBodyGeneratedIntegration generates services whose
// request body is a nullable constructor OneOf union payload attribute
// selected with Body, optional in one service and required in the other. It
// compiles and vets them in a temporary module and round-trips absent, null
// and concrete unions through the generated client and server, checking that
// an absent optional union is sent as no body and a null union as JSON null.
// It also sends empty, whitespace, null, concrete, empty-object, invalid,
// malformed and truncated bodies to the generated servers and checks the
// status, the problem code and detail, whether the service is invoked and the
// decoded absent, null or concrete state.
func TestNullableUnionRequestBodyGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/nullunionbody"

	root := RunHTTPDSL(t, nullableUnionRequestBodyIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullable_union_body_test.go"), []byte(nullableUnionRequestBodyHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func nullableUnionRequestBodyIntegrationDSL() {
	var Leaf = Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	var Other = Type("Other", func() {
		Attribute("count", Int)
	})
	for _, required := range []bool{false, true} {
		name, path := "optnull", "/opt"
		if required {
			name, path = "reqnull", "/req"
		}
		Service(name, func() {
			Method("pick", func() {
				Payload(func() {
					Attribute("q", String)
					Attribute("u", OneOf(Leaf, Other), func() {
						Nullable()
					})
					if required {
						Required("u")
					}
				})
				HTTP(func() {
					POST(path)
					Param("q")
					Body("u")
				})
			})
		})
	}
}

const nullableUnionRequestBodyHarness = `package nullunionbody

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

	optclient "example.com/nullunionbody/gen/http/optnull/client"
	optserver "example.com/nullunionbody/gen/http/optnull/server"
	reqclient "example.com/nullunionbody/gen/http/reqnull/client"
	reqserver "example.com/nullunionbody/gen/http/reqnull/server"
	optnull "example.com/nullunionbody/gen/optnull"
	reqnull "example.com/nullunionbody/gen/reqnull"
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

type optService struct{ recorder[optnull.PickPayload] }

func (s *optService) Pick(_ context.Context, p *optnull.PickPayload) error {
	s.record(p)
	return nil
}

type reqService struct{ recorder[reqnull.PickPayload] }

func (s *reqService) Pick(_ context.Context, p *reqnull.PickPayload) error {
	s.record(p)
	return nil
}

func ptr[T any](v T) *T { return &v }

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

func TestOptionalNullableUnionBody(t *testing.T) {
	svc := &optService{}
	mux := loomhttp.NewMuxer()
	optserver.Mount(mux, optserver.New(optnull.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := optclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []struct {
		p    optnull.PickPayload
		body string
		ct   string
	}{
		{optnull.PickPayload{Q: ptr("a")}, "", ""},
		{optnull.PickPayload{U: loom.NullValue[optnull.LeafOrOther]()}, "null", "application/json"},
		{optnull.PickPayload{U: loom.NullableValue(optnull.NewLeafOrOtherLeaf(&optnull.Leaf{Name: "a"}))}, "", "application/json"},
		{optnull.PickPayload{U: loom.NullableValue(optnull.NewLeafOrOtherOther(&optnull.Other{Count: ptr(3)}))}, "", "application/json"},
	}
	for _, tc := range payloads {
		if _, err := c.Pick()(context.Background(), &tc.p); err != nil {
			t.Errorf("pick %s: %v", state(tc.p.U), err)
			continue
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(*seen[0], tc.p) {
			t.Errorf("pick %s: server received %+v, want %+v", state(tc.p.U), seen, tc.p)
		}
		body, ct, length := wire.take()
		if tc.ct == "" {
			if len(body) != 0 || ct != "" || length != 0 {
				t.Errorf("pick absent union: sent body %q, Content-Type %q, Content-Length %d, want no body", body, ct, length)
			}
			continue
		}
		if ct != tc.ct || len(body) == 0 || (tc.body != "" && strings.TrimSpace(string(body)) != tc.body) {
			t.Errorf("pick %s: sent body %q, Content-Type %q, want %q with %s", state(tc.p.U), body, ct, tc.body, tc.ct)
		}
	}

	cases := []bodyCase{
		{"empty", "", http.StatusNoContent, "", "", "absent"},
		{"whitespace", " \n\t", http.StatusNoContent, "", "", "absent"},
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"leaf", ` + "`" + `{"type":"Leaf","value":{"name":"b"}}` + "`" + `, http.StatusNoContent, "", "", state(loom.NullableValue(optnull.NewLeafOrOtherLeaf(&optnull.Leaf{Name: "b"})))},
		{"other", ` + "`" + `{"type":"Other","value":{}}` + "`" + `, http.StatusNoContent, "", "", state(loom.NullableValue(optnull.NewLeafOrOtherOther(&optnull.Other{})))},
		{"empty object", "{}", http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, ""},
		{"invalid branch", ` + "`" + `{"type":"Leaf","value":{}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", ""},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
		{"truncated", ` + "`" + `{"type":"Leaf"` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/opt", tc, &svc.recorder, func(p *optnull.PickPayload) string { return state(p.U) })
	}
}

func TestRequiredNullableUnionBody(t *testing.T) {
	svc := &reqService{}
	mux := loomhttp.NewMuxer()
	reqserver.Mount(mux, reqserver.New(reqnull.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := reqclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []reqnull.PickPayload{
		{U: loom.NullValue[reqnull.LeafOrOther]()},
		{Q: ptr("a"), U: loom.NullableValue(reqnull.NewLeafOrOtherLeaf(&reqnull.Leaf{Name: "a"}))},
	}
	for _, p := range payloads {
		if _, err := c.Pick()(context.Background(), &p); err != nil {
			t.Errorf("pick %s: %v", state(p.U), err)
			continue
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(*seen[0], p) {
			t.Errorf("pick %s: server received %+v, want %+v", state(p.U), seen, p)
		}
		if body, ct, _ := wire.take(); len(body) == 0 || ct != "application/json" {
			t.Errorf("pick %s: sent body %q, Content-Type %q, want a JSON body", state(p.U), body, ct)
		}
	}
	// An absent required union cannot be encoded.
	if _, err := c.Pick()(context.Background(), &reqnull.PickPayload{}); err == nil {
		t.Errorf("pick absent required union: no error")
	}
	if seen := svc.take(); len(seen) != 0 {
		t.Errorf("pick absent required union: service invoked with %+v", seen)
	}

	cases := []bodyCase{
		{"null", "null", http.StatusNoContent, "", "", "null"},
		{"leaf", ` + "`" + `{"type":"Leaf","value":{"name":"b"}}` + "`" + `, http.StatusNoContent, "", "", state(loom.NullableValue(reqnull.NewLeafOrOtherLeaf(&reqnull.Leaf{Name: "b"})))},
		{"empty", "", http.StatusBadRequest, "missing_payload", "validation error", ""},
		{"whitespace", " \n\t", http.StatusBadRequest, "missing_payload", "validation error", ""},
		{"empty object", "{}", http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, ""},
		{"invalid branch", ` + "`" + `{"type":"Leaf","value":{}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", ""},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", ""},
	}
	for _, tc := range cases {
		checkBody(t, hs, "/req", tc, &svc.recorder, func(p *reqnull.PickPayload) string { return state(p.U) })
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

// bodyCase is a raw JSON request body sent to a generated server with the
// expected response status, problem code and detail, and the state of the
// union the service receives when the request succeeds.
type bodyCase struct {
	name   string
	body   string
	status int
	code   string
	detail string
	want   string
}

// checkBody posts the body of tc and asserts the status and, for an error,
// the problem code and detail and that the service was not invoked. For a
// success it asserts that the service was invoked once with a union in the
// state want.
func checkBody[P any](t *testing.T, hs *httptest.Server, path string, tc bodyCase, r *recorder[P], union func(*P) string) {
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
	if got := union(seen[0]); got != tc.want {
		t.Errorf("%s %s: service received union %s, want %s", path, tc.name, got, tc.want)
	}
}
`
