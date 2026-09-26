package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCOptionalValueParams asserts that the JSON-RPC client omits
// params when the optional primitive, array, map or bytes payload attribute
// mapped to the params with Body is nil, and that the server then decodes
// absent params as a nil attribute instead of substituting {}. A primitive
// with a default value and a required attribute are always encoded, and the
// server still decodes their absent params as {}.
func TestJSONRPCOptionalValueParams(t *testing.T) {
	cases := []struct {
		Name      string
		Attribute func()
		Optional  bool
		// Decode lists fragments of the server request decoder of an
		// optional attribute.
		Decode []string
	}{
		{
			Name:      "string",
			Attribute: func() { dsl.Attribute("v", dsl.String, func() { dsl.MinLength(2) }) },
			Optional:  true,
			Decode: []string{
				"\t\t\tbody = new(string)\n",
				"\t\terr = decoder(r).Decode(body)\n",
				"\t\t\t\tbody = nil\n",
			},
		},
		{
			Name:      "array",
			Attribute: func() { dsl.Attribute("v", dsl.ArrayOf(dsl.String)) },
			Optional:  true,
			Decode:    []string{"\t\tif body != nil {\nfor i, e := range body {\n"},
		},
		{
			Name:      "map",
			Attribute: func() { dsl.Attribute("v", dsl.MapOf(dsl.String, dsl.Int)) },
			Optional:  true,
			Decode:    []string{"\t\tif body != nil {\nfor _, v := range body {\n"},
		},
		{
			Name:      "bytes",
			Attribute: func() { dsl.Attribute("v", dsl.Bytes) },
			Optional:  true,
		},
		{
			Name:      "default",
			Attribute: func() { dsl.Attribute("v", dsl.String, func() { dsl.Default("d") }) },
		},
	}
	substitute := "\t\tif len(params) == 0 {\n\t\t\tparams = []byte(\"{}\")\n\t\t}\n"
	for _, c := range cases {
		for _, required := range []bool{false, true} {
			name := c.Name + "/optional"
			if required {
				name = c.Name + "/required"
			}
			t.Run(name, func(t *testing.T) {
				root := RunJSONRPCDSL(t, jsonrpcOptionalValueParamsDSL(c.Attribute, required))
				services := CreateJSONRPCServices(root)
				data := services.Get("Picker").Endpoint("Send")
				require.NotNil(t, data)
				optional := c.Optional && !required
				assert.Equal(t, optional, data.Payload.Request.OptionalBodyAttribute)

				encoder := jsonrpcSectionsSource(requireEncodeDecodeFile(t, ClientFiles("", services), "client"), "jsonrpc-request-encoder")
				decoder := jsonrpcSectionsSource(requireEncodeDecodeFile(t, ServerFiles("", services), "server"), "jsonrpc-request-decoder")
				if !optional {
					assert.NotContains(t, encoder, "if p.V != nil {")
					assert.Contains(t, encoder, "\t\t\tParams:  b,\n")
					assert.Contains(t, decoder, substitute)
					assert.NotContains(t, decoder, "body = nil")
					return
				}
				assert.Contains(t, encoder, "\t\tif p.V != nil {\n\t\t\tbody.Params = p.V\n\t\t}\n")
				assert.NotContains(t, decoder, substitute)
				for _, want := range c.Decode {
					assert.Contains(t, decoder, want)
				}
			})
		}
	}
}

// TestJSONRPCOptionalValueParamsGeneratedModule compiles and vets a JSON-RPC
// service whose params are optional string, integer, array, map, bytes and
// string alias payload attributes or a defaulted string selected with Body. It round-trips nil and concrete
// values through the generated client and server, checks that a nil value
// is sent without params, and sends absent, null, concrete, empty, invalid
// and malformed params to the generated server.
func TestJSONRPCOptionalValueParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcOptionalValueParamsModuleDSL("v"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcoptvalue", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_value_params_test.go"), []byte(jsonRPCOptionalValueParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// jsonrpcOptionalValueParamsDSL returns a JSON-RPC design whose method maps
// the payload attribute "v" declared by attribute to the params with Body.
// The attribute is required when required is true.
func jsonrpcOptionalValueParamsDSL(attribute func(), required bool) func() {
	return func() {
		dsl.API("optvalue", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Service("Picker", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("Send", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
					attribute()
					if required {
						dsl.Required("v")
					}
				})
				dsl.JSONRPC(func() {
					dsl.Body("v")
				})
			})
		})
	}
}

