package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestMappedNamesBodies checks the HTTP bodies of payload and result
// attributes declared with a transport element name suffix, such as "n:m":
// generation does not panic, the body fields are named after the attribute
// and use the suffix as their JSON name, required attributes are validated
// and are values in the response body, and the conversions refer to the
// fields of the attributes.
func TestMappedNamesBodies(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedNamesDSL)
	services := CreateHTTPServices(root)

	var server, client, serverTransport, clientTransport string
	require.NotPanics(t, func() {
		server = filesCode(t, ServerTypeFiles("gen", services))
		client = filesCode(t, ClientTypeFiles("gen", services))
		serverTransport = filesCode(t, ServerFiles("gen", services))
		clientTransport = filesCode(t, ClientFiles("gen", services))
	})

	for _, want := range []string{
		"type EchoRequestBody struct {\n" +
			"\tN      loom.Optional[string]                                     `form:\"m,omitempty\" json:\"m,omitzero\" xml:\"m,omitempty\"`\n" +
			"\tReq    *int                                                      `form:\"r,omitempty\" json:\"r,omitempty\" xml:\"r,omitempty\"`\n",
		"type EchoResponseBody struct {\n" +
			"\tN      *string                      `form:\"m,omitempty\" json:\"m,omitempty\" xml:\"m,omitempty\"`\n" +
			"\tReq    int                          `form:\"r\" json:\"r\" xml:\"r\"`\n" +
			"\tDef    int                          `form:\"d\" json:\"d\" xml:\"d\"`\n",
		"type LeafRequestBody struct {\n" +
			"\tLeaf  loom.Optional[string] `form:\"l,omitempty\" json:\"l,omitzero\" xml:\"l,omitempty\"`\n" +
			"\tCount *int                  `form:\"c,omitempty\" json:\"c,omitempty\" xml:\"c,omitempty\"`\n",
		"\tif body.Req == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"req\", \"body\"))\n\t}\n",
		"\tif body.Count == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"count\", \"body\"))\n\t}\n",
		"\tif body.ID == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"id\", \"body\"))\n\t}\n",
		"\tv := &mappednames.Envelope{\n\t\tReq: *body.Req,\n\t}\n",
		"\t\tv.N = &vNValue\n",
		"\tbody := &EchoResponseBody{\n\t\tN:   res.N,\n\t\tReq: res.Req,\n\t\tDef: res.Def,\n\t}\n",
	} {
		assert.Contains(t, server, want)
	}
	for _, want := range []string{
		"\tbody := &EchoRequestBody{\n\t\tN:   p.N,\n\t\tReq: p.Req,\n\t\tDef: p.Def,\n\t}\n",
		"\tv := &mappednames.Envelope{\n\t\tReq: *body.Req,\n\t}\n",
		"\tif body.Count == nil {\n\t\terr = loom.MergeErrors(err, loom.MissingFieldError(\"count\", \"body\"))\n\t}\n",
	} {
		assert.Contains(t, client, want)
	}
	elemField := regexp.MustCompile(`\b(body|p|res|v)\.(M|R|D|P|O|Ls|Ix|Ch|L|C|I)\b`)
	assert.Contains(t, serverTransport, "\tres := &mappednames.Leaf{\n\t\tCount: *v.Count,\n\t}\n")
	assert.Contains(t, clientTransport, "\tres := &LeafRequestBody{\n\t\tLeaf:  v.Leaf,\n\t\tCount: v.Count,\n\t}\n")
	for _, code := range []string{server, client, serverTransport, clientTransport} {
		assert.Empty(t, elemField.FindAllString(code, -1), "fields named after the element names")
	}
}

// TestMappedNamesOpenAPIUnionBranches checks that the OpenAPI document parses
// and names the branches of a OneOf block union declared with element name
// suffixes, and their discriminator values, after the attribute names.
func TestMappedNamesOpenAPIUnionBranches(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedNamesDSL)
	spec := renderOpenAPIJSON(t, OpenAPIFiles, root)
	doc := parseOpenAPIV3Document(t, spec)

	for _, name := range []string{"ChoiceTextEnvelope", "ChoiceLeafBranchEnvelope"} {
		_, ok := doc.Components.Schemas.Get(name)
		assert.True(t, ok, name)
	}
	code := string(spec)
	assert.Contains(t, code, `"text": "#/components/schemas/ChoiceTextEnvelope"`)
	assert.Contains(t, code, `"leaf_branch": "#/components/schemas/ChoiceLeafBranchEnvelope"`)
	for _, stale := range []string{"text:t", "leaf_branch:lb", "ChoiceCh"} {
		assert.NotContains(t, code, stale)
	}
}

