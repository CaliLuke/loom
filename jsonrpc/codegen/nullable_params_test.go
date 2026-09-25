package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCNullableParams asserts that the JSON-RPC server passes the params
// of a nullable object or an Any payload attribute selected with Body to the
// payload constructor by value, and that the client omits params for an
// absent optional attribute while the server decodes absent params as an
// absent value instead of substituting {}. A required attribute is always
// encoded.
func TestJSONRPCNullableParams(t *testing.T) {
	cases := []struct {
		Name string
		// Guard is the client condition that sends the params.
		Guard string
	}{
		{Name: "object", Guard: "p.O.Present()"},
		{Name: "any", Guard: "p.A != nil"},
	}
	for _, c := range cases {
		for _, required := range []bool{false, true} {
			name := c.Name + "/optional"
			if required {
				name = c.Name + "/required"
			}
			t.Run(name, func(t *testing.T) {
				root := RunJSONRPCDSL(t, jsonrpcNullableParamsDSL(required))
				services := CreateJSONRPCServices(root)
				method := "PickObject"
				if c.Name == "any" {
					method = "PickAny"
				}
				data := services.Get("Picker").Endpoint(method)
				require.NotNil(t, data)
				assert.Equal(t, !required, data.Payload.Request.OptionalBodyAttribute)

				encoder := jsonrpcSectionsSource(requireEncodeDecodeFile(t, ClientFiles("", services), "client"), "jsonrpc-request-encoder")
				decoder := jsonrpcSectionsSource(requireEncodeDecodeFile(t, ServerFiles("", services), "server"), "jsonrpc-request-decoder")
				assert.Contains(t, decoder, "payload = New"+method+"Payload(body, ")
				assert.NotContains(t, decoder, "Payload(&body")
				if required {
					assert.NotContains(t, encoder, c.Guard)
					assert.Contains(t, decoder, "params = []byte(\"{}\")")
					return
				}
				assert.Contains(t, encoder, "\t\tif "+c.Guard+" {\n\t\t\tbody.Params = ")
				assert.NotContains(t, decoder, "params = []byte(\"{}\")")
			})
		}
	}
}

// TestJSONRPCNullableParamsGeneratedModule compiles and vets a JSON-RPC
// service whose params are an optional nullable object or an optional Any
// payload attribute selected with Body, round-trips absent and concrete
// values through the generated client and server, checks that an absent
// value is sent without params, and sends absent, null, concrete and invalid
// params to the generated server.
func TestJSONRPCNullableParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcNullableParamsDSL(false))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcnull", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nullable_params_test.go"), []byte(jsonRPCNullableParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// jsonrpcSectionsSource returns the source of the sections of file named
// name, one per method.
func jsonrpcSectionsSource(file *codegen.File, name string) string {
	var source strings.Builder
	for _, section := range file.AllSections() {
		if section.SectionName() == name {
			source.WriteString(renderSectionSource(section))
		}
	}
	return source.String()
}

// jsonrpcNullableParamsDSL returns a JSON-RPC design whose methods map a
// nullable object and an Any payload attribute to the params with Body. The
// attributes are required when required is true.
func jsonrpcNullableParamsDSL(required bool) func() {
	return func() {
		dsl.API("nullparams", func() {
			dsl.JSONRPC(func() {})
		})
		leaf := dsl.Type("Leaf", func() {
			dsl.Attribute("name", dsl.String)
			dsl.Required("name")
		})
		dsl.Service("Picker", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("PickObject", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
					dsl.Attribute("o", leaf, func() {
						dsl.Nullable()
					})
					if required {
						dsl.Required("o")
					}
				})
				dsl.JSONRPC(func() {
					dsl.Body("o")
				})
			})
			dsl.Method("PickAny", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
					dsl.Attribute("a", dsl.Any)
					if required {
						dsl.Required("a")
					}
				})
				dsl.JSONRPC(func() {
					dsl.Body("a")
				})
			})
		})
	}
}

const jsonRPCNullableParamsHarness = `package jsonrpcnull_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcnull/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcnull/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcnull/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type service struct {
	mu   sync.Mutex
	seen []string
}

func (s *service) PickObject(_ context.Context, p *picker.PickObjectPayload) error {
	s.record(state(p.O))
	return nil
}

func (s *service) PickAny(_ context.Context, p *picker.PickAnyPayload) error {
	if p.A == nil {
		s.record("absent")
		return nil
	}
	s.record(string(p.A))
	return nil
}

func (s *service) record(v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, v)
}

func (s *service) take() []string {
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

func TestNullableParams(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(picker.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []struct {
		name   string
		call   func() error
		want   string
		params string
	}{
		{"absent object", func() error {
			_, err := c.PickObject()(context.Background(), &picker.PickObjectPayload{ID: ptr("1")})
			return err
		}, "absent", ""},
		{"concrete object", func() error {
			_, err := c.PickObject()(context.Background(), &picker.PickObjectPayload{ID: ptr("2"), O: loom.NullableValue(picker.Leaf{Name: "a"})})
			return err
		}, ` + "`" + `{"name":"a"}` + "`" + `, ` + "`" + `{"name":"a"}` + "`" + `},
		{"absent any", func() error {
			_, err := c.PickAny()(context.Background(), &picker.PickAnyPayload{ID: ptr("3")})
			return err
		}, "absent", ""},
		{"concrete any", func() error {
			_, err := c.PickAny()(context.Background(), &picker.PickAnyPayload{ID: ptr("4"), A: loom.JSONValue("[1,2.50]")})
			return err
		}, "[1,2.50]", "[1,2.50]"},
	}
	for _, tc := range payloads {
		if err := tc.call(); err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if seen := svc.take(); len(seen) != 1 || seen[0] != tc.want {
			t.Errorf("%s: server received %v, want %s", tc.name, seen, tc.want)
		}
		if got := string(sentParams(t, wire.take())); got != tc.params {
			t.Errorf("%s: sent params %q, want %q", tc.name, got, tc.params)
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
		{"absent object params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"PickObject"}` + "`" + `, 0, "", "", "absent"},
		{"null object params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"PickObject","params":null}` + "`" + `, 0, "", "", "null"},
		{"concrete object params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"PickObject","params":{"name":"b"}}` + "`" + `, 0, "", "", ` + "`" + `{"name":"b"}` + "`" + `},
		{"invalid object params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"PickObject","params":{}}` + "`" + `, -32602, "Missing required field: name", "missing_field", ""},
		{"absent any params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"PickAny"}` + "`" + `, 0, "", "", "absent"},
		{"null any params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"PickAny","params":null}` + "`" + `, 0, "", "", "null"},
		{"concrete any params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"PickAny","params":{"k":1}}` + "`" + `, 0, "", "", ` + "`" + `{"k":1}` + "`" + `},
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
			if len(seen) != 1 || seen[0] != tc.want {
				t.Errorf("%s: server received %v, want one call with %s", tc.name, seen, tc.want)
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
			t.Errorf("%s: service invoked with %v", tc.name, seen)
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
