package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestAliasRequestBodyGeneratedIntegration generates a service whose methods
// select an optional or required alias of a primitive, an array or a map
// with validations, or a union with an alias of a primitive branch, with
// Body. It compiles and vets it in a temporary module, round-trips values
// through the generated client and server, and sends empty, whitespace,
// null, valid, invalid, malformed and truncated bodies to the generated
// server, checking the status, the problem code and detail, whether the
// service is invoked and the decoded value. It also calls the generated CLI
// payload builders.
func TestAliasRequestBodyGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/aliasbody"

	root := RunHTTPDSL(t, aliasBodyModuleDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alias_body_test.go"), []byte(aliasRequestBodyHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

// aliasBodyModuleDSL declares a service whose methods select an optional
// and a required alias of String, Int, an array and a map with validations,
// and a union with an alias of String branch, with Body("v").
func aliasBodyModuleDSL() {
	var Code = Type("Code", String, func() {
		MinLength(1)
	})
	var Amount = Type("Amount", Int, func() {
		Minimum(1)
	})
	var Tags = Type("Tags", ArrayOf(String), func() {
		MinLength(1)
	})
	var Dict = Type("Dict", MapOf(String, Int), func() {
		MaxLength(2)
	})
	var Leaf = Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	methods := []struct {
		name string
		att  any
	}{
		{"code", Code},
		{"amount", Amount},
		{"tags", Tags},
		{"dict", Dict},
		{"union", OneOf(Leaf, Code)},
	}
	Service("aliases", func() {
		for _, m := range methods {
			for _, required := range []bool{false, true} {
				name := "send" + m.name
				if required {
					name += "req"
				}
				Method(name, func() {
					Payload(func() {
						Attribute("q", String)
						Attribute("v", m.att)
						if required {
							Required("v")
						}
					})
					HTTP(func() {
						POST("/" + name)
						Param("q")
						Body("v")
					})
				})
			}
		}
	})
}

const aliasRequestBodyHarness = `package aliasbody

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	json "encoding/json/v2"

	aliases "example.com/aliasbody/gen/aliases"
	client "example.com/aliasbody/gen/http/aliases/client"
	server "example.com/aliasbody/gen/http/aliases/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []any
}

func (s *service) Sendcode(_ context.Context, p *aliases.SendcodePayload) error {
	return s.record(p)
}
func (s *service) Sendcodereq(_ context.Context, p *aliases.SendcodereqPayload) error {
	return s.record(p)
}
func (s *service) Sendamount(_ context.Context, p *aliases.SendamountPayload) error {
	return s.record(p)
}
func (s *service) Sendamountreq(_ context.Context, p *aliases.SendamountreqPayload) error {
	return s.record(p)
}
func (s *service) Sendtags(_ context.Context, p *aliases.SendtagsPayload) error {
	return s.record(p)
}
func (s *service) Sendtagsreq(_ context.Context, p *aliases.SendtagsreqPayload) error {
	return s.record(p)
}
func (s *service) Senddict(_ context.Context, p *aliases.SenddictPayload) error {
	return s.record(p)
}
func (s *service) Senddictreq(_ context.Context, p *aliases.SenddictreqPayload) error {
	return s.record(p)
}
func (s *service) Sendunion(_ context.Context, p *aliases.SendunionPayload) error {
	return s.record(p)
}
func (s *service) Sendunionreq(_ context.Context, p *aliases.SendunionreqPayload) error {
	return s.record(p)
}

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

// wireRecorder records the body of the last request.
type wireRecorder struct {
	next http.Handler
	mu   sync.Mutex
	body []byte
}

func (w *wireRecorder) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	w.mu.Lock()
	w.body = body
	w.mu.Unlock()
	r.Body = io.NopCloser(bytes.NewReader(body))
	w.next.ServeHTTP(rw, r)
}

func (w *wireRecorder) take() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body
}

func ptr[T any](v T) *T {
	return &v
}

func serve(t *testing.T) (*service, *wireRecorder, *httptest.Server, *client.Client) {
	t.Helper()
	svc := &service{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(aliases.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	t.Cleanup(hs.Close)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return svc, wire, hs, c
}

func TestRoundTrip(t *testing.T) {
	svc, wire, _, c := serve(t)
	code := aliases.NewCodeOrLeafCode(aliases.Code("ab"))
	leaf := aliases.NewCodeOrLeafLeaf(&aliases.Leaf{Name: "a"})
	trips := []struct {
		name    string
		call    func(context.Context, any) (any, error)
		payload any
		body    string
	}{
		{"code", c.Sendcode(), &aliases.SendcodePayload{Q: ptr("q"), V: ptr(aliases.Code("ab"))}, "\"ab\""},
		{"code nil", c.Sendcode(), &aliases.SendcodePayload{}, ""},
		{"codereq", c.Sendcodereq(), &aliases.SendcodereqPayload{V: "ab"}, "\"ab\""},
		{"amount", c.Sendamount(), &aliases.SendamountPayload{V: ptr(aliases.Amount(2))}, "2"},
		{"amount nil", c.Sendamount(), &aliases.SendamountPayload{}, ""},
		{"amountreq", c.Sendamountreq(), &aliases.SendamountreqPayload{V: 2}, "2"},
		{"tags", c.Sendtags(), &aliases.SendtagsPayload{V: aliases.Tags{"a"}}, "[\"a\"]"},
		{"tags nil", c.Sendtags(), &aliases.SendtagsPayload{}, ""},
		{"tagsreq", c.Sendtagsreq(), &aliases.SendtagsreqPayload{V: aliases.Tags{"a"}}, "[\"a\"]"},
		{"dict", c.Senddict(), &aliases.SenddictPayload{V: aliases.Dict{"k": 1}}, "{\"k\":1}"},
		{"dict nil", c.Senddict(), &aliases.SenddictPayload{}, ""},
		{"dictreq", c.Senddictreq(), &aliases.SenddictreqPayload{V: aliases.Dict{"k": 1}}, "{\"k\":1}"},
		{"union code", c.Sendunion(), &aliases.SendunionPayload{V: &code}, "{\"type\":\"Code\",\"value\":\"ab\"}"},
		{"union leaf", c.Sendunion(), &aliases.SendunionPayload{V: &leaf}, "{\"type\":\"Leaf\",\"value\":{\"name\":\"a\"}}"},
		{"union nil", c.Sendunion(), &aliases.SendunionPayload{}, ""},
		{"unionreq code", c.Sendunionreq(), &aliases.SendunionreqPayload{V: code}, "{\"type\":\"Code\",\"value\":\"ab\"}"},
	}
	for _, trip := range trips {
		if _, err := trip.call(context.Background(), trip.payload); err != nil {
			t.Errorf("%s: %v", trip.name, err)
			continue
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(seen[0], trip.payload) {
			t.Errorf("%s: server received %#v, want %#v", trip.name, seen, trip.payload)
		}
		if got := strings.TrimSpace(string(wire.take())); got != trip.body {
			t.Errorf("%s: sent body %q, want %q", trip.name, got, trip.body)
		}
	}
	// The client validates nothing: an invalid value reaches the server,
	// which rejects it.
	if _, err := c.Sendcodereq()(context.Background(), &aliases.SendcodereqPayload{V: ""}); err == nil {
		t.Errorf("codereq empty code: no error")
	}
	if seen := svc.take(); len(seen) != 0 {
		t.Errorf("codereq empty code: service invoked with %#v", seen)
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
	code := aliases.NewCodeOrLeafCode(aliases.Code("ab"))
	leaf := aliases.NewCodeOrLeafLeaf(&aliases.Leaf{Name: "a"})
	cases := []bodyCase{
		{"/sendcode", "empty", "", ok, "", "", (*aliases.Code)(nil)},
		{"/sendcode", "whitespace", " \n\t", ok, "", "", (*aliases.Code)(nil)},
		{"/sendcode", "valid", "\"ab\"", ok, "", "", ptr(aliases.Code("ab"))},
		{"/sendcode", "empty string", "\"\"", bad, "invalid_length", invalid, nil},
		{"/sendcode", "null", "null", bad, "invalid_length", invalid, nil},
		{"/sendcode", "wrong type", "1", bad, "decode_payload", malformed, nil},
		{"/sendcode", "malformed", "{x}", bad, "decode_payload", malformed, nil},
		{"/sendcode", "truncated", "\"ab", bad, "decode_payload", malformed, nil},
		{"/sendcodereq", "empty", "", bad, "missing_payload", invalid, nil},
		{"/sendcodereq", "whitespace", " ", bad, "missing_payload", invalid, nil},
		{"/sendcodereq", "valid", "\"ab\"", ok, "", "", aliases.Code("ab")},
		{"/sendcodereq", "empty string", "\"\"", bad, "invalid_length", invalid, nil},
		{"/sendcodereq", "null", "null", bad, "invalid_length", invalid, nil},
		{"/sendcodereq", "malformed", "{x}", bad, "decode_payload", malformed, nil},
		{"/sendcodereq", "truncated", "\"ab", bad, "decode_payload", malformed, nil},
		{"/sendamount", "empty", "", ok, "", "", (*aliases.Amount)(nil)},
		{"/sendamount", "valid", "2", ok, "", "", ptr(aliases.Amount(2))},
		{"/sendamount", "zero", "0", bad, "invalid_range", invalid, nil},
		{"/sendamount", "malformed", "\"x\"", bad, "decode_payload", malformed, nil},
		{"/sendamountreq", "empty", "", bad, "missing_payload", invalid, nil},
		{"/sendamountreq", "valid", "2", ok, "", "", aliases.Amount(2)},
		{"/sendamountreq", "zero", "0", bad, "invalid_range", invalid, nil},
		{"/sendamountreq", "truncated", "[", bad, "decode_payload", malformed, nil},
		{"/sendtags", "empty", "", ok, "", "", aliases.Tags(nil)},
		{"/sendtags", "null", "null", ok, "", "", aliases.Tags(nil)},
		{"/sendtags", "valid", "[\"a\"]", ok, "", "", aliases.Tags{"a"}},
		{"/sendtags", "empty array", "[]", bad, "invalid_length", invalid, nil},
		{"/sendtags", "truncated", "[\"a\"", bad, "decode_payload", malformed, nil},
		{"/sendtagsreq", "empty", "", bad, "missing_payload", invalid, nil},
		{"/sendtagsreq", "valid", "[\"a\"]", ok, "", "", aliases.Tags{"a"}},
		{"/sendtagsreq", "empty array", "[]", bad, "invalid_length", invalid, nil},
		{"/sendtagsreq", "null", "null", bad, "invalid_length", invalid, nil},
		{"/sendtagsreq", "malformed", "[x]", bad, "decode_payload", malformed, nil},
		{"/senddict", "empty", "", ok, "", "", aliases.Dict(nil)},
		{"/senddict", "valid", "{\"k\":1}", ok, "", "", aliases.Dict{"k": 1}},
		{"/senddict", "too long", "{\"a\":1,\"b\":2,\"c\":3}", bad, "invalid_length", invalid, nil},
		{"/senddict", "wrong type", "{\"k\":\"x\"}", bad, "decode_payload", malformed, nil},
		{"/senddictreq", "empty", "", bad, "missing_payload", invalid, nil},
		{"/senddictreq", "empty object", "{}", ok, "", "", aliases.Dict{}},
		{"/senddictreq", "too long", "{\"a\":1,\"b\":2,\"c\":3}", bad, "invalid_length", invalid, nil},
		{"/senddictreq", "truncated", "{\"k\":", bad, "decode_payload", malformed, nil},
		{"/sendunion", "empty", "", ok, "", "", (*aliases.CodeOrLeaf)(nil)},
		{"/sendunion", "code", "{\"type\":\"Code\",\"value\":\"ab\"}", ok, "", "", &code},
		{"/sendunion", "leaf", "{\"type\":\"Leaf\",\"value\":{\"name\":\"a\"}}", ok, "", "", &leaf},
		{"/sendunion", "empty code", "{\"type\":\"Code\",\"value\":\"\"}", bad, "invalid_length", invalid, nil},
		{"/sendunion", "malformed", "{x}", bad, "decode_payload", malformed, nil},
		{"/sendunionreq", "empty", "", bad, "missing_payload", invalid, nil},
		{"/sendunionreq", "code", "{\"type\":\"Code\",\"value\":\"ab\"}", ok, "", "", code},
		{"/sendunionreq", "empty code", "{\"type\":\"Code\",\"value\":\"\"}", bad, "invalid_length", invalid, nil},
		{"/sendunionreq", "truncated", "{\"type\":\"Code\"", bad, "decode_payload", malformed, nil},
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

// TestCLI calls the generated CLI payload builders, which validate the body
// flag.
func TestCLI(t *testing.T) {
	code, err := client.BuildSendcodePayload("", "")
	if err != nil || code.V != nil {
		t.Errorf("code empty flag: payload %+v, error %v, want nil value", code, err)
	}
	code, err = client.BuildSendcodePayload("\"ab\"", "")
	if err != nil || code.V == nil || *code.V != "ab" {
		t.Errorf("code flag: payload %+v, error %v, want ab", code, err)
	}
	if _, err = client.BuildSendcodePayload("\"\"", ""); err == nil {
		t.Errorf("code empty string flag: no error")
	}
	codereq, err := client.BuildSendcodereqPayload("\"ab\"", "")
	if err != nil || codereq.V != "ab" {
		t.Errorf("codereq flag: payload %+v, error %v, want ab", codereq, err)
	}
	if _, err = client.BuildSendcodereqPayload("\"\"", ""); err == nil {
		t.Errorf("codereq empty string flag: no error")
	}
	amountreq, err := client.BuildSendamountreqPayload("2", "")
	if err != nil || amountreq.V != 2 {
		t.Errorf("amountreq flag: payload %+v, error %v, want 2", amountreq, err)
	}
	if _, err = client.BuildSendamountreqPayload("0", ""); err == nil {
		t.Errorf("amountreq zero flag: no error")
	}
	tags, err := client.BuildSendtagsPayload("", "")
	if err != nil || tags.V != nil {
		t.Errorf("tags empty flag: payload %+v, error %v, want nil value", tags, err)
	}
	if _, err = client.BuildSendtagsreqPayload("[]", ""); err == nil {
		t.Errorf("tagsreq empty array flag: no error")
	}
	dictreq, err := client.BuildSenddictreqPayload("{\"k\":1}", "")
	if err != nil || !reflect.DeepEqual(dictreq.V, aliases.Dict{"k": 1}) {
		t.Errorf("dictreq flag: payload %+v, error %v, want k=1", dictreq, err)
	}
}
`