// TestMappedNamesGeneratedModule compiles and vets the services of
// MappedNamesDSL in a temporary module, round-trips payloads through the
// generated client and server, and sends raw bodies that use the element
// names to the generated server, checking the status, the problem code and
// detail, whether the service is invoked, the decoded payload and the JSON
// names of the response.
func TestMappedNamesGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedNamesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/httpmappednames", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_names_test.go"), []byte(mappedNamesHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

// filesCode returns the code of the sections of files that follow their
// headers.
func filesCode(t *testing.T, files []*codegen.File) string {
	t.Helper()
	var code string
	for _, file := range files {
		code += codegen.SectionsCode(t, file.Sections[1:])
	}
	return code
}

const mappedNamesHarness = `package httpmappednames

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

	"example.com/httpmappednames/gen/http/mappednames/client"
	"example.com/httpmappednames/gen/http/mappednames/server"
	mappednames "example.com/httpmappednames/gen/mappednames"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu     sync.Mutex
	echoes []*mappednames.Envelope
	inline []*mappednames.InlinePayload
}

func (s *service) Echo(_ context.Context, p *mappednames.Envelope) (*mappednames.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.echoes = append(s.echoes, p)
	return p, nil
}

func (s *service) Inline(_ context.Context, p *mappednames.InlinePayload) (*mappednames.InlineResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inline = append(s.inline, p)
	return &mappednames.InlineResult{ID: p.ID, Count: p.Count}, nil
}

func (s *service) take() ([]*mappednames.Envelope, []*mappednames.InlinePayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	echoes, inline := s.echoes, s.inline
	s.echoes, s.inline = nil, nil
	return echoes, inline
}

func ptr[T any](v T) *T { return &v }

func start(t *testing.T) (*service, *httptest.Server, *client.Client) {
	t.Helper()
	svc := &service{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(mappednames.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return svc, hs, c
}

func envelopes() map[string]*mappednames.Envelope {
	full := &mappednames.Envelope{
		N:     ptr("name"),
		Req:   7,
		Def:   5,
		Obj:   &mappednames.Leaf{Leaf: ptr("leaf"), Count: 2},
		List:  []string{"a", "b"},
		Index: map[string]*mappednames.Leaf{"k": {Count: 3}},
	}
	full.Pick.SetString("picked")
	full.Choice = &mappednames.Choice{}
	full.Choice.SetText("text")
	minimal := &mappednames.Envelope{Def: 3, Obj: &mappednames.Leaf{}}
	minimal.Pick.SetInt(4)
	branch := &mappednames.Envelope{Def: 3, Obj: &mappednames.Leaf{}}
	branch.Pick.SetInt(1)
	branch.Choice = &mappednames.Choice{}
	branch.Choice.SetLeafBranch(&mappednames.Leaf{Leaf: ptr("leaf"), Count: 9})
	return map[string]*mappednames.Envelope{"full": full, "minimal": minimal, "leaf branch": branch}
}

func TestRoundTrip(t *testing.T) {
	svc, _, c := start(t)
	for name, envelope := range envelopes() {
		res, err := c.Echo()(context.Background(), envelope)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		echoes, _ := svc.take()
		if len(echoes) != 1 || !reflect.DeepEqual(echoes[0], envelope) {
			t.Errorf("%s: service received %+v, want %+v", name, echoes, envelope)
		}
		if !reflect.DeepEqual(res, envelope) {
			t.Errorf("%s: client received %+v, want %+v", name, res, envelope)
		}
	}
	payload := &mappednames.InlinePayload{ID: "id", Count: ptr(2)}
	res, err := c.Inline()(context.Background(), payload)
	if err != nil {
		t.Fatalf("inline: %v", err)
	}
	if want := (&mappednames.InlineResult{ID: "id", Count: ptr(2)}); !reflect.DeepEqual(res, want) {
		t.Errorf("inline: client received %+v, want %+v", res, want)
	}
}

type bodyCase struct {
	name   string
	body   string
	status int
	code   string
	detail string
	want   *mappednames.Envelope
	json   map[string]any
}

func TestWire(t *testing.T) {
	svc, hs, _ := start(t)
	minimal := &mappednames.Envelope{Req: 1, Def: 3, Obj: &mappednames.Leaf{Count: 2}}
	minimal.Pick.SetInt(4)
	full := &mappednames.Envelope{N: ptr("nm"), Req: 1, Def: 8, Obj: &mappednames.Leaf{Leaf: ptr("x"), Count: 2}, List: []string{"a"}, Index: map[string]*mappednames.Leaf{"k": {Count: 5}}}
	full.Pick.SetString("s")
	full.Choice = &mappednames.Choice{}
	full.Choice.SetText("t")
	cases := []bodyCase{
		{"minimal", ` + "`" + `{"r":1,"p":{"type":"Int","value":4},"o":{"c":2}}` + "`" + `, http.StatusOK, "", "", minimal,
			map[string]any{"r": 1.0, "d": 3.0, "p": map[string]any{"type": "Int", "value": 4.0}, "o": map[string]any{"c": 2.0}}},
		{"full", ` + "`" + `{"m":"nm","r":1,"d":8,"p":{"type":"String","value":"s"},"o":{"l":"x","c":2},"ls":["a"],"ix":{"k":{"c":5}},"ch":{"type":"text","value":"t"}}` + "`" + `, http.StatusOK, "", "", full,
			map[string]any{"m": "nm", "r": 1.0, "d": 8.0, "p": map[string]any{"type": "String", "value": "s"}, "o": map[string]any{"l": "x", "c": 2.0}, "ls": []any{"a"}, "ix": map[string]any{"k": map[string]any{"c": 5.0}}, "ch": map[string]any{"type": "text", "value": "t"}}},
		{"attribute names", ` + "`" + `{"req":1,"pick":{"type":"Int","value":4},"obj":{"count":2}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: req", nil, nil},
		{"missing required", ` + "`" + `{"p":{"type":"Int","value":4},"o":{"c":2}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: req", nil, nil},
		{"missing nested required", ` + "`" + `{"r":1,"p":{"type":"Int","value":4},"o":{}}` + "`" + `, http.StatusBadRequest, "missing_field", "Missing required field: count", nil, nil},
		{"too short", ` + "`" + `{"m":"x","r":1,"p":{"type":"Int","value":4},"o":{"c":2}}` + "`" + `, http.StatusBadRequest, "invalid_length", "validation error", nil, nil},
		{"empty", "", http.StatusBadRequest, "missing_payload", "validation error", nil, nil},
		{"whitespace", " \n\t", http.StatusBadRequest, "missing_payload", "validation error", nil, nil},
		{"malformed", "{x}", http.StatusBadRequest, "decode_payload", "invalid request body", nil, nil},
		{"truncated", ` + "`" + `{"r":1` + "`" + `, http.StatusBadRequest, "decode_payload", "invalid request body", nil, nil},
	}
	for _, tc := range cases {
		resp, err := hs.Client().Post(hs.URL+"/echo", "application/json", strings.NewReader(tc.body))
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
		echoes, _ := svc.take()
		if tc.status != http.StatusOK {
			var problem loomhttp.ProblemResponse
			if err := json.Unmarshal(raw, &problem); err != nil {
				t.Errorf("%s: decode problem %q: %v", tc.name, raw, err)
			}
			if problem.Code != tc.code || problem.Detail != tc.detail {
				t.Errorf("%s: problem (%q, %q), want (%q, %q)", tc.name, problem.Code, problem.Detail, tc.code, tc.detail)
			}
			if len(echoes) != 0 {
				t.Errorf("%s: service invoked with %+v", tc.name, echoes)
			}
			continue
		}
		if len(echoes) != 1 || !reflect.DeepEqual(echoes[0], tc.want) {
			t.Errorf("%s: service received %+v, want %+v", tc.name, echoes, tc.want)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Errorf("%s: decode response %q: %v", tc.name, raw, err)
		}
		if !reflect.DeepEqual(got, tc.json) {
			t.Errorf("%s: response %s, want %v", tc.name, raw, tc.json)
		}
	}
}

func TestInlineWire(t *testing.T) {
	svc, hs, _ := start(t)
	for _, tc := range []struct {
		name   string
		body   string
		status int
		detail string
		want   *mappednames.InlinePayload
		json   string
	}{
		{"valid", ` + "`" + `{"i":"a","c":2}` + "`" + `, http.StatusOK, "", &mappednames.InlinePayload{ID: "a", Count: ptr(2)}, ` + "`" + `{"i":"a","c":2}` + "`" + `},
		{"missing required", ` + "`" + `{"c":2}` + "`" + `, http.StatusBadRequest, "Missing required field: id", nil, ""},
	} {
		req, err := http.NewRequest(http.MethodPut, hs.URL+"/inline", strings.NewReader(tc.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
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
		_, inline := svc.take()
		if tc.status != http.StatusOK {
			var problem loomhttp.ProblemResponse
			if err := json.Unmarshal(raw, &problem); err != nil || problem.Detail != tc.detail {
				t.Errorf("%s: problem %q, want detail %q (%v)", tc.name, raw, tc.detail, err)
			}
			if len(inline) != 0 {
				t.Errorf("%s: service invoked with %+v", tc.name, inline)
			}
			continue
		}
		if len(inline) != 1 || !reflect.DeepEqual(inline[0], tc.want) {
			t.Errorf("%s: service received %+v, want %+v", tc.name, inline, tc.want)
		}
		if got := strings.TrimSpace(string(raw)); got != tc.json {
			t.Errorf("%s: response %s, want %s", tc.name, got, tc.json)
		}
	}
}
`