// jsonrpcOptionalValueParamsModuleDSL returns a JSON-RPC design whose methods
// map optional string, integer, array, map, bytes and string alias payload
// attributes and a defaulted string to the params with Body("v"). The
// payload declares the attribute with the object key key, "v" or a key with
// an element name suffix such as "v:x".
func jsonrpcOptionalValueParamsModuleDSL(key string) func() {
	return func() {
		dsl.API("optvalue", func() {
			dsl.JSONRPC(func() {})
		})
		var Plain = dsl.Type("Plain", dsl.String)
		method := func(name string, attribute func()) {
			dsl.Method(name, func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
					attribute()
				})
				dsl.JSONRPC(func() {
					dsl.Body("v")
				})
			})
		}
		dsl.Service("Picker", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			method("Str", func() { dsl.Attribute(key, dsl.String, func() { dsl.MinLength(2) }) })
			method("Num", func() { dsl.Attribute(key, dsl.Int) })
			method("Strs", func() { dsl.Attribute(key, dsl.ArrayOf(dsl.String)) })
			method("Counts", func() { dsl.Attribute(key, dsl.MapOf(dsl.String, dsl.Int)) })
			method("Blob", func() { dsl.Attribute(key, dsl.Bytes) })
			method("Alias", func() { dsl.Attribute(key, Plain) })
			method("Dflt", func() { dsl.Attribute(key, dsl.String, func() { dsl.Default("d") }) })
		})
	}
}

