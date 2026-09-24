package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestUnionBodyNamesGeneratedIntegration generates services whose payloads
// and results are anonymous and named unions, including one union returned
// by two methods, compiles and vets them in a temporary module and
// round-trips every branch through the generated client and server. It sends
// invalid request bodies to the generated servers and checks the status, the
// problem code and detail, whether the service is invoked and the decoded
// payload, and the wire shape of the responses. It also serves valid and
// invalid union responses to the generated clients and checks how they
// decode.
func TestUnionBodyNamesGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/unionbodynames"

	root := RunHTTPDSL(t, unionBodyNamesIntegrationDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_body_names_test.go"), []byte(unionBodyNamesHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func unionBodyNamesIntegrationDSL() {
	leaf, other := unionBodyBranchTypes()
	var Choice = Type("Choice", OneOf(leaf, other))
	Service("anon", func() {
		Method("echo", func() {
			Payload(OneOf(leaf, other))
			Result(OneOf(leaf, other))
			HTTP(func() {
				POST("/anon")
			})
		})
	})
	Service("named", func() {
		Method("echo", func() {
			Payload(Choice)
			Result(Choice)
			HTTP(func() {
				POST("/named")
			})
		})
	})
	Service("shared", func() {
		Method("first", func() {
			Result(OneOf(leaf, other))
			HTTP(func() {
				GET("/first")
			})
		})
		Method("second", func() {
			Result(OneOf(leaf, other))
			HTTP(func() {
				GET("/second")
			})
		})
		Method("third", func() {
			Result(Choice)
			HTTP(func() {
				GET("/third")
			})
		})
	})
}

const unionBodyNamesHarness = `package unionbodynames

import (
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	anon "example.com/unionbodynames/gen/anon"
	anonclient "example.com/unionbodynames/gen/http/anon/client"
	anonserver "example.com/unionbodynames/gen/http/anon/server"
	namedclient "example.com/unionbodynames/gen/http/named/client"
	namedserver "example.com/unionbodynames/gen/http/named/server"
	sharedclient "example.com/unionbodynames/gen/http/shared/client"
	sharedserver "example.com/unionbodynames/gen/http/shared/server"
	named "example.com/unionbodynames/gen/named"
	shared "example.com/unionbodynames/gen/shared"
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

type anonService struct{ recorder[anon.LeafOrOther] }

func (s *anonService) Echo(_ context.Context, p *anon.LeafOrOther) (*anon.LeafOrOther, error) {
	s.record(p)
	return p, nil
}

type namedService struct{ recorder[named.Choice] }

func (s *namedService) Echo(_ context.Context, p *named.Choice) (*named.Choice, error) {
	s.record(p)
	return p, nil
}

type sharedService struct{}

func (sharedService) First(context.Context) (*shared.LeafOrOther, error) {
	return ptr(shared.NewLeafOrOtherLeaf(&shared.Leaf{Name: "first"})), nil
}

func (sharedService) Second(context.Context) (*shared.LeafOrOther, error) {
	return ptr(shared.NewLeafOrOtherOther(&shared.Other{Count: ptr(2)})), nil
}

func (sharedService) Third(context.Context) (*shared.Choice, error) {
	return ptr(shared.NewChoiceOther(&shared.Other{})), nil
}

func ptr[T any](v T) *T { return &v }

func serve(t *testing.T, mount func(loomhttp.Muxer)) *httptest.Server {
	t.Helper()
	mux := loomhttp.NewMuxer()
	mount(mux)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return hs
}

func host(hs *httptest.Server) string { return strings.TrimPrefix(hs.URL, "http://") }

// bodyCase is a raw request body with the expected response status and, for
// an error, the problem code and detail, or for a success the union the
// service receives and the response body.
type bodyCase struct {
	name   string
	body   string
	status int
	code   string
	detail string
	wants  bool
}

var bodyCases = []bodyCase{
	{"leaf", ` + "`" + `{"type":"Leaf","value":{"name":"b"}}` + "`" + `, http.StatusOK, "", "", true},
	{"other", ` + "`" + `{"type":"Other","value":{"count":7}}` + "`" + `, http.StatusOK, "", "", true},
	{"empty other", ` + "`" + `{"type":"Other","value":{}}` + "`" + `, http.StatusOK, "", "", true},
	{"empty", "", http.StatusBadRequest, "missing_payload", "validation error", false},
	{"whitespace", " \n\t", http.StatusBadRequest, "missing_payload", "validation error", false},
	{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", false},
	{"truncated", ` + "`" + `{"type":"Leaf"` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", false},
	{"no type", "{}", http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, false},
	{"unknown branch", ` + "`" + `{"type":"Nope","value":{}}` + "`" + `, http.StatusBadRequest, "invalid_enum_value", ` + "`" + `invalid value for "type": got "Nope", expected one of "Leaf", "Other"` + "`" + `, false},
	{"missing value", ` + "`" + `{"type":"Leaf"}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: value", false},
	{"null value", ` + "`" + `{"type":"Leaf","value":null}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: value", false},
	{"invalid branch", ` + "`" + `{"type":"Leaf","value":{}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", false},
}

