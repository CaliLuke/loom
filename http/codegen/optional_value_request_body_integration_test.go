package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestOptionalValueRequestBodyGeneratedIntegration generates a service whose
// methods select an optional string, integer, array, map, bytes or string
// alias payload attribute, a defaulted string or a required string with
// Body. It compiles
// and vets it in a temporary module and round-trips nil and non-nil values
// through the generated client and server, checking that a nil value is
// sent as no body at all. It sends empty, whitespace, null, empty, valid,
// invalid, malformed and truncated bodies to the generated server and checks
// the status, the problem code and detail, whether the service is invoked
// and the decoded value. It also calls the generated CLI payload builders
// with empty and set body flags.
func TestOptionalValueRequestBodyGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/optvaluebody"

	root := RunHTTPDSL(t, optionalValueRequestBodyIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_value_body_test.go"), []byte(optionalValueRequestBodyHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_value_cli_test.go"), []byte(optionalValueRequestBodyCLIHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

const optionalValueRequestBodyHarness = `package optvaluebody

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

	client "example.com/optvaluebody/gen/http/values/client"
	server "example.com/optvaluebody/gen/http/values/server"
	values "example.com/optvaluebody/gen/values"
	loomhttp "github.com/CaliLuke/loom/http"
)

// service records the payload of every call.
type service struct {
	mu   sync.Mutex
	seen []any
}

func (s *service) Str(_ context.Context, p *values.StrPayload) error       { return s.record(p) }
func (s *service) Num(_ context.Context, p *values.NumPayload) error       { return s.record(p) }
func (s *service) Strs(_ context.Context, p *values.StrsPayload) error     { return s.record(p) }
func (s *service) Leaves(_ context.Context, p *values.LeavesPayload) error { return s.record(p) }
func (s *service) Counts(_ context.Context, p *values.CountsPayload) error { return s.record(p) }
func (s *service) Blob(_ context.Context, p *values.BlobPayload) error     { return s.record(p) }
func (s *service) Alias(_ context.Context, p *values.AliasPayload) error   { return s.record(p) }
func (s *service) Dflt(_ context.Context, p *values.DfltPayload) error     { return s.record(p) }
func (s *service) Reqstr(_ context.Context, p *values.ReqstrPayload) error { return s.record(p) }

func (s *service) record(p any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
	return nil
}

func (s *service) take() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
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

func ptr[T any](v T) *T { return &v }

func serve(t *testing.T) (*service, *wireRecorder, *httptest.Server, *client.Client) {
	t.Helper()
	svc := &service{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(values.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	t.Cleanup(hs.Close)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return svc, wire, hs, c
}

// roundTrip is a payload sent through the generated client. absent is true
// when the payload has no body value, which must be sent as no body.
type roundTrip struct {
	name    string
	call    func(context.Context) (any, error)
	payload any
	absent  bool
}

func TestRoundTrip(t *testing.T) {
	svc, wire, _, c := serve(t)
	var trips []roundTrip
	add := func(name string, call func(context.Context, any) (any, error), payload any, absent bool) {
		trips = append(trips, roundTrip{name, func(ctx context.Context) (any, error) { return call(ctx, payload) }, payload, absent})
	}
	add("str", c.Str(), &values.StrPayload{Q: ptr("a"), V: ptr("ab")}, false)
	add("str nil", c.Str(), &values.StrPayload{Q: ptr("a")}, true)
	add("num", c.Num(), &values.NumPayload{V: ptr(2)}, false)
	add("num nil", c.Num(), &values.NumPayload{}, true)
	add("strs", c.Strs(), &values.StrsPayload{V: []string{"a", "b"}}, false)
	add("strs nil", c.Strs(), &values.StrsPayload{}, true)
	add("leaves", c.Leaves(), &values.LeavesPayload{V: []*values.Leaf{{Name: "a"}}}, false)
	add("leaves empty", c.Leaves(), &values.LeavesPayload{V: []*values.Leaf{}}, false)
	add("leaves nil", c.Leaves(), &values.LeavesPayload{}, true)
	add("counts", c.Counts(), &values.CountsPayload{V: map[string]int{"k": 1}}, false)
	add("counts empty", c.Counts(), &values.CountsPayload{V: map[string]int{}}, false)
	add("counts nil", c.Counts(), &values.CountsPayload{}, true)
	add("blob", c.Blob(), &values.BlobPayload{V: []byte("ab")}, false)
	add("blob nil", c.Blob(), &values.BlobPayload{}, true)
	add("alias", c.Alias(), &values.AliasPayload{V: ptr(values.Plain("x"))}, false)
	add("alias nil", c.Alias(), &values.AliasPayload{}, true)
	add("dflt", c.Dflt(), &values.DfltPayload{V: "x"}, false)
	add("reqstr", c.Reqstr(), &values.ReqstrPayload{V: "x"}, false)
	for _, trip := range trips {
		err := func() (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					err = fmt.Errorf("client panicked: %v", rec)
				}
			}()
			_, err = trip.call(context.Background())
			return err
		}()
		if err != nil {
			t.Errorf("%s: %v", trip.name, err)
			continue
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(seen[0], trip.payload) {
			t.Errorf("%s: server received %#v, want %#v", trip.name, seen, trip.payload)
		}
		body, ct, length := wire.take()
		if trip.absent {
			if len(body) != 0 || ct != "" || length != 0 {
				t.Errorf("%s: sent body %q, Content-Type %q, Content-Length %d, want no body", trip.name, body, ct, length)
			}
			continue
		}
		if len(body) == 0 || ct != "application/json" {
			t.Errorf("%s: sent body %q, Content-Type %q, want a JSON body", trip.name, body, ct)
		}
	}
}

// bodyCase is a raw request body sent to a generated server with the expected
// response status, problem code and detail, and the value the service
// receives when the request succeeds.
type bodyCase struct {
	path   string
	name   string
	body   string
	status int
	code   string
	detail string
	want   any
}

func TestServerBodies(t *testing.T) {
	svc, _, hs, _ := serve(t)
	const (
		ok        = http.StatusNoContent
		bad       = http.StatusBadRequest
		malformed = "invalid request body"
		invalid   = "validation error"
	)
	cases := []bodyCase{
		{"/str", "empty", "", ok, "", "", (*string)(nil)},
		{"/str", "whitespace", " \n\t", ok, "", "", (*string)(nil)},
		{"/str", "valid", ` + "`" + `"ab"` + "`" + `, ok, "", "", ptr("ab")},
		{"/str", "too short", ` + "`" + `"a"` + "`" + `, bad, "invalid_length", invalid, nil},
		{"/str", "empty string", ` + "`" + `""` + "`" + `, bad, "invalid_length", invalid, nil},
		{"/str", "null", "null", bad, "invalid_length", invalid, nil},
		{"/str", "wrong type", "1", bad, "decode_payload", malformed, nil},
		{"/str", "malformed", "{x}", bad, "decode_payload", malformed, nil},
		{"/str", "truncated", ` + "`" + `"ab` + "`" + `, bad, "decode_payload", malformed, nil},
		{"/num", "empty", "", ok, "", "", (*int)(nil)},
		{"/num", "valid", "2", ok, "", "", ptr(2)},
		{"/num", "zero", "0", bad, "invalid_range", invalid, nil},
		{"/num", "null", "null", bad, "invalid_range", invalid, nil},
		{"/num", "malformed", ` + "`" + `"x"` + "`" + `, bad, "decode_payload", malformed, nil},
		{"/strs", "empty", "", ok, "", "", []string(nil)},
		{"/strs", "whitespace", " ", ok, "", "", []string(nil)},
		{"/strs", "null", "null", ok, "", "", []string(nil)},
		{"/strs", "valid", ` + "`" + `["a"]` + "`" + `, ok, "", "", []string{"a"}},
		{"/strs", "empty array", "[]", bad, "invalid_length", invalid, nil},
		{"/strs", "null element", "[null]", bad, "invalid_field_type", "invalid null value for \"body[0]\"; array element must be non-null", nil},
		{"/strs", "truncated", ` + "`" + `["a"` + "`" + `, bad, "decode_payload", malformed, nil},
		{"/leaves", "empty", "", ok, "", "", []*values.Leaf(nil)},
		{"/leaves", "empty array", "[]", ok, "", "", []*values.Leaf{}},
		{"/leaves", "valid", ` + "`" + `[{"name":"a"}]` + "`" + `, ok, "", "", []*values.Leaf{{Name: "a"}}},
		{"/leaves", "missing field", "[{}]", bad, "missing_field", "Missing required field: name", nil},
		{"/leaves", "malformed", "[{x}]", bad, "decode_payload", malformed, nil},
		{"/counts", "empty", "", ok, "", "", map[string]int(nil)},
		{"/counts", "null", "null", ok, "", "", map[string]int(nil)},
		{"/counts", "empty object", "{}", ok, "", "", map[string]int{}},
		{"/counts", "valid", ` + "`" + `{"k":1}` + "`" + `, ok, "", "", map[string]int{"k": 1}},
		{"/counts", "wrong type", ` + "`" + `{"k":"x"}` + "`" + `, bad, "decode_payload", malformed, nil},
		{"/blob", "empty", "", ok, "", "", []byte(nil)},
		{"/blob", "valid", ` + "`" + `"YWI="` + "`" + `, ok, "", "", []byte("ab")},
		{"/blob", "malformed", ` + "`" + `"!"` + "`" + `, bad, "decode_payload", malformed, nil},
		{"/alias", "empty", "", ok, "", "", (*values.Plain)(nil)},
		{"/alias", "valid", ` + "`" + `"x"` + "`" + `, ok, "", "", ptr(values.Plain("x"))},
		{"/alias", "malformed", "{x}", bad, "decode_payload", malformed, nil},
		{"/dflt", "valid", ` + "`" + `"x"` + "`" + `, ok, "", "", "x"},
		{"/reqstr", "empty", "", bad, "missing_payload", "validation error", nil},
		{"/reqstr", "valid", ` + "`" + `"x"` + "`" + `, ok, "", "", "x"},
	}
	for _, tc := range cases {
		checkBody(t, hs, svc, tc)
	}
}

// checkBody posts the body of tc and asserts the status and, for an error,
// the problem code and detail and that the service was not invoked. For a
// success it asserts that the service was invoked once with the value want.
func checkBody(t *testing.T, hs *httptest.Server, svc *service, tc bodyCase) {
	t.Helper()
	resp, err := hs.Client().Post(hs.URL+tc.path, "application/json", strings.NewReader(tc.body))
	if err != nil {
		t.Fatalf("%s %s: %v", tc.path, tc.name, err)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("%s %s: read body: %v", tc.path, tc.name, err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("%s %s: close body: %v", tc.path, tc.name, err)
	}
	if resp.StatusCode != tc.status {
		t.Errorf("%s %s: status %d, want %d (%s)", tc.path, tc.name, resp.StatusCode, tc.status, raw)
	}
	seen := svc.take()
	if tc.status != http.StatusNoContent {
		var problem loomhttp.ProblemResponse
		if err := json.Unmarshal(raw, &problem); err != nil {
			t.Errorf("%s %s: decode problem %q: %v", tc.path, tc.name, raw, err)
		}
		if problem.Code != tc.code || problem.Detail != tc.detail {
			t.Errorf("%s %s: problem (%q, %q), want (%q, %q)", tc.path, tc.name, problem.Code, problem.Detail, tc.code, tc.detail)
		}
		if len(seen) != 0 {
			t.Errorf("%s %s: service invoked with %#v", tc.path, tc.name, seen)
		}
		return
	}
	if len(seen) != 1 {
		t.Errorf("%s %s: service invoked %d times, want 1", tc.path, tc.name, len(seen))
		return
	}
	if got := reflect.ValueOf(seen[0]).Elem().FieldByName("V").Interface(); !reflect.DeepEqual(got, tc.want) {
		t.Errorf("%s %s: service received %#v, want %#v", tc.path, tc.name, got, tc.want)
	}
}
`

// optionalValueRequestBodyCLIHarness tests the generated CLI payload
// builders of optionalValueRequestBodyIntegrationDSL.
const optionalValueRequestBodyCLIHarness = `package optvaluebody

import (
	"reflect"
	"testing"

	client "example.com/optvaluebody/gen/http/values/client"
)

// TestCLI calls the generated CLI payload builders. An empty body flag leaves
// an optional value nil, and a set flag is decoded.
func TestCLI(t *testing.T) {
	str, err := client.BuildStrPayload("", "q")
	if err != nil || str.V != nil || str.Q == nil || *str.Q != "q" {
		t.Errorf("str empty flag: payload %+v, error %v, want nil value", str, err)
	}
	str, err = client.BuildStrPayload("ab", "")
	if err != nil || str.V == nil || *str.V != "ab" {
		t.Errorf("str flag: payload %+v, error %v, want ab", str, err)
	}
	num, err := client.BuildNumPayload("", "")
	if err != nil || num.V != nil {
		t.Errorf("num empty flag: payload %+v, error %v, want nil value", num, err)
	}
	num, err = client.BuildNumPayload("3", "")
	if err != nil || num.V == nil || *num.V != 3 {
		t.Errorf("num flag: payload %+v, error %v, want 3", num, err)
	}
	if _, err = client.BuildNumPayload("x", ""); err == nil {
		t.Errorf("num invalid flag: no error")
	}
	strs, err := client.BuildStrsPayload("", "")
	if err != nil || strs.V != nil {
		t.Errorf("strs empty flag: payload %+v, error %v, want nil value", strs, err)
	}
	strs, err = client.BuildStrsPayload(` + "`" + `["a"]` + "`" + `, "")
	if err != nil || !reflect.DeepEqual(strs.V, []string{"a"}) {
		t.Errorf("strs flag: payload %+v, error %v, want [a]", strs, err)
	}
	if _, err = client.BuildStrsPayload("{x}", ""); err == nil {
		t.Errorf("strs invalid flag: no error")
	}
	leaves, err := client.BuildLeavesPayload("", "")
	if err != nil || leaves.V != nil {
		t.Errorf("leaves empty flag: payload %+v, error %v, want nil value", leaves, err)
	}
	leaves, err = client.BuildLeavesPayload(` + "`" + `[{"name":"a"}]` + "`" + `, "")
	if err != nil || len(leaves.V) != 1 || leaves.V[0].Name != "a" {
		t.Errorf("leaves flag: payload %+v, error %v, want one leaf", leaves, err)
	}
	counts, err := client.BuildCountsPayload("", "")
	if err != nil || counts.V != nil {
		t.Errorf("counts empty flag: payload %+v, error %v, want nil value", counts, err)
	}
	counts, err = client.BuildCountsPayload(` + "`" + `{"k":1}` + "`" + `, "")
	if err != nil || !reflect.DeepEqual(counts.V, map[string]int{"k": 1}) {
		t.Errorf("counts flag: payload %+v, error %v, want k=1", counts, err)
	}
	blob, err := client.BuildBlobPayload("", "")
	if err != nil || blob.V != nil {
		t.Errorf("blob empty flag: payload %+v, error %v, want nil value", blob, err)
	}
	blob, err = client.BuildBlobPayload("ab", "")
	if err != nil || string(blob.V) != "ab" {
		t.Errorf("blob flag: payload %+v, error %v, want ab", blob, err)
	}
	alias, err := client.BuildAliasPayload("", "")
	if err != nil || alias.V != nil {
		t.Errorf("alias empty flag: payload %+v, error %v, want nil value", alias, err)
	}
	alias, err = client.BuildAliasPayload(` + "`" + `"x"` + "`" + `, "")
	if err != nil || alias.V == nil || *alias.V != "x" {
		t.Errorf("alias flag: payload %+v, error %v, want x", alias, err)
	}
	dflt, err := client.BuildDfltPayload("x", "")
	if err != nil || dflt.V != "x" {
		t.Errorf("dflt flag: payload %+v, error %v, want x", dflt, err)
	}
	reqstr, err := client.BuildReqstrPayload("x", "")
	if err != nil || reqstr.V != "x" {
		t.Errorf("reqstr flag: payload %+v, error %v, want x", reqstr, err)
	}
}
`

func optionalValueRequestBodyIntegrationDSL() {
	var Leaf = Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	var Plain = Type("Plain", String)
	methods := []struct {
		name      string
		required  bool
		attribute func()
	}{
		{"str", false, func() { Attribute("v", String, func() { MinLength(2) }) }},
		{"num", false, func() { Attribute("v", Int, func() { Minimum(1) }) }},
		{"strs", false, func() { Attribute("v", ArrayOf(String), func() { MinLength(1) }) }},
		{"leaves", false, func() { Attribute("v", ArrayOf(Leaf)) }},
		{"counts", false, func() { Attribute("v", MapOf(String, Int)) }},
		{"blob", false, func() { Attribute("v", Bytes) }},
		{"alias", false, func() { Attribute("v", Plain) }},
		{"dflt", false, func() { Attribute("v", String, func() { Default("d") }) }},
		{"reqstr", true, func() { Attribute("v", String) }},
	}
	Service("values", func() {
		for _, m := range methods {
			Method(m.name, func() {
				Payload(func() {
					Attribute("q", String)
					m.attribute()
					if m.required {
						Required("v")
					}
				})
				HTTP(func() {
					POST("/" + m.name)
					Param("q")
					Body("v")
				})
			})
		}
	})
}
