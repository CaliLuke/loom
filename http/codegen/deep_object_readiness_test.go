package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestDeepObjectReadinessGeneratedWire(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	t.Setenv("LOOM_DIR", repoRoot)
	root := RunHTTPDSL(t, func() {
		filter := Type("Filter", MapOf(String, String))
		address := Type("Address", func() {
			Attribute("street", String, func() {
				Meta("struct:tag:json", "json_street")
				Meta("struct:tag:form", "form_street")
			})
		})
		Service("shape", func() {
			Method("formRoot", func() {
				Payload(MapOf(String, MapOf(String, String)))
				HTTP(func() {
					POST("/form-root")
					FormRequest()
				})
			})
			Method("root", func() {
				Payload(MapOf(String, String))
				HTTP(func() {
					GET("/root")
					MapParams()
				})
			})

			Method("query", func() {
				Payload(func() {
					Attribute("filter", filter)
					Attribute("many", MapOf(String, ArrayOf(String)))
					Attribute("tree", MapOf(String, MapOf(String, String)))
					Attribute("page", Int)
					Attribute("tags", ArrayOf(String))
					Required("page")
				})
				HTTP(func() {
					GET("/query")
					Param("filter")
					Param("many")
					Param("tree")
					Param("page")
					Param("tags")
				})
			})
			Method("form", func() {
				Payload(func() {
					Attribute("address", address)
					Attribute("labels", MapOf(String, String))
					Attribute("many", MapOf(String, ArrayOf(String)))
					Attribute("tree", MapOf(String, MapOf(String, String)))
					Attribute("tags", ArrayOf(String))
				})
				HTTP(func() {
					POST("/form")
					FormRequest()
				})
			})
		})
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/shape", root)
	runGoCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012215933-22e452f21fb3")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(deepObjectReadinessHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-v", "./...")
}

const deepObjectReadinessHarness = `package shape_test

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	client "example.com/shape/gen/http/shape/client"
	server "example.com/shape/gen/http/shape/server"
	service "example.com/shape/gen/shape"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootQueryWire(t *testing.T) {
	payload := map[string]string{"first": "one", "second": "two"}
	request := httptest.NewRequest("GET", "/root", nil)
	require.NoError(t, client.EncodeRootRequest(nil)(request, payload))
	require.Equal(t, "first=one&second=two", request.URL.RawQuery)
	decoded, err := server.DecodeRootRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	require.Equal(t, payload, decoded)
	var received map[string]string
	mux := loomhttp.NewMuxer()
	endpoints := &service.Endpoints{Root: func(_ context.Context, value any) (any, error) {
		received = value.(map[string]string)
		return nil, nil
	}}
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, 204, response.Code, response.Body.String())
	require.Equal(t, payload, received)
}

func TestRootFormJSONChild(t *testing.T) {
	payload := map[string]map[string]string{"": {"key": "value"}}
	request := httptest.NewRequest("POST", "/form-root", nil)
	require.NoError(t, client.EncodeFormRootRequest(nil)(request, payload))
	reader, err := request.GetBody()
	require.NoError(t, err)
	wire, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	want := url.Values{"": {"{\"key\":\"value\"}"}}.Encode()
	require.Equal(t, want, string(wire))
	request.Body = io.NopCloser(strings.NewReader(want))
	request.ContentLength = int64(len(want))
	actual, err := server.DecodeFormRootRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)(request)
	require.NoError(t, err)
	require.Equal(t, payload, actual)
	invalid := url.Values{"": {"{\"key\":null}"}}.Encode()
	request = httptest.NewRequest("POST", "/form-root", strings.NewReader(invalid))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = server.DecodeFormRootRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)(request)
	require.Error(t, err)
}

func TestQueryWire(t *testing.T) {
	payload := &service.QueryPayload{
		Filter: map[string]string{"space key": "a+b&c", "": "empty key"},
		Many:   map[string][]string{"k": {"first", "second"}},
		Tree:   map[string]map[string]string{"branch": {"leaf": "value"}},
		Page:   2, Tags: []string{"a", "b"},
	}
	req := httptest.NewRequest("GET", "/query", nil)
	require.NoError(t, client.EncodeQueryRequest(nil)(req, payload))
	want := url.Values{
		"filter": {"{\"\":\"empty key\",\"space key\":\"a+b&c\"}"},
		"many":   {"{\"k\":[\"first\",\"second\"]}"}, "tree": {"{\"branch\":{\"leaf\":\"value\"}}"},
		"page": {"2"}, "tags": {"a", "b"},
	}
	assert.Equal(t, want.Encode(), req.URL.RawQuery)
	req.URL.RawQuery = want.Encode() // Decode an independent external JSON request.
	got, err := server.DecodeQueryRequest(loomhttp.NewMuxer(), nil)(req)
	require.NoError(t, err)
	require.Equal(t, payload, got)
	t.Logf("query wire: %s", req.URL.RawQuery)
	flattened := httptest.NewRequest("GET", "/query?page=2&space+key=a%2Bb%26c", nil)
	decoded, err := server.DecodeQueryRequest(loomhttp.NewMuxer(), nil)(flattened)
	require.NoError(t, err)
	require.Empty(t, decoded.Filter, "flattened map keys are not the named JSON field")
}

func TestFormWire(t *testing.T) {
	street := "Main Street"
	payload := &service.FormPayload{Address: &service.Address{Street: &street}, Labels: map[string]string{"key": "value"}, Many: map[string][]string{"key": {"value"}}, Tree: map[string]map[string]string{"key": {"leaf": "value"}}, Tags: []string{"one", "two"}}
	req := httptest.NewRequest("POST", "/form", nil)
	require.NoError(t, client.EncodeFormRequest(nil)(req, payload))
	reader, err := req.GetBody()
	require.NoError(t, err)
	wire, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	want := url.Values{"address": {"{\"json_street\":\"Main Street\"}"}, "labels": {"{\"key\":\"value\"}"}, "many": {"{\"key\":[\"value\"]}"}, "tree": {"{\"key\":{\"leaf\":\"value\"}}"}, "tags": {"one", "two"}}.Encode()
	assert.Equal(t, want, string(wire))
	req.Body = io.NopCloser(strings.NewReader(want))
	req.ContentLength = int64(len(want))
	got, err := server.DecodeFormRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)(req)
	require.NoError(t, err)
	require.Equal(t, payload, got)
	t.Logf("form wire: %s", wire)
}

func TestMountedJSONFields(t *testing.T) {
	for _, kind := range []string{"query", "form"} {
		for _, raw := range []string{"", " ", "null", " \nnull\t", "{", "[]", "{} {}", "{\"key\":\"value\"}"} {
			t.Run(kind+raw, func(t *testing.T) {
				calls := 0
				var received any
				mux := loomhttp.NewMuxer()
				endpoints := &service.Endpoints{
					Query: func(_ context.Context, p any) (any, error) {
						calls++
						received = p
						return nil, nil
					},
					Form: func(_ context.Context, p any) (any, error) {
						calls++
						received = p
						return nil, nil
					},
				}
				server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				field := "filter"
				values := url.Values{field: {raw}, "page": {"2"}}
				req := httptest.NewRequest("GET", "/query?"+values.Encode(), nil)
				if kind == "form" {
					field = "labels"
					req = httptest.NewRequest("POST", "/form", strings.NewReader(url.Values{field: {raw}}.Encode()))
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, req)
				if raw == "{\"key\":\"value\"}" {
					require.Equal(t, 204, response.Code, response.Body.String())
					require.Equal(t, 1, calls)
					if kind == "query" {
						require.Equal(t, service.Filter{"key": "value"}, received.(*service.QueryPayload).Filter)
					} else {
						require.Equal(t, map[string]string{"key": "value"}, received.(*service.FormPayload).Labels)
					}
					return
				}
				require.Equal(t, 400, response.Code, response.Body.String())
				require.Zero(t, calls)
				require.Nil(t, received)
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, "decode_payload", problem.Code)
				require.Equal(t, "invalid request body", problem.Detail)
			})
		}
	}
}

func TestNestedNullJSONFields(t *testing.T) {
	for _, test := range []struct{ method, path, field, raw string }{
		{"GET", "/query", "filter", "{\"key\":null}"},
		{"GET", "/query", "many", "{\"key\":null}"},
		{"GET", "/query", "many", "{\"key\":[null]}"},
		{"GET", "/query", "tree", "{\"key\":null}"},
		{"GET", "/query", "tree", "{\"key\":{\"leaf\":null}}"},
		{"POST", "/form", "labels", "{\"key\":null}"},
		{"POST", "/form", "many", "{\"key\":[null]}"},
		{"POST", "/form", "tree", "{\"key\":{\"leaf\":null}}"},
		{"POST", "/form", "address", "{\"json_street\":null}"},
	} {
		t.Run(test.field+test.raw, func(t *testing.T) {
			calls := 0
			endpoint := func(context.Context, any) (any, error) {
				calls++
				return nil, nil
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(&service.Endpoints{Query: endpoint, Form: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			values := url.Values{test.field: {test.raw}, "page": {"2"}}
			request := httptest.NewRequest(test.method, test.path+"?"+values.Encode(), nil)
			if test.method == "POST" {
				request = httptest.NewRequest(test.method, test.path, strings.NewReader(values.Encode()))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, 400, response.Code, response.Body.String())
			require.Zero(t, calls)
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			require.Equal(t, "decode_payload", problem.Code)
			require.Equal(t, "invalid request body", problem.Detail)
		})
	}
}

func TestMalformedJSONFields(t *testing.T) {
	for _, raw := range []string{"{", "[]", "{} {}"} {
		t.Run(raw, func(t *testing.T) {
			query := url.Values{"filter": {raw}, "page": {"2"}}
			req := httptest.NewRequest("GET", "/query?"+query.Encode(), nil)
			_, err := server.DecodeQueryRequest(loomhttp.NewMuxer(), nil)(req)
			assert.Error(t, err, "invalid named query JSON must not be ignored")
			form := url.Values{"labels": {raw}}
			req = httptest.NewRequest("POST", "/form", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			_, err = server.DecodeFormRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)(req)
			assert.Error(t, err, "invalid named form JSON must not be ignored")
		})
	}
}
`
