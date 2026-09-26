package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestMappedParamsRouteElementNames checks that a Param declared with an
// element name suffix, such as Param("key:k"), is the path parameter that the
// route wildcard "{k}" names: the server reads the "k" path value into the key
// attribute, the path builders take the key attribute, and the OpenAPI
// operation names the path parameter "k". Suffixed params absent from the
// route stay query parameters named after their element names.
func TestMappedParamsRouteElementNames(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedParamsDSL)
	services := CreateHTTPServices(root)

	var server, client, paths string
	require.NotPanics(t, func() {
		server = filesCode(t, ServerFiles("gen", services))
		client = filesCode(t, ClientFiles("gen", services))
		paths = filesCode(t, PathFiles(services))
	})
	for _, want := range []string{
		`loomhttp.MountHandler(mux, "GET", "/items/{k}/{id}", h)`,
		"\t\tkey = params[\"k\"]\n",
		"\t\t\tidRaw := params[\"id\"]\n",
		"\t\t\trawValues, present := qp[\"query\"]\n",
		"\t\t\tflagRaw := qp.Get(\"f\")\n",
	} {
		assert.Contains(t, server, want)
	}
	for _, want := range []string{
		"\t\tkey = p.Key\n\t\tid = p.ID\n",
		"u, err := loomhttp.RequestURL(c.scheme, c.host, ShowMappedparamsPath(key, id))\n",
	} {
		assert.Contains(t, client, want)
	}
	assert.Contains(t, paths, "func ShowMappedparamsPath(key string, id int) string {\n\treturn fmt.Sprintf(\"/items/%v/%v\", loomhttp.EscapePathSegment(key), id)\n}\n")

	doc := parseOpenAPIV3Document(t, renderOpenAPIJSON(t, OpenAPIFiles, root))
	item := doc.Paths.PathItems.GetOrZero("/items/{k}/{id}")
	require.NotNil(t, item, "OpenAPI path /items/{k}/{id}")
	require.NotNil(t, item.Get)
	params := make(map[string]string, len(item.Get.Parameters))
	for _, param := range item.Get.Parameters {
		params[param.Name] = param.In
	}
	assert.Equal(t, map[string]string{"k": "path", "id": "path", "query": "query", "f": "query"}, params)
}

// TestMappedParamsGeneratedModule compiles and vets the service of
// MappedParamsDSL in a temporary module, round-trips payloads through the
// generated client and server, and sends raw requests to the generated server,
// checking the status, the problem code and detail, whether the service is
// invoked and the decoded payload.
func TestMappedParamsGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MappedParamsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/httpmappedparams", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_params_test.go"), []byte(mappedParamsHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

const mappedParamsHarness = `package httpmappedparams

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

	"example.com/httpmappedparams/gen/http/mappedparams/client"
	"example.com/httpmappedparams/gen/http/mappedparams/server"
	mappedparams "example.com/httpmappedparams/gen/mappedparams"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []*mappedparams.ShowPayload
}

func (s *service) Show(_ context.Context, p *mappedparams.ShowPayload) (*mappedparams.ShowResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
	return &mappedparams.ShowResult{Key: p.Key, ID: p.ID, Q: p.Q, Flag: p.Flag}, nil
}

func (s *service) take() []*mappedparams.ShowPayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func ptr[T any](v T) *T { return &v }

func TestMappedParams(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(mappedparams.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	for _, payload := range []*mappedparams.ShowPayload{
		{Key: "ab", ID: 3},
		{Key: "a-b", ID: 4, Q: ptr("x"), Flag: ptr(true)},
	} {
		res, err := c.Show()(context.Background(), payload)
		if err != nil {
			t.Errorf("%+v: %v", payload, err)
			continue
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], payload) {
			t.Errorf("service received %+v, want %+v", seen, payload)
		}
		want := &mappedparams.ShowResult{Key: payload.Key, ID: payload.ID, Q: payload.Q, Flag: payload.Flag}
		if !reflect.DeepEqual(res, want) {
			t.Errorf("client received %+v, want %+v", res, want)
		}
	}

	for _, tc := range []struct {
		name   string
		path   string
		status int
		code   string
		detail string
		want   *mappedparams.ShowPayload
	}{
		{"element names", "/items/ab/3?query=x&f=true", http.StatusOK, "", "", &mappedparams.ShowPayload{Key: "ab", ID: 3, Q: ptr("x"), Flag: ptr(true)}},
		{"attribute names in query", "/items/ab/3?q=x&flag=true", http.StatusOK, "", "", &mappedparams.ShowPayload{Key: "ab", ID: 3}},
		{"invalid path value", "/items/a/3", http.StatusBadRequest, "invalid_length", "validation error", nil},
		{"invalid implicit path value", "/items/ab/x", http.StatusBadRequest, "invalid_field_type", "validation error", nil},
	} {
		resp, err := hs.Client().Get(hs.URL + tc.path)
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
		if tc.want == nil {
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
	}
}
`
