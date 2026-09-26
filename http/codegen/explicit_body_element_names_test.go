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

// TestMappedExplicitBodyCode checks that the HTTP code generated for explicit
// request and response bodies whose attributes carry an element name suffix,
// such as Attribute("name:n") in Body, is the code generated for the same
// bodies without the suffixes except for the names in the struct tags:
// generation does not panic, the body fields have the types, validations and
// requiredness of the payload and result attributes, and validation errors
// name the attributes.
func TestMappedExplicitBodyCode(t *testing.T) {
	render := func(design func()) (code string) {
		services := CreateHTTPServices(RunHTTPDSL(t, design))
		require.NotPanics(t, func() {
			code = filesCode(t, ServerTypeFiles("gen", services)) + filesCode(t, ClientTypeFiles("gen", services)) +
				filesCode(t, ServerFiles("gen", services)) + filesCode(t, ClientFiles("gen", services))
		})
		return code
	}
	mapped := render(testdata.MappedExplicitBodyDSL)
	plain := render(mappedExplicitBodyPlainDSL)

	tags := regexp.MustCompile(`(form|json|xml):"(n|ag|x|y)([,"])`)
	names := map[string]string{"n": "name", "ag": "age", "x": "a", "y": "b"}
	normalized := tags.ReplaceAllStringFunc(mapped, func(tag string) string {
		m := tags.FindStringSubmatch(tag)
		return m[1] + `:"` + names[m[2]] + m[3]
	})
	assert.Equal(t, plain, normalized)
	for _, want := range []string{
		"\t// Account name\n\tName *string `form:\"n,omitempty\" json:\"n,omitempty\" xml:\"n,omitempty\"`\n" +
			"\t// Account age\n\tAge loom.Optional[int] `form:\"ag,omitempty\" json:\"ag,omitzero\" xml:\"ag,omitempty\"`\n",
		"\t// Account name\n\tName string `form:\"n\" json:\"n\" xml:\"n\"`\n",
		"\tA *int               `form:\"x,omitempty\" json:\"x,omitempty\" xml:\"x,omitempty\"`\n",
		"\tif body.Name == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"name\", \"body\"))\n\t}\n",
		`loom.InvalidLengthError("body.name", *body.Name, utf8.RuneCountInString(*body.Name), 2, true)`,
		"\tif body.A == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"a\", \"body\"))\n\t}\n",
	} {
		assert.Contains(t, mapped, want)
	}
}

// TestMappedExplicitBodyOpenAPI checks that the OpenAPI document parses and
// that the schemas of explicit bodies with suffixed attributes name their
// properties and required properties after the element names and carry the
// types and validations of the payload and result attributes.
func TestMappedExplicitBodyOpenAPI(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedExplicitBodyDSL)
	doc := parseOpenAPIV3Document(t, renderOpenAPIJSON(t, OpenAPIFiles, root))

	create := doc.Paths.PathItems.GetOrZero("/accounts/{id}")
	require.NotNil(t, create)
	require.NotNil(t, create.Post)
	request := create.Post.RequestBody.Content.GetOrZero("application/json").Schema.Schema()
	assert.Contains(t, request.Required, "n")
	name := request.Properties.GetOrZero("n").Schema()
	assert.Equal(t, []string{"string"}, name.Type)
	assert.Equal(t, "Account name", name.Description)
	require.NotNil(t, name.MinLength)
	assert.Equal(t, int64(2), *name.MinLength)
	assert.Equal(t, []string{"integer"}, request.Properties.GetOrZero("ag").Schema().Type)
	for _, stale := range []string{"name", "age", "name:n", "age:ag"} {
		assert.Nil(t, request.Properties.GetOrZero(stale), stale)
	}

	response := create.Post.Responses.Codes.GetOrZero("200").Content.GetOrZero("application/json").Schema.Schema()
	assert.Equal(t, []string{"n"}, response.Required)
	assert.Equal(t, []string{"string"}, response.Properties.GetOrZero("n").Schema().Type)
	assert.Equal(t, []string{"integer"}, response.Properties.GetOrZero("ag").Schema().Type)

	pair := doc.Paths.PathItems.GetOrZero("/pair")
	require.NotNil(t, pair)
	require.NotNil(t, pair.Post)
	body := pair.Post.RequestBody.Content.GetOrZero("application/json").Schema.Schema()
	assert.Equal(t, []string{"x"}, body.Required)
	for _, prop := range []string{"x", "y"} {
		assert.Equal(t, []string{"integer"}, body.Properties.GetOrZero(prop).Schema().Type, prop)
	}
}

