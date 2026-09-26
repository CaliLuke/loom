package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestMappedPayloadReferencesCode checks that the HTTP code generated when
// headers, params, cookies, route wildcards, MapParams and bodies select
// payload, result and error attributes declared with an element name suffix
// is the code generated for the same attributes declared without the suffix,
// except for the JSON names of the body fields: the transport elements are
// named by the mappings, and the Go fields, validations and requiredness are
// those of the attributes.
func TestMappedPayloadReferencesCode(t *testing.T) {
	render := func(design func()) (code string) {
		root := RunHTTPDSL(t, design)
		services := CreateHTTPServices(root)
		require.NotPanics(t, func() {
			code = filesCode(t, ServerTypeFiles("gen", services)) + filesCode(t, ClientTypeFiles("gen", services)) +
				filesCode(t, ServerFiles("gen", services)) + filesCode(t, ClientFiles("gen", services)) +
				filesCode(t, PathFiles(services))
		})
		return code
	}
	mapped := render(testdata.MappedPayloadReferencesDSL)
	plain := render(mappedPayloadReferencesPlainDSL)

	tags := regexp.MustCompile(`(form|json|xml):"(nm|dt)([,"])`)
	names := map[string]string{"nm": "name", "dt": "detail"}
	normalized := tags.ReplaceAllStringFunc(mapped, func(tag string) string {
		m := tags.FindStringSubmatch(tag)
		return m[1] + `:"` + names[m[2]] + m[3]
	})
	assert.Equal(t, plain, normalized)
	for _, want := range []string{
		`tok = r.Header.Get("tok")`,
		`ver = &verRaw`,
		`r.Header.Get("X-Version")`,
		`rawValues, present := qp["q"]`,
		`r.Cookie("sid")`,
		`loomhttp.MountHandler(mux, "POST", "/items/{id}", h)`,
		`w.Header().Set("Location", res.Loc)`,
		`w.Header().Set("X-Code", res.Code)`,
		"\tName *string `form:\"nm,omitempty\" json:\"nm,omitempty\" xml:\"nm,omitempty\"`\n",
		"\tName *string `form:\"name,omitempty\" json:\"name,omitempty\" xml:\"name,omitempty\"`\n",
		`loom.MissingFieldError("tok", "header")`,
		`loom.InvalidLengthError("tok", tok, utf8.RuneCountInString(tok), 2, true)`,
	} {
		assert.Contains(t, mapped, want)
	}
	for _, stale := range []string{`"tok:t"`, `"t"`, `"qq"`, `"sid:s"`, `"loc:l"`, `"code:c"`, `"i"`} {
		assert.NotContains(t, mapped, stale)
	}
}

// TestMappedPayloadReferencesOpenAPI checks that the OpenAPI document parses
// and names the parameters after the mappings and the body properties after
// the element names.
func TestMappedPayloadReferencesOpenAPI(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedPayloadReferencesDSL)
	doc := parseOpenAPIV3Document(t, renderOpenAPIJSON(t, OpenAPIFiles, root))

	send := doc.Paths.PathItems.GetOrZero("/items/{id}")
	require.NotNil(t, send)
	require.NotNil(t, send.Post)
	params := make(map[string]string, len(send.Post.Parameters))
	for _, param := range send.Post.Parameters {
		params[param.Name] = param.In
	}
	assert.Equal(t, map[string]string{"id": "path", "tok": "header", "X-Version": "header", "q": "query", "sid": "cookie"}, params)
	body := send.Post.RequestBody.Content.GetOrZero("application/json").Schema.Schema()
	assert.NotNil(t, body.Properties.GetOrZero("nm"))
	assert.Equal(t, 1, body.Properties.Len())
	ok := send.Post.Responses.Codes.GetOrZero("200")
	assert.NotNil(t, ok.Headers.GetOrZero("Location"))
	bad := send.Post.Responses.Codes.GetOrZero("400")
	assert.NotNil(t, bad.Headers.GetOrZero("X-Code"))

	rename := doc.Paths.PathItems.GetOrZero("/rename").Post.RequestBody.Content.GetOrZero("application/json").Schema.Schema()
	assert.Equal(t, []string{"name"}, rename.Required)
	assert.NotNil(t, rename.Properties.GetOrZero("name"))
}