const jsonRPCOptionalValueParamsHarness = `package jsonrpcoptvalue_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcoptvalue/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcoptvalue/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcoptvalue/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
)

// service records the payload of every call.
type service struct {
	mu   sync.Mutex
	seen []any
}

func (s *service) Str(_ context.Context, p *picker.StrPayload) error       { return s.record(p) }
func (s *service) Num(_ context.Context, p *picker.NumPayload) error       { return s.record(p) }
func (s *service) Strs(_ context.Context, p *picker.StrsPayload) error     { return s.record(p) }
func (s *service) Counts(_ context.Context, p *picker.CountsPayload) error { return s.record(p) }
func (s *service) Blob(_ context.Context, p *picker.BlobPayload) error     { return s.record(p) }
func (s *service) Alias(_ context.Context, p *picker.AliasPayload) error   { return s.record(p) }
func (s *service) Dflt(_ context.Context, p *picker.DfltPayload) error     { return s.record(p) }

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

// wireRecorder records the body of the last request before passing it on.
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

func ptr[T any](v T) *T { return &v }

// sentParams returns the raw params of the JSON-RPC request body, or nil when
// the request has no params member.
func sentParams(t *testing.T, body []byte) jsontext.Value {
	t.Helper()
	var envelope map[string]jsontext.Value
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode request %q: %v", body, err)
	}
	return envelope["params"]
}

func serve(t *testing.T) (*service, *wireRecorder, *httptest.Server) {
	t.Helper()
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(_ context.Context, _ http.ResponseWriter, err error) {
		t.Errorf("handler error: %v", err)
	}
	server.Mount(mux, server.New(picker.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	t.Cleanup(hs.Close)
	return svc, wire, hs
}

// TestClient round-trips nil and concrete values through the generated
// client and checks the params that it sends: none for a nil value.
func TestClient(t *testing.T) {
	svc, wire, hs := serve(t)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	cases := []struct {
		name    string
		call    func(context.Context, any) (any, error)
		payload any
		params  string
	}{
		{"Str", c.Str(), &picker.StrPayload{ID: ptr("1"), V: ptr("ab")}, "\"ab\""},
		{"Str nil", c.Str(), &picker.StrPayload{ID: ptr("2")}, ""},
		{"Num", c.Num(), &picker.NumPayload{ID: ptr("3"), V: ptr(0)}, "0"},
		{"Num nil", c.Num(), &picker.NumPayload{ID: ptr("4")}, ""},
		{"Strs", c.Strs(), &picker.StrsPayload{ID: ptr("5"), V: []string{"a"}}, "[\"a\"]"},
		{"Strs empty", c.Strs(), &picker.StrsPayload{ID: ptr("6"), V: []string{}}, "[]"},
		{"Strs nil", c.Strs(), &picker.StrsPayload{ID: ptr("7")}, ""},
		{"Counts", c.Counts(), &picker.CountsPayload{ID: ptr("8"), V: map[string]int{"k": 1}}, "{\"k\":1}"},
		{"Counts nil", c.Counts(), &picker.CountsPayload{ID: ptr("9")}, ""},
		{"Blob", c.Blob(), &picker.BlobPayload{ID: ptr("10"), V: []byte("ab")}, "\"YWI=\""},
		{"Blob nil", c.Blob(), &picker.BlobPayload{ID: ptr("11")}, ""},
		{"Alias", c.Alias(), &picker.AliasPayload{ID: ptr("12"), V: ptr(picker.Plain("x"))}, "\"x\""},
		{"Alias nil", c.Alias(), &picker.AliasPayload{ID: ptr("13")}, ""},
		{"Dflt", c.Dflt(), &picker.DfltPayload{ID: ptr("14"), V: "x"}, "\"x\""},
		{"Dflt empty", c.Dflt(), &picker.DfltPayload{ID: ptr("15")}, "\"\""},
	}
	for _, tc := range cases {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("client panicked: %v", r)
				}
			}()
			_, err = tc.call(context.Background(), tc.payload)
			return err
		}()
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], tc.payload) {
			t.Errorf("%s: server received %#v, want %#v", tc.name, seen, tc.payload)
		}
		if got := string(sentParams(t, wire.take())); got != tc.params {
			t.Errorf("%s: sent params %q, want %q", tc.name, got, tc.params)
		}
	}
}

// TestWire sends absent, null, concrete, empty, invalid and malformed params
// and checks the error or the value that the service receives.
func TestWire(t *testing.T) {
	svc, _, hs := serve(t)
	const invalid = "invalid request body"
	cases := []struct {
		method  string
		params  string
		code    int
		message string
		errName string
		want    any
	}{
		{"Str", "", 0, "", "", (*string)(nil)},
		{"Str", "\"ab\"", 0, "", "", ptr("ab")},
		{"Str", "\"a\"", -32602, "", "invalid_length", nil},
		{"Str", "null", -32602, "", "invalid_length", nil},
		{"Str", "{}", -32602, invalid, "decode_payload", nil},
		{"Str", "\"ab", -32700, "", "", nil},
		{"Num", "", 0, "", "", (*int)(nil)},
		{"Num", "0", 0, "", "", ptr(0)},
		{"Num", "null", 0, "", "", ptr(0)},
		{"Num", "\"x\"", -32602, invalid, "decode_payload", nil},
		{"Strs", "", 0, "", "", []string(nil)},
		{"Strs", "null", 0, "", "", []string(nil)},
		{"Strs", "[]", 0, "", "", []string{}},
		{"Strs", "[\"a\"]", 0, "", "", []string{"a"}},
		{"Strs", "{}", -32602, invalid, "decode_payload", nil},
		{"Counts", "", 0, "", "", map[string]int(nil)},
		{"Counts", "{}", 0, "", "", map[string]int{}},
		{"Counts", "{\"k\":1}", 0, "", "", map[string]int{"k": 1}},
		{"Blob", "", 0, "", "", []byte(nil)},
		{"Blob", "\"YWI=\"", 0, "", "", []byte("ab")},
		{"Blob", "\"!\"", -32602, invalid, "decode_payload", nil},
		{"Alias", "", 0, "", "", (*picker.Plain)(nil)},
		{"Alias", "\"x\"", 0, "", "", ptr(picker.Plain("x"))},
		{"Dflt", "\"x\"", 0, "", "", "x"},
	}
	for _, tc := range cases {
		name := tc.method + " " + tc.params
		body := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":" + strconv.Quote(tc.method)
		if tc.params == "" {
			name = tc.method + " absent"
		} else {
			body += ",\"params\":" + tc.params
		}
		body += "}"
		resp, err := hs.Client().Post(hs.URL+"/rpc", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("%s: read: %v", name, err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("%s: close: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d, want 200 (%s)", name, resp.StatusCode, raw)
		}
		var envelope struct {
			Result jsontext.Value
			Error  *struct {
				Code    int
				Message string
				Data    struct {
					Name string
				}
			}
		}
		if err := json.Unmarshal(raw, &envelope, json.MatchCaseInsensitiveNames(true)); err != nil {
			t.Errorf("%s: decode response %q: %v", name, raw, err)
			continue
		}
		seen := svc.take()
		if tc.code == 0 {
			if envelope.Error != nil || string(envelope.Result) != "null" {
				t.Errorf("%s: got %s, want a null result", name, raw)
			}
			if len(seen) != 1 {
				t.Errorf("%s: service invoked %d times, want 1", name, len(seen))
				continue
			}
			if got := reflect.ValueOf(seen[0]).Elem().FieldByName("V").Interface(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("%s: service received %#v, want %#v", name, got, tc.want)
			}
			continue
		}
		switch {
		case envelope.Error == nil:
			t.Errorf("%s: no error in %s", name, raw)
		case envelope.Error.Code != tc.code || tc.message != "" && envelope.Error.Message != tc.message || envelope.Error.Data.Name != tc.errName:
			t.Errorf("%s: error (%d, %q, %q), want (%d, %q, %q)", name, envelope.Error.Code, envelope.Error.Message, envelope.Error.Data.Name, tc.code, tc.message, tc.errName)
		}
		if len(seen) != 0 {
			t.Errorf("%s: service invoked with %#v", name, seen)
		}
	}
}
`