// TestMappedExplicitBodyGeneratedModule compiles and vets the service of
// MappedExplicitBodyDSL in a temporary module, round-trips payloads through
// the generated client and server, and sends raw bodies to the generated
// server, checking the status, the problem code and detail, whether the
// service is invoked, the decoded payload and the JSON names of the response.
func TestMappedExplicitBodyGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedExplicitBodyDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/httpmappedbody", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_body_test.go"), []byte(mappedExplicitBodyHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

// mappedExplicitBodyPlainDSL is testdata.MappedExplicitBodyDSL without the
// element name suffixes of the body attributes.
func mappedExplicitBodyPlainDSL() {
	var Account = Type("Account", func() {
		Attribute("id", Int, "Account ID")
		Attribute("name", String, "Account name", func() {
			MinLength(2)
		})
		Attribute("age", Int, "Account age")
		Required("id", "name")
	})
	Service("mappedbody", func() {
		Method("create", func() {
			NoSecurity()
			Payload(Account)
			Result(func() {
				Attribute("id", Int, "Account ID")
				Attribute("name", String, "Account name", func() {
					MinLength(2)
				})
				Attribute("age", Int, "Account age")
				Required("name")
			})
			HTTP(func() {
				POST("/accounts/{id}")
				Body(func() {
					Attribute("name")
					Attribute("age")
				})
				Response(StatusOK, func() {
					Body(func() {
						Attribute("name")
						Attribute("age")
					})
				})
			})
		})
		Method("pair", func() {
			NoSecurity()
			Payload(func() {
				Attribute("a", Int)
				Attribute("b", Int)
				Required("a")
			})
			HTTP(func() {
				POST("/pair")
				Body(func() {
					Attribute("a")
					Attribute("b")
					Required("a")
				})
			})
		})
	})
}

const mappedExplicitBodyHarness = `package httpmappedbody

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	json "encoding/json/v2"

	"example.com/httpmappedbody/gen/http/mappedbody/client"
	"example.com/httpmappedbody/gen/http/mappedbody/server"
	mappedbody "example.com/httpmappedbody/gen/mappedbody"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu       sync.Mutex
	accounts []*mappedbody.Account
	pairs    []*mappedbody.PairPayload
}

func (s *service) Create(_ context.Context, p *mappedbody.Account) (*mappedbody.CreateResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts = append(s.accounts, p)
	return &mappedbody.CreateResult{ID: &p.ID, Name: p.Name, Age: p.Age}, nil
}

func (s *service) Pair(_ context.Context, p *mappedbody.PairPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pairs = append(s.pairs, p)
	return nil
}

func (s *service) take() ([]*mappedbody.Account, []*mappedbody.PairPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, pairs := s.accounts, s.pairs
	s.accounts, s.pairs = nil, nil
	return accounts, pairs
}

func ptr[T any](v T) *T { return &v }

func start(t *testing.T) (*service, *httptest.Server, *client.Client) {
	t.Helper()
	svc := &service{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(mappedbody.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return svc, hs, c
}

func TestRoundTrip(t *testing.T) {
	svc, _, c := start(t)
	for _, payload := range []*mappedbody.Account{{ID: 1, Name: "nm"}, {ID: 2, Name: "name", Age: ptr(40)}} {
		res, err := c.Create()(context.Background(), payload)
		if err != nil {
			t.Errorf("create %+v: %v", payload, err)
			continue
		}
		if accounts, _ := svc.take(); len(accounts) != 1 || !reflect.DeepEqual(accounts[0], payload) {
			t.Errorf("create: service received %+v, want %+v", accounts, payload)
		}
		if want := (&mappedbody.CreateResult{Name: payload.Name, Age: payload.Age}); !reflect.DeepEqual(res, want) {
			t.Errorf("create: client received %+v, want %+v", res, want)
		}
	}
	for _, payload := range []*mappedbody.PairPayload{{A: 1}, {A: 2, B: ptr(3)}} {
		if _, err := c.Pair()(context.Background(), payload); err != nil {
			t.Errorf("pair %+v: %v", payload, err)
			continue
		}
		if _, pairs := svc.take(); len(pairs) != 1 || !reflect.DeepEqual(pairs[0], payload) {
			t.Errorf("pair: service received %+v, want %+v", pairs, payload)
		}
	}
}

func TestWire(t *testing.T) {
	svc, hs, _ := start(t)
	for _, tc := range []struct {
		name    string
		path    string
		body    string
		status  int
		code    string
		detail  string
		account *mappedbody.Account
		pair    *mappedbody.PairPayload
		json    string
	}{
		{"element names", "/accounts/5", ` + "`" + `{"n":"nm","ag":3}` + "`" + `, http.StatusOK, "", "", &mappedbody.Account{ID: 5, Name: "nm", Age: ptr(3)}, nil, ` + "`" + `{"n":"nm","ag":3}` + "`" + `},
		{"required only", "/accounts/5", ` + "`" + `{"n":"nm"}` + "`" + `, http.StatusOK, "", "", &mappedbody.Account{ID: 5, Name: "nm"}, nil, ` + "`" + `{"n":"nm"}` + "`" + `},
		{"attribute names", "/accounts/5", ` + "`" + `{"name":"nm","age":3}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: name", nil, nil, ""},
		{"too short", "/accounts/5", ` + "`" + `{"n":"x"}` + "`" + `, http.StatusBadRequest, "invalid_length", "validation error", nil, nil, ""},
		{"wrong type", "/accounts/5", ` + "`" + `{"n":"nm","ag":"3"}` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", nil, nil, ""},
		{"pair", "/pair", ` + "`" + `{"x":1,"y":2}` + "`" + `, http.StatusNoContent, "", "", nil, &mappedbody.PairPayload{A: 1, B: ptr(2)}, ""},
		{"pair missing required", "/pair", ` + "`" + `{"y":2}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: a", nil, nil, ""},
		{"pair attribute names", "/pair", ` + "`" + `{"a":1,"b":2}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: a", nil, nil, ""},
	} {
		resp, err := hs.Client().Post(hs.URL+tc.path, "application/json", strings.NewReader(tc.body))
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
		accounts, pairs := svc.take()
		if tc.code != "" {
			var problem loomhttp.ProblemResponse
			if err := json.Unmarshal(raw, &problem); err != nil || problem.Code != tc.code || problem.Detail != tc.detail {
				t.Errorf("%s: problem %s, want (%q, %q) (%v)", tc.name, raw, tc.code, tc.detail, err)
			}
			if len(accounts)+len(pairs) != 0 {
				t.Errorf("%s: service invoked with %+v %+v", tc.name, accounts, pairs)
			}
			continue
		}
		if tc.account != nil && (len(accounts) != 1 || !reflect.DeepEqual(accounts[0], tc.account)) {
			t.Errorf("%s: service received %+v, want %+v", tc.name, accounts, tc.account)
		}
		if tc.pair != nil && (len(pairs) != 1 || !reflect.DeepEqual(pairs[0], tc.pair)) {
			t.Errorf("%s: service received %+v, want %+v", tc.name, pairs, tc.pair)
		}
		if got := strings.TrimSpace(string(raw)); got != tc.json {
			t.Errorf("%s: response %s, want %s", tc.name, got, tc.json)
		}
	}
}
`