// TestMappedPayloadReferencesGeneratedModule compiles and vets the service of
// MappedPayloadReferencesDSL in a temporary module, round-trips payloads,
// results and errors through the generated client and server, and sends raw
// requests to the generated server, checking the status, the problem code and
// detail, whether the service is invoked, the decoded payload and the names of
// the response headers, cookies and body fields.
func TestMappedPayloadReferencesGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedPayloadReferencesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/httpmappedrefs", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_refs_test.go"), []byte(mappedPayloadReferencesHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

// TestMappedOptionalValueBodyCode checks that the HTTP code generated when
// Body("v") selects optional primitive, array, map, bytes and alias payload
// attributes, a defaulted string and a required string declared with the key
// "v:x" is the code generated for the same attributes declared as "v": the
// requiredness, default and pointer semantics of the body come from the
// attribute that the key declares.
func TestMappedOptionalValueBodyCode(t *testing.T) {
	render := func(design func()) (code string) {
		services := CreateHTTPServices(RunHTTPDSL(t, design))
		require.NotPanics(t, func() {
			code = filesCode(t, ServerTypeFiles("gen", services)) + filesCode(t, ClientTypeFiles("gen", services)) +
				filesCode(t, ServerFiles("gen", services)) + filesCode(t, ClientFiles("gen", services)) +
				filesCode(t, ClientCLIFiles("gen", services))
		})
		return code
	}
	assert.Equal(t, render(optionalValueBodyModuleDSL("v")), render(optionalValueBodyModuleDSL("v:x")))
}

// TestMappedOptionalValueBodyGeneratedModule runs the generated module test of
// optional values selected with Body (see
// TestOptionalValueRequestBodyGeneratedIntegration) on payload attributes
// declared with the key "v:x": it compiles and vets the module, round-trips
// nil and non-nil values, including an optional alias and a defaulted string,
// sends raw bodies to the server and calls the CLI payload builders.
func TestMappedOptionalValueBodyGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, optionalValueBodyModuleDSL("v:x"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/optvaluebody", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_value_body_test.go"), []byte(optionalValueRequestBodyHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_value_cli_test.go"), []byte(optionalValueRequestBodyCLIHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

// mappedPayloadReferencesPlainDSL is testdata.MappedPayloadReferencesDSL
// without the element name suffixes of the payload, result and error
// attributes.
func mappedPayloadReferencesPlainDSL() {
	var Fault = Type("Fault", func() {
		Attribute("code", String)
		Attribute("detail", String)
		Required("code")
	})
	Service("mappedrefs", func() {
		Method("send", func() {
			NoSecurity()
			Payload(func() {
				Attribute("id", Int)
				Attribute("tok", String, func() {
					MinLength(2)
				})
				Attribute("ver", String)
				Attribute("q", String)
				Attribute("sid", String)
				Attribute("name", String)
				Required("id", "tok")
			})
			Result(func() {
				Attribute("loc", String)
				Attribute("sid", String)
				Attribute("name", String)
				Required("loc")
			})
			Error("bad", Fault)
			HTTP(func() {
				POST("/items/{id}")
				Header("tok")
				Header("ver:X-Version")
				Param("q")
				Cookie("sid")
				Response(StatusOK, func() {
					Header("loc:Location")
					Cookie("sid")
				})
				Response("bad", StatusBadRequest, func() {
					Header("code:X-Code")
				})
			})
		})
		Method("put", func() {
			NoSecurity()
			Payload(func() {
				Attribute("id", Int)
				Attribute("data", MapOf(String, Int))
				Required("data")
			})
			Result(func() {
				Attribute("data", ArrayOf(String))
				Required("data")
			})
			HTTP(func() {
				PUT("/data/{id}")
				Body("data")
				Response(StatusOK, func() {
					Body("data")
				})
			})
		})
		Method("rename", func() {
			NoSecurity()
			Payload(func() {
				Attribute("name", String, func() {
					MinLength(2)
				})
				Required("name")
			})
			HTTP(func() {
				POST("/rename")
				Body(func() {
					Attribute("name")
					Required("name")
				})
			})
		})
		Method("filter", func() {
			NoSecurity()
			Payload(func() {
				Attribute("filters", MapOf(String, String))
			})
			HTTP(func() {
				GET("/filter")
				MapParams("filters")
			})
		})
	})
}

const mappedPayloadReferencesHarness = `package httpmappedrefs

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	json "encoding/json/v2"

	"example.com/httpmappedrefs/gen/http/mappedrefs/client"
	"example.com/httpmappedrefs/gen/http/mappedrefs/server"
	mappedrefs "example.com/httpmappedrefs/gen/mappedrefs"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []any
}

func (s *service) record(p any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
}

func (s *service) take() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func (s *service) Send(_ context.Context, p *mappedrefs.SendPayload) (*mappedrefs.SendResult, error) {
	s.record(p)
	if p.Name != nil && *p.Name == "fail" {
		return nil, &mappedrefs.Fault{Code: "E1", Detail: ptr("failed")}
	}
	return &mappedrefs.SendResult{Loc: "/items/" + p.Tok, Sid: p.Sid, Name: p.Name}, nil
}

func (s *service) Put(_ context.Context, p *mappedrefs.PutPayload) (*mappedrefs.PutResult, error) {
	s.record(p)
	res := &mappedrefs.PutResult{Data: []string{}}
	for k := range p.Data {
		res.Data = append(res.Data, k)
	}
	return res, nil
}

func (s *service) Rename(_ context.Context, p *mappedrefs.RenamePayload) error {
	s.record(p)
	return nil
}

func (s *service) Filter(_ context.Context, p *mappedrefs.FilterPayload) error {
	s.record(p)
	return nil
}

func ptr[T any](v T) *T { return &v }

func start(t *testing.T) (*service, *httptest.Server, *client.Client) {
	t.Helper()
	svc := &service{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(mappedrefs.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return svc, hs, c
}

func TestRoundTrip(t *testing.T) {
	svc, _, c := start(t)
	for _, payload := range []*mappedrefs.SendPayload{
		{ID: 1, Tok: "tk"},
		{ID: 2, Tok: "tok", Ver: ptr("v1"), Q: ptr("x"), Sid: ptr("s1"), Name: ptr("n")},
	} {
		res, err := c.Send()(context.Background(), payload)
		if err != nil {
			t.Errorf("send %+v: %v", payload, err)
			continue
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], payload) {
			t.Errorf("send: service received %+v, want %+v", seen, payload)
		}
		want := &mappedrefs.SendResult{Loc: "/items/" + payload.Tok, Sid: payload.Sid, Name: payload.Name}
		if !reflect.DeepEqual(res, want) {
			t.Errorf("send: client received %+v, want %+v", res, want)
		}
	}
	_, err := c.Send()(context.Background(), &mappedrefs.SendPayload{ID: 3, Tok: "tk", Name: ptr("fail")})
	var fault *mappedrefs.Fault
	if !errors.As(err, &fault) || !reflect.DeepEqual(fault, &mappedrefs.Fault{Code: "E1", Detail: ptr("failed")}) {
		t.Errorf("send: error %v, want the fault", err)
	}
	svc.take()

	put := &mappedrefs.PutPayload{ID: ptr(4), Data: map[string]int{"a": 1}}
	res, err := c.Put()(context.Background(), put)
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], put) {
		t.Errorf("put: service received %+v, want %+v", seen, put)
	}
	if want := (&mappedrefs.PutResult{Data: []string{"a"}}); !reflect.DeepEqual(res, want) {
		t.Errorf("put: client received %+v, want %+v", res, want)
	}

	rename := &mappedrefs.RenamePayload{Name: "ab"}
	if _, err := c.Rename()(context.Background(), rename); err != nil {
		t.Errorf("rename: %v", err)
	}
	if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], rename) {
		t.Errorf("rename: service received %+v, want %+v", seen, rename)
	}

	filter := &mappedrefs.FilterPayload{Filters: map[string]string{"k": "v"}}
	if _, err := c.Filter()(context.Background(), filter); err != nil {
		t.Errorf("filter: %v", err)
	}
	if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], filter) {
		t.Errorf("filter: service received %+v, want %+v", seen, filter)
	}
}

func TestWire(t *testing.T) {
	svc, hs, _ := start(t)
	for _, tc := range []struct {
		name    string
		method  string
		path    string
		headers map[string]string
		body    string
		status  int
		code    string
		detail  string
		want    any
		resp    map[string]string
		json    string
	}{
		{"send", "POST", "/items/5?q=x", map[string]string{"tok": "tk", "X-Version": "v", "Cookie": "sid=s"}, ` + "`" + `{"nm":"n"}` + "`" + `, http.StatusOK, "", "",
			&mappedrefs.SendPayload{ID: 5, Tok: "tk", Ver: ptr("v"), Q: ptr("x"), Sid: ptr("s"), Name: ptr("n")},
			map[string]string{"Location": "/items/tk", "Set-Cookie": "sid=s"}, ` + "`" + `{"nm":"n"}` + "`" + `},
		{"send element names", "POST", "/items/5?qq=x", map[string]string{"t": "tk", "Cookie": "s=s"}, ` + "`" + `{}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: tok", nil, nil, ""},
		{"send too short", "POST", "/items/5", map[string]string{"tok": "t"}, ` + "`" + `{}` + "`" + `, http.StatusBadRequest, "invalid_length", "validation error", nil, nil, ""},
		{"send fault", "POST", "/items/5", map[string]string{"tok": "tk"}, ` + "`" + `{"nm":"fail"}` + "`" + `, http.StatusBadRequest, "", "",
			&mappedrefs.SendPayload{ID: 5, Tok: "tk", Name: ptr("fail")}, map[string]string{"X-Code": "E1"}, ` + "`" + `{"dt":"failed"}` + "`" + `},
		{"put", "PUT", "/data/3", nil, ` + "`" + `{"a":1}` + "`" + `, http.StatusOK, "", "",
			&mappedrefs.PutPayload{ID: ptr(3), Data: map[string]int{"a": 1}}, nil, ` + "`" + `["a"]` + "`" + `},
		{"rename", "POST", "/rename", nil, ` + "`" + `{"name":"ab"}` + "`" + `, http.StatusNoContent, "", "", &mappedrefs.RenamePayload{Name: "ab"}, nil, ""},
		{"rename element name", "POST", "/rename", nil, ` + "`" + `{"nm":"ab"}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", nil, nil, ""},
		{"filter", "GET", "/filter?a=1&b=2", nil, "", http.StatusNoContent, "", "", &mappedrefs.FilterPayload{Filters: map[string]string{"a": "1", "b": "2"}}, nil, ""},
	} {
		var body io.Reader
		if tc.body != "" {
			body = strings.NewReader(tc.body)
		}
		req, err := http.NewRequest(tc.method, hs.URL+tc.path, body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range tc.headers {
			req.Header.Set(k, v)
		}
		resp, err := hs.Client().Do(req)
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
		seen := svc.take()
		if tc.code != "" {
			var problem loomhttp.ProblemResponse
			if err := json.Unmarshal(raw, &problem); err != nil || problem.Code != tc.code || problem.Detail != tc.detail {
				t.Errorf("%s: problem %s, want (%q, %q) (%v)", tc.name, raw, tc.code, tc.detail, err)
			}
			if len(seen) != 0 {
				t.Errorf("%s: service invoked with %+v", tc.name, seen)
			}
			continue
		}
		if len(seen) != 1 || !reflect.DeepEqual(seen[0], tc.want) {
			t.Errorf("%s: service received %+v, want %+v", tc.name, seen, tc.want)
		}
		for k, v := range tc.resp {
			got := resp.Header.Get(k)
			if k == "Set-Cookie" {
				got, _, _ = strings.Cut(got, ";")
			}
			if got != v {
				t.Errorf("%s: response header %s %q, want %q", tc.name, k, got, v)
			}
		}
		if got := strings.TrimSpace(string(raw)); got != tc.json {
			t.Errorf("%s: response %s, want %s", tc.name, got, tc.json)
		}
	}
}
`
