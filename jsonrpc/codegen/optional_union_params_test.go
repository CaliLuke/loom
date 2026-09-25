package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCOptionalUnionParamsClientEncoder asserts that the JSON-RPC
// client request encoder omits params when the optional union payload
// attribute mapped to the params with Body is nil and that the server request
// decoder then decodes absent params as an empty body instead of {}. A
// required union attribute is still always encoded and absent params still
// decode as {}.
func TestJSONRPCOptionalUnionParamsClientEncoder(t *testing.T) {
	cases := []struct {
		Name     string
		Required bool
		Contains string
	}{
		{
			Name: "optional",
			Contains: "\t\tbody := &jsonrpc.Request{\n" +
				"\t\t\tJSONRPC: \"2.0\",\n" +
				"\t\t\tMethod:  \"Pick\",\n" +
				"\t\t}\n" +
				"\t\tif p.U != nil {\n" +
				"\t\t\tbody.Params = NewPickRequestBody(p)\n" +
				"\t\t}\n",
		},
		{
			Name:     "required",
			Required: true,
			Contains: "\t\tb := NewPickRequestBody(p)\n" +
				"\t\tbody := &jsonrpc.Request{\n" +
				"\t\t\tJSONRPC: \"2.0\",\n" +
				"\t\t\tMethod:  \"Pick\",\n" +
				"\t\t\tParams:  b,\n" +
				"\t\t}\n",
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, jsonrpcOptionalUnionParamsDSL(c.Required))
			services := CreateJSONRPCServices(root)
			data := services.Get("Picker").Endpoint("Pick")
			require.NotNil(t, data)
			assert.Equal(t, !c.Required, data.Payload.Request.OptionalBodyAttribute)

			file := requireEncodeDecodeFile(t, ClientFiles("", services), "client")
			code := sectionSourceByName(t, file, "jsonrpc-request-encoder")
			assert.Contains(t, code, c.Contains)
			assert.Contains(t, code, "if err := encoder(req).Encode(&body); err != nil {")
			if c.Required {
				assert.NotContains(t, code, "if p.U != nil {")
			}

			decoder := sectionSourceByName(t, requireEncodeDecodeFile(t, ServerFiles("", services), "server"), "jsonrpc-request-decoder")
			substitute := "\t\tif len(params) == 0 {\n\t\t\tparams = []byte(\"{}\")\n\t\t}\n"
			if c.Required {
				assert.Contains(t, decoder, substitute)
				return
			}
			assert.Contains(t, decoder, "\t\tparams := req.Params\n\t\tr.Body = io.NopCloser(bytes.NewReader(params))\n")
			assert.NotContains(t, decoder, substitute)
		})
	}
}

// TestJSONRPCOptionalUnionParamsGeneratedModule compiles and vets a JSON-RPC
// service whose params are an optional union payload attribute selected with
// Body, round-trips every branch and a nil union through the generated client
// and server, checks that a nil union is sent without params, and sends
// absent, null and empty-object params to the generated server.
func TestJSONRPCOptionalUnionParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcOptionalUnionParamsDSL(false))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcoptunion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_union_params_test.go"), []byte(jsonRPCOptionalUnionParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// jsonrpcOptionalUnionParamsDSL returns a JSON-RPC design whose method maps a
// constructor OneOf union payload attribute to the params with Body. The
// attribute is required when required is true.
func jsonrpcOptionalUnionParamsDSL(required bool) func() {
	return func() {
		dsl.API("optunion", func() {
			dsl.JSONRPC(func() {})
		})
		leaf := dsl.Type("Leaf", func() {
			dsl.Attribute("name", dsl.String)
			dsl.Required("name")
		})
		other := dsl.Type("Other", func() {
			dsl.Attribute("count", dsl.Int)
		})
		dsl.Service("Picker", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("Pick", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
					dsl.Attribute("u", dsl.OneOf(leaf, other))
					if required {
						dsl.Required("u")
					}
				})
				dsl.JSONRPC(func() {
					dsl.Body("u")
				})
			})
		})
	}
}

