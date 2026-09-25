package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCNullableUnionParams asserts that the JSON-RPC server passes the
// loom.Nullable params of a nullable union payload attribute selected with
// Body to the payload constructor by value, and that the client omits params
// for an absent optional attribute while the server decodes absent params as
// an absent value instead of substituting {}. A required attribute is always
// encoded.
func TestJSONRPCNullableUnionParams(t *testing.T) {
	for _, required := range []bool{false, true} {
		name := "optional"
		if required {
			name = "required"
		}
		t.Run(name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, jsonrpcNullableUnionParamsDSL(required))
			services := CreateJSONRPCServices(root)
			data := services.Get("Picker").Endpoint("Pick")
			require.NotNil(t, data)
			assert.Equal(t, !required, data.Payload.Request.OptionalBodyAttribute)
			assert.Equal(t, !required, data.Payload.Request.OptionalBodyNullable)

			encoder := sectionSourceByName(t, requireEncodeDecodeFile(t, ClientFiles("", services), "client"), "jsonrpc-request-encoder")
			decoder := sectionSourceByName(t, requireEncodeDecodeFile(t, ServerFiles("", services), "server"), "jsonrpc-request-decoder")
			assert.Contains(t, decoder, "payload = NewPickPayload(body, ")
			assert.NotContains(t, decoder, "NewPickPayload(&body")
			if required {
				assert.NotContains(t, encoder, "p.U.Present()")
				assert.Contains(t, decoder, "params = []byte(\"{}\")")
				return
			}
			assert.Contains(t, encoder, "\t\tif p.U.Present() {\n\t\t\tbody.Params = NewLoomNullablePickRequestBody(p)\n\t\t}\n")
			assert.NotContains(t, decoder, "params = []byte(\"{}\")")
		})
	}
}

// TestJSONRPCNullableUnionParamsGeneratedModule compiles and vets a JSON-RPC
// service whose params are an optional nullable union payload attribute
// selected with Body, round-trips absent and concrete unions through
// the generated client and server, checks that an absent union is sent
// without params, and sends absent, null, concrete, empty-object and invalid
// params to the generated server. The client cannot send a null union: the
// JSON-RPC envelope omits null params, which JSON-RPC 2.0 does not allow.
func TestJSONRPCNullableUnionParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcNullableUnionParamsDSL(false))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcnullunion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullable_union_params_test.go"), []byte(jsonRPCNullableUnionParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// jsonrpcNullableUnionParamsDSL returns a JSON-RPC design whose method maps a
// nullable constructor OneOf union payload attribute to the params with Body.
// The attribute is required when required is true.
func jsonrpcNullableUnionParamsDSL(required bool) func() {
	return func() {
		dsl.API("nullunion", func() {
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
					dsl.Attribute("u", dsl.OneOf(leaf, other), func() {
						dsl.Nullable()
					})
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

const jsonRPCNullableUnionParamsHarness = `package jsonrpcnullunion_test

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
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcnullunion/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcnullunion/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcnullunion/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
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

// state describes an absent, null or concrete nullable value.
func state[T any](n loom.Nullable[T]) string {
	switch {
	case !n.Present():
		return "absent"
	case n.IsNull():
		return "null"
	}
	v, _ := n.Value()
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("marshal error: %v", err)
	}
	return string(b)
}

func TestNullableUnionParams(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(picker.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []struct {
		p      picker.PickPayload
		params string
	}{
		{picker.PickPayload{ID: ptr("1")}, ""},
		{picker.PickPayload{ID: ptr("3"), U: loom.NullableValue(picker.NewLeafOrOtherLeaf(&picker.Leaf{Name: "a"}))}, ` + "`" + `{"type":"Leaf","value":{"name":"a"}}` + "`" + `},
	}
	for _, tc := range payloads {
		if _, err := c.Pick()(context.Background(), &tc.p); err != nil {
			t.Errorf("pick %s: %v", state(tc.p.U), err)
			continue
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(seen[0].U, tc.p.U) {
			t.Errorf("pick %s: server received %+v, want union %s", state(tc.p.U), seen, state(tc.p.U))
		}
		params := sentParams(t, wire.take())
		if got := string(params); got != tc.params {
			t.Errorf("pick %s: sent params %q, want %q", state(tc.p.U), got, tc.params)
		}
	}

	cases := []struct {
		name    string
		body    string
		code    int
		message string
		errName string
		want    string
	}{
		{"absent params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick"}` + "`" + `, 0, "", "", "absent"},
		{"null params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":null}` + "`" + `, 0, "", "", "null"},
		{"concrete params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":{"type":"Other","value":{}}}` + "`" + `, 0, "", "", state(loom.NullableValue(picker.NewLeafOrOtherOther(&picker.Other{})))},
		{"empty object params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":{}}` + "`" + `, -32602, ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, "invalid_enum_value", ""},
		{"invalid branch params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":{"type":"Leaf","value":{}}}` + "`" + `, -32602, "Missing required field: name", "missing_field", ""},
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
			if len(seen) != 1 || state(seen[0].U) != tc.want {
				t.Errorf("%s: server received %+v, want one call with union %s", tc.name, seen, tc.want)
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