// checkBodies posts every body case and checks the status, the problem code
// and detail of errors, that errors do not invoke the service, and that a
// success invokes it once with a union whose kind is the one on the wire and
// echoes the request body unchanged.
func checkBodies[P interface{ Kind() K }, K ~string](t *testing.T, hs *httptest.Server, path string, r *recorder[P]) {
	t.Helper()
	for _, tc := range bodyCases {
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
		if !tc.wants {
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
			continue
		}
		var sent struct {
			Type string ` + "`" + `json:"type"` + "`" + `
		}
		if err := json.Unmarshal([]byte(tc.body), &sent); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(seen) != 1 || string((*seen[0]).Kind()) != sent.Type {
			t.Errorf("%s: service received %+v, want kind %s", tc.name, seen, sent.Type)
		}
		if got := strings.TrimSpace(string(raw)); got != tc.body {
			t.Errorf("%s: response body %s, want %s", tc.name, got, tc.body)
		}
	}
}

func TestAnonymousUnionRoundTrip(t *testing.T) {
	svc := &anonService{}
	hs := serve(t, func(mux loomhttp.Muxer) {
		anonserver.Mount(mux, anonserver.New(anon.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := anonclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	payloads := []anon.LeafOrOther{
		anon.NewLeafOrOtherLeaf(&anon.Leaf{Name: "a"}),
		anon.NewLeafOrOtherOther(&anon.Other{Count: ptr(3)}),
		anon.NewLeafOrOtherOther(&anon.Other{}),
	}
	for _, p := range payloads {
		res, err := c.Echo()(context.Background(), &p)
		if err != nil {
			t.Fatalf("echo %s: %v", p.Kind(), err)
		}
		if got, ok := res.(*anon.LeafOrOther); !ok || !reflect.DeepEqual(*got, p) {
			t.Errorf("echo %s: result %#v, want %#v", p.Kind(), res, p)
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(*seen[0], p) {
			t.Errorf("echo %s: server received %+v, want %+v", p.Kind(), seen, p)
		}
	}
	checkBodies(t, hs, "/anon", &svc.recorder)
}

func TestNamedUnionRoundTrip(t *testing.T) {
	svc := &namedService{}
	hs := serve(t, func(mux loomhttp.Muxer) {
		namedserver.Mount(mux, namedserver.New(named.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := namedclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	payloads := []named.Choice{
		named.NewChoiceLeaf(&named.Leaf{Name: "a"}),
		named.NewChoiceOther(&named.Other{Count: ptr(3)}),
	}
	for _, p := range payloads {
		res, err := c.Echo()(context.Background(), &p)
		if err != nil {
			t.Fatalf("echo %s: %v", p.Kind(), err)
		}
		if got, ok := res.(*named.Choice); !ok || !reflect.DeepEqual(*got, p) {
			t.Errorf("echo %s: result %#v, want %#v", p.Kind(), res, p)
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(*seen[0], p) {
			t.Errorf("echo %s: server received %+v, want %+v", p.Kind(), seen, p)
		}
	}
	checkBodies(t, hs, "/named", &svc.recorder)
}

func TestSharedUnionResults(t *testing.T) {
	hs := serve(t, func(mux loomhttp.Muxer) {
		sharedserver.Mount(mux, sharedserver.New(shared.NewEndpoints(sharedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	})
	c := sharedclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	cases := []struct {
		name     string
		endpoint func(context.Context, any) (any, error)
		want     any
	}{
		{"first", c.First(), ptr(shared.NewLeafOrOtherLeaf(&shared.Leaf{Name: "first"}))},
		{"second", c.Second(), ptr(shared.NewLeafOrOtherOther(&shared.Other{Count: ptr(2)}))},
		{"third", c.Third(), ptr(shared.NewChoiceOther(&shared.Other{}))},
	}
	for _, tc := range cases {
		res, err := tc.endpoint(ctx, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !reflect.DeepEqual(res, tc.want) {
			t.Errorf("%s: result %#v, want %#v", tc.name, res, tc.want)
		}
	}
}

// TestClientDecodesUnionResponses serves raw response bodies to the
// generated clients and checks that valid unions decode to the right branch
// and invalid ones fail.
func TestClientDecodesUnionResponses(t *testing.T) {
	var response string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, response); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer hs.Close()
	anonc := anonclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	namedc := namedclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	sharedc := sharedclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	payload := anon.NewLeafOrOtherOther(&anon.Other{})
	namedPayload := named.NewChoiceOther(&named.Other{})
	endpoints := []struct {
		name string
		call func() (any, error)
		kind func(any) string
	}{
		{"anon", func() (any, error) { return anonc.Echo()(ctx, &payload) }, func(v any) string { return string(v.(*anon.LeafOrOther).Kind()) }},
		{"named", func() (any, error) { return namedc.Echo()(ctx, &namedPayload) }, func(v any) string { return string(v.(*named.Choice).Kind()) }},
		{"first", func() (any, error) { return sharedc.First()(ctx, nil) }, func(v any) string { return string(v.(*shared.LeafOrOther).Kind()) }},
		{"third", func() (any, error) { return sharedc.Third()(ctx, nil) }, func(v any) string { return string(v.(*shared.Choice).Kind()) }},
	}
	cases := []struct {
		name string
		body string
		kind string
	}{
		{"leaf", ` + "`" + `{"type":"Leaf","value":{"name":"x"}}` + "`" + `, "Leaf"},
		{"other", ` + "`" + `{"type":"Other","value":{"count":1}}` + "`" + `, "Other"},
		{"unknown branch", ` + "`" + `{"type":"Nope","value":{}}` + "`" + `, ""},
		{"no type", "{}", ""},
		{"missing value", ` + "`" + `{"type":"Leaf"}` + "`" + `, ""},
		{"invalid branch", ` + "`" + `{"type":"Leaf","value":{}}` + "`" + `, ""},
		{"malformed", "{x}", ""},
	}
	for _, e := range endpoints {
		for _, tc := range cases {
			response = tc.body
			res, err := e.call()
			if tc.kind == "" {
				if err == nil {
					t.Errorf("%s %s: decoded %#v, want an error", e.name, tc.name, res)
				}
				continue
			}
			if err != nil {
				t.Errorf("%s %s: %v", e.name, tc.name, err)
				continue
			}
			if got := e.kind(res); got != tc.kind {
				t.Errorf("%s %s: kind %s, want %s", e.name, tc.name, got, tc.kind)
			}
		}
	}
}
`