const jsonRPCOptionalUnionParamsHarness = `package jsonrpcoptunion_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"encoding/json/jsontext"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcoptunion/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcoptunion/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcoptunion/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []*picker.PickPayload
}

func (s *service) Pick(_ context.Context, p *picker.PickPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
	return nil
}

func (s *service) take() []*picker.PickPayload {
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

func TestOptionalUnionParams(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(picker.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	unions := []picker.LeafOrOther{
		picker.NewLeafOrOtherLeaf(&picker.Leaf{Name: "a"}),
		picker.NewLeafOrOtherOther(&picker.Other{Count: ptr(3)}),
	}
	for _, u := range unions {
		p := picker.PickPayload{ID: ptr("1"), U: &u}
		if _, err := c.Pick()(context.Background(), &p); err != nil {
			t.Fatalf("pick %s: %v", u.Kind(), err)
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(seen[0].U, p.U) {
			t.Errorf("pick %s: server received %+v, want union %+v", u.Kind(), seen, u)
		}
		if params := sentParams(t, wire.take()); params == nil {
			t.Errorf("pick %s: request has no params", u.Kind())
		}
	}

	// A nil optional union is sent without params, and the server decodes
	// absent params as a nil union.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("pick nil union: client panicked: %v", r)
			}
		}()
		p := picker.PickPayload{ID: ptr("2")}
		if _, err := c.Pick()(context.Background(), &p); err != nil {
			t.Errorf("pick nil union: %v", err)
		}
		seen := svc.take()
		if len(seen) != 1 || seen[0].U != nil {
			t.Errorf("pick nil union: server received %+v, want one call with a nil union", seen)
		}
	}()
	if params := sentParams(t, wire.take()); params != nil {
		t.Errorf("pick nil union: sent params %s, want none", params)
	}

	cases := []struct {
		name    string
		body    string
		code    int
		message string
		errName string
	}{
		{"absent params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick"}` + "`" + `, 0, "", ""},
		{"null params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":null}` + "`" + `, -32602, ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, "invalid_enum_value"},
		{"empty object params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":{}}` + "`" + `, -32602, ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, "invalid_enum_value"},
	}
	for _, tc := range cases {
		resp, err := hs.Client().Post(hs.URL+"/rpc", "application/json", strings.NewReader(tc.body))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("%s: read: %v", tc.name, err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("%s: close: %v", tc.name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d, want 200 (%s)", tc.name, resp.StatusCode, raw)
		}
		var envelope struct {
			Error *struct {
				Code    int    ` + "`" + `json:"code"` + "`" + `
				Message string ` + "`" + `json:"message"` + "`" + `
				Data    struct {
					Name string ` + "`" + `json:"name"` + "`" + `
				} ` + "`" + `json:"data"` + "`" + `
			} ` + "`" + `json:"error"` + "`" + `
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Errorf("%s: decode response %q: %v", tc.name, raw, err)
			continue
		}
		seen := svc.take()
		if tc.code == 0 {
			if envelope.Error != nil {
				t.Errorf("%s: unexpected error %s", tc.name, raw)
			}
			if len(seen) != 1 || seen[0].U != nil {
				t.Errorf("%s: server received %+v, want one call with a nil union", tc.name, seen)
			}
			continue
		}
		switch {
		case envelope.Error == nil:
			t.Errorf("%s: no error in %s", tc.name, raw)
		case envelope.Error.Code != tc.code || envelope.Error.Message != tc.message || envelope.Error.Data.Name != tc.errName:
			t.Errorf("%s: error (%d, %q, %q), want (%d, %q, %q)", tc.name, envelope.Error.Code, envelope.Error.Message, envelope.Error.Data.Name, tc.code, tc.message, tc.errName)
		}
		if len(seen) != 0 {
			t.Errorf("%s: service invoked with %+v", tc.name, seen)
		}
	}
}

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
`
