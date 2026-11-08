package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestJSONQueryCarrierDefaults compiles and exercises authored defaults of
// named JSON query values. Defaults apply only to an absent query key and are
// materialized in the native parameter's layout, including service, custom
// package and custom text declarations.
func TestJSONQueryCarrierDefaults(t *testing.T) {
	root := RunHTTPDSL(t, jsonQueryDefaultsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/carrier", root)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "label"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "label", "label.go"), []byte(jsonQueryLabelSource), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(jsonQueryDefaultsHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	t.Log(runCommand(t, dir, "go", "test", "-count=1", "-v", "./..."))
}

func jsonQueryDefaultsDSL() {
	cat, _ := jsonQueryPets()
	packagedCat := Type("PackagedCat", func() {
		Meta("struct:pkg:path", "types")
		Attribute("kind", String, func() {
			Enum("cat")
		})
		Attribute("name", String)
		Required("kind", "name")
	})
	label := func() {
		Meta("struct:field:type", "label.Label", "example.com/carrier/label")
	}
	filter := Type("Filter", func() {
		Attribute("name", String)
		Attribute("tag", String, func() {
			label()
			Default("oops")
		})
	})
	defaultCats := map[string]any{"a": map[string]any{"kind": "cat", "name": "Tom"}}
	Service("carrier", func() {
		Method("defaults", func() {
			Payload(func() {
				Attribute("cats", MapOf(String, cat), func() {
					Default(defaultCats)
				})
				Attribute("packaged", MapOf(String, packagedCat), func() {
					Default(map[string]any{"p": map[string]any{"kind": "cat", "name": "Pat"}})
				})
				Attribute("labels", MapOf(String, String), func() {
					Default(map[string]any{"k": "v"})
				})
				Attribute("counts", MapOf(String, MapOf(String, Int)), func() {
					Default(map[string]any{"a": map[string]any{"zero": 0, "one": 1}})
				})
				Attribute("tokens", MapOf(String, String), func() {
					Elem(label)
					Default(map[string]any{"a": "label:x"})
				})
				Attribute("values", MapOf(String, ArrayOf(String)), func() {
					Meta("struct:field:type", "url.Values", "net/url")
				})
			})
			HTTP(func() {
				GET("/defaults")
				Param("cats")
				Param("packaged")
				Param("labels")
				Param("counts")
				Param("tokens")
				Param("values")
				Response(StatusNoContent)
			})
		})
		Method("required", func() {
			Payload(func() {
				Attribute("cats", MapOf(String, cat), func() {
					Default(defaultCats)
				})
				Required("cats")
			})
			HTTP(func() {
				GET("/required")
				Param("cats")
				Response(StatusNoContent)
			})
		})
		Method("invalid", func() {
			Payload(func() {
				Attribute("page", Int)
				Attribute("tokens", MapOf(String, String), func() {
					Elem(label)
					Default(map[string]any{"a": "oops"})
				})
				Required("page")
			})
			HTTP(func() {
				GET("/invalid")
				Param("page")
				Param("tokens")
				Response(StatusNoContent)
			})
		})
		Method("styled", func() {
			Payload(func() {
				Attribute("tags", ArrayOf(String), func() {
					Elem(label)
					Default([]any{"oops"})
				})
			})
			HTTP(func() {
				GET("/styled")
				Param("tags", func() {
					Meta("openapi:style", "pipeDelimited")
				})
				Response(StatusNoContent)
			})
		})
		Method("member", func() {
			Payload(func() {
				Attribute("filter", filter)
			})
			HTTP(func() {
				GET("/member")
				Param("filter", func() {
					Meta("openapi:style", "deepObject")
				})
				Response(StatusNoContent)
			})
		})
		Method("mixed", func() {
			Payload(func() {
				Attribute("cat", cat)
				Attribute("cats", MapOf(String, cat), func() {
					Default(defaultCats)
				})
			})
			HTTP(func() {
				POST("/mixed")
				Param("cats")
				Body(func() {
					Attribute("cat")
				})
				Response(StatusNoContent)
			})
		})
	})
}

const jsonQueryLabelSource = `package label

import (
	"fmt"
	"strings"
)

// Label is a custom text value whose wire form starts with "label:".
type Label struct {
	// Text is the parsed label value.
	Text string
}

// MarshalText returns the label's wire representation.
func (l Label) MarshalText() ([]byte, error) {
	return []byte("label:" + l.Text), nil
}

// UnmarshalText parses a label.
func (l *Label) UnmarshalText(b []byte) error {
	if !strings.HasPrefix(string(b), "label:") {
		return fmt.Errorf("invalid label %q", b)
	}
	l.Text = strings.TrimPrefix(string(b), "label:")
	return nil
}
`

const jsonQueryDefaultsHarness = `package carrier_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"flag"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	service "example.com/carrier/gen/carrier"
	cli "example.com/carrier/gen/http/cli/test_api"
	client "example.com/carrier/gen/http/carrier/client"
	server "example.com/carrier/gen/http/carrier/server"
	types "example.com/carrier/gen/types"
	"example.com/carrier/label"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaults() *service.DefaultsPayload {
	return &service.DefaultsPayload{
		Cats:     map[string]*service.Cat{"a": {Kind: "cat", Name: "Tom"}},
		Packaged: map[string]*types.PackagedCat{"p": {Kind: "cat", Name: "Pat"}},
		Labels:   map[string]string{"k": "v"},
		Counts:   map[string]map[string]int{"a": {"zero": 0, "one": 1}},
		Tokens:   map[string]label.Label{"a": {Text: "x"}},
	}
}

func mount(endpoint func(context.Context, any) (any, error)) loomhttp.ResolverMuxer {
	mux := loomhttp.NewMuxer()
	endpoints := &service.Endpoints{Defaults: endpoint, Required: endpoint, Invalid: endpoint, Styled: endpoint, Member: endpoint, Mixed: endpoint}
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	return mux
}

func TestAbsentQueryUsesDefaults(t *testing.T) {
	request := httptest.NewRequest("GET", "/defaults", nil)
	require.NoError(t, client.EncodeDefaultsRequest(nil)(request, &service.DefaultsPayload{}))
	assert.Empty(t, request.URL.RawQuery)
	decoded, err := server.DecodeDefaultsRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	assert.Equal(t, defaults(), decoded)
}

func TestSuppliedQueryOverridesDefaults(t *testing.T) {
	for _, test := range []struct {
		name  string
		query url.Values
		want  func(*service.DefaultsPayload)
	}{
		{"empty maps stay empty", url.Values{"cats": {"{}"}, "packaged": {"{}"}, "labels": {"{}"}, "counts": {"{}"}, "tokens": {"{}"}}, func(p *service.DefaultsPayload) {
			p.Cats, p.Packaged, p.Labels, p.Counts, p.Tokens = map[string]*service.Cat{}, map[string]*types.PackagedCat{}, map[string]string{}, map[string]map[string]int{}, map[string]label.Label{}
		}},
		{"supplied zero and empty string stay supplied", url.Values{"counts": {"{\"a\":{\"zero\":0}}"}, "labels": {"{\"k\":\"\"}"}}, func(p *service.DefaultsPayload) {
			p.Counts = map[string]map[string]int{"a": {"zero": 0}}
			p.Labels = map[string]string{"k": ""}
		}},
		{"supplied values replace defaults", url.Values{"cats": {"{\"b\":{\"kind\":\"cat\",\"name\":\"Bo\"}}"}, "tokens": {"{\"b\":\"label:y\"}"}}, func(p *service.DefaultsPayload) {
			p.Cats = map[string]*service.Cat{"b": {Kind: "cat", Name: "Bo"}}
			p.Tokens = map[string]label.Label{"b": {Text: "y"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/defaults?"+test.query.Encode(), nil)
			decoded, err := server.DecodeDefaultsRequest(loomhttp.NewMuxer(), nil)(request)
			require.NoError(t, err)
			want := defaults()
			test.want(want)
			assert.Equal(t, want, decoded)
		})
	}
}

func TestDefaultedQueryRoundTrip(t *testing.T) {
	payload := &service.DefaultsPayload{Counts: map[string]map[string]int{}, Labels: map[string]string{"k": "w"}}
	request := httptest.NewRequest("GET", "/defaults", nil)
	require.NoError(t, client.EncodeDefaultsRequest(nil)(request, payload))
	assert.Equal(t, url.Values{"counts": {"{}"}, "labels": {"{\"k\":\"w\"}"}}.Encode(), request.URL.RawQuery)
	decoded, err := server.DecodeDefaultsRequest(loomhttp.NewMuxer(), nil)(request)
	require.NoError(t, err)
	want := defaults()
	want.Counts, want.Labels = payload.Counts, payload.Labels
	assert.Equal(t, want, decoded)
}

func TestMountedDefaults(t *testing.T) {
	var received any
	mux := mount(func(_ context.Context, payload any) (any, error) {
		received = payload
		return nil, nil
	})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/defaults", nil))
	require.Equal(t, 204, response.Code, response.Body.String())
	assert.Equal(t, defaults(), received)

	body, err := json.Marshal(map[string]any{"cat": map[string]any{"kind": "cat", "name": "Bo"}})
	require.NoError(t, err)
	request := httptest.NewRequest("POST", "/mixed", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, 204, response.Code, response.Body.String())
	assert.Equal(t, &service.MixedPayload{Cat: &service.Cat{Kind: "cat", Name: "Bo"}, Cats: defaults().Cats}, received)
}

func TestRejectedDefaultedQueries(t *testing.T) {
	for _, test := range []struct {
		target, code, detail string
	}{
		{"/defaults?" + url.Values{"cats": {"null"}}.Encode(), "decode_payload", ""},
		{"/defaults?" + url.Values{"labels": {"null"}}.Encode(), "decode_payload", ""},
		{"/defaults?" + url.Values{"labels": {"{\"k\":null}"}}.Encode(), "decode_payload", ""},
		{"/defaults?" + url.Values{"values": {"null"}}.Encode(), "decode_payload", ""},
		{"/defaults?" + url.Values{"values": {"{\"k\":[null]}"}}.Encode(), "decode_payload", ""},
		{"/defaults?" + url.Values{"cats": {"{\"a\":{\"kind\":\"cat\"}}"}}.Encode(), "missing_field", ""},
		{"/defaults?" + url.Values{"tokens": {"{\"a\":\"oops\"}"}}.Encode(), "decode_payload", ""},
		{"/required", "missing_field", ""},
		{"/invalid?" + url.Values{"tokens": {"{\"a\":\"label:y\"}"}}.Encode(), "missing_field", ""},
	} {
		t.Run(test.target, func(t *testing.T) {
			calls := 0
			mux := mount(func(context.Context, any) (any, error) {
				calls++
				return nil, nil
			})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest("GET", test.target, nil))
			require.Equal(t, 400, response.Code, response.Body.String())
			require.Zero(t, calls)
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			assert.Equal(t, test.code, problem.Code, response.Body.String())
			assert.Contains(t, response.Body.String(), test.detail)
		})
	}
}

// A server-authored default that cannot be materialized is a server fault on
// every parameter path. Only absence triggers it, and it outranks a client
// error found in the same request.
func TestFailedDefaultsAreServerFaults(t *testing.T) {
	for _, test := range []struct {
		target string
		status int
		code   string
	}{
		{"/invalid?page=1", 500, "fault"},
		{"/invalid", 500, "fault"},
		{"/invalid?page=1&" + url.Values{"tokens": {"{\"a\":\"label:y\"}"}}.Encode(), 204, ""},
		{"/styled", 500, "fault"},
		{"/styled?" + url.Values{"tags": {"label:a"}}.Encode(), 204, ""},
		{"/member?" + url.Values{"filter[name]": {"x"}}.Encode(), 500, "fault"},
		{"/member?" + url.Values{"filter[name]": {"x"}, "filter[tag]": {"label:t"}}.Encode(), 204, ""},
	} {
		t.Run(test.target, func(t *testing.T) {
			calls := 0
			mux := mount(func(context.Context, any) (any, error) {
				calls++
				return nil, nil
			})
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest("GET", test.target, nil))
			require.Equal(t, test.status, response.Code, response.Body.String())
			if test.status == 204 {
				require.Equal(t, 1, calls)
				return
			}
			require.Zero(t, calls)
			var problem loomhttp.ProblemResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			assert.Equal(t, test.code, problem.Code, response.Body.String())
			assert.True(t, problem.Fault, response.Body.String())
		})
	}
}

func TestCLIDefaults(t *testing.T) {
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	for _, test := range []struct {
		name  string
		args  []string
		want  func(*service.DefaultsPayload)
		valid bool
	}{
		{"omitted flags use defaults", nil, func(*service.DefaultsPayload) {}, true},
		{"empty map stays empty", []string{"--cats={}", "--counts={}"}, func(p *service.DefaultsPayload) {
			p.Cats, p.Counts = map[string]*service.Cat{}, map[string]map[string]int{}
		}, true},
		{"explicit value overrides", []string{"--cats={\"b\":{\"kind\":\"cat\",\"name\":\"Bo\"}}"}, func(p *service.DefaultsPayload) {
			p.Cats = map[string]*service.Cat{"b": {Kind: "cat", Name: "Bo"}}
		}, true},
		{"invalid member", []string{"--cats={\"b\":{\"kind\":\"cat\"}}"}, nil, false},
		{"null", []string{"--cats=null"}, nil, false},
		{"null scalar map", []string{"--labels=null"}, nil, false},
		{"null scalar map member", []string{"--labels={\"k\":null}"}, nil, false},
		{"null nested map member", []string{"--counts={\"a\":{\"zero\":null}}"}, nil, false},
		{"null custom element map member", []string{"--tokens={\"a\":null}"}, nil, false},
		{"null composite metadata map", []string{"--values=null"}, nil, false},
		{"null composite metadata map member", []string{"--values={\"k\":[null]}"}, nil, false},
		{"supplied zero and empty string", []string{"--counts={\"a\":{\"zero\":0}}", "--labels={\"k\":\"\"}"}, func(p *service.DefaultsPayload) {
			p.Counts = map[string]map[string]int{"a": {"zero": 0}}
			p.Labels = map[string]string{"k": ""}
		}, true},
		{"custom element value", []string{"--tokens={\"b\":\"label:y\"}"}, func(p *service.DefaultsPayload) {
			p.Tokens = map[string]label.Label{"b": {Text: "y"}}
		}, true},
		{"composite metadata map value", []string{"--values={\"k\":[\"v\"]}"}, func(p *service.DefaultsPayload) {
			p.Values = map[string][]string{"k": {"v"}}
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet("carrier", flag.ContinueOnError)
			require.NoError(t, flag.CommandLine.Parse(append([]string{"carrier", "defaults"}, test.args...)))
			_, payload, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			if !test.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			want := defaults()
			test.want(want)
			assert.Equal(t, want, payload)
		})
	}
}
`

// TestJSONQueryCarrierCLIParity requires each JSON query CLI flag to apply the
// server's decoding contract: maps without declarations, including a map whose
// composite Go metadata selects no emitted custom type, decode into the same
// physical carrier and validate it before native conversion.
func TestJSONQueryCarrierCLIParity(t *testing.T) {
	root := RunHTTPDSL(t, jsonQueryCLIParityDSL)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/carrier/gen", root)
	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dir, "gen", "http", "carrier", name))
		require.NoError(t, err)
		return string(data)
	}
	server, cli := read("server/encode_decode.go"), read("client/cli.go")
	from := func(code, marker string) string {
		index := strings.Index(code, marker)
		require.NotEqual(t, -1, index, "generated code contains %q", marker)
		return code[index:]
	}
	build := from(cli, "func BuildFlagsPayload(")
	for _, name := range []string{"labels", "tree", "values", "cats"} {
		serverCarrier := regexp.MustCompile(`var decoded (.+)\n\s*present, decodeErr := loomhttp\.DecodeJSONField\(qp, "` + name + `"`).FindStringSubmatch(server)
		require.Len(t, serverCarrier, 2, name)
		cliCarrier := regexp.MustCompile(`var decoded (.+)\n\s*err = loomhttpcli\.UnmarshalNonNullableJSON\(\[\]byte\(carrierFlags` + cg.Goify(name, true) + `\), &decoded\)`).FindStringSubmatch(build)
		require.Len(t, cliCarrier, 2, "%s: CLI decodes through a physical carrier", name)
		require.Contains(t, serverCarrier[1], "loom.Optional", name)
		if name != "cats" {
			require.Equal(t, serverCarrier[1], cliCarrier[1], "%s: non-declaring carriers are identical", name)
		}
	}
	for label, code := range map[string]string{"server": from(server, `"labels"`), "cli": from(build, "carrierFlagsLabels")} {
		validation, conversion := strings.Index(code, "InvalidLengthError"), strings.Index(code, "labels = make(")
		require.True(t, validation > 0 && conversion > 0 && validation < conversion, "%s validates the labels carrier before conversion", label)
	}
}

func jsonQueryCLIParityDSL() {
	cat, _ := jsonQueryPets()
	Service("carrier", func() {
		Method("flags", func() {
			Payload(func() {
				Attribute("labels", MapOf(String, String), func() {
					Elem(func() {
						MinLength(1)
					})
				})
				Attribute("tree", MapOf(String, MapOf(String, Int)))
				Attribute("values", MapOf(String, ArrayOf(String)), func() {
					Meta("struct:field:type", "url.Values", "net/url")
				})
				Attribute("cats", MapOf(String, cat))
			})
			HTTP(func() {
				GET("/flags")
				Param("labels")
				Param("tree")
				Param("values")
				Param("cats")
				Response(StatusNoContent)
			})
		})
	})
}
