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

// TestJSONRPCRequiredPresenceParams asserts that the JSON-RPC server does not
// replace absent params with {} when the params are a required nullable or
// Any payload attribute selected with Body, or a nullable or Any payload, so
// that absent params are a missing payload. The params of a required object
// without explicit presence still decode as {} when absent.
func TestJSONRPCRequiredPresenceParams(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcRequiredPresenceParamsDSL)
	services := CreateJSONRPCServices(root)
	decoders := requireEncodeDecodeFile(t, ServerFiles("", services), "server")
	substitute := "\t\tif len(params) == 0 {\n\t\t\tparams = []byte(\"{}\")\n\t\t}\n"
	missing := "\t\t\tif errors.Is(err, io.EOF) {\n\t\t\t\treturn payload, loom.MissingPayloadError()\n\t\t\t}\n"
	for _, c := range []struct {
		Method     string
		Substitute bool
	}{
		{"NullString", false},
		{"NullArray", false},
		{"NullMap", false},
		{"NullObject", false},
		{"NullOpen", false},
		{"AnyValue", false},
		{"NullUnion", false},
		{"PayloadAny", false},
		{"PayloadNullString", false},
		{"Object", true},
		{"Str", true},
		{"Arr", true},
	} {
		t.Run(c.Method, func(t *testing.T) {
			data := services.Get("Picker").Endpoint(c.Method)
			require.NotNil(t, data)
			assert.False(t, data.Payload.Request.OptionalBodyAttribute)
			assert.Equal(t, !c.Substitute, data.Payload.Request.ExplicitPresenceBody)
			decoder := requestDecoderSource(t, decoders, data.RequestDecoder)
			assert.Contains(t, decoder, missing)
			if c.Substitute {
				assert.Contains(t, decoder, substitute)
				return
			}
			assert.NotContains(t, decoder, substitute)
		})
	}
}

// TestJSONRPCRequiredPresenceParamsGeneratedModule compiles and vets a
// JSON-RPC service whose params are required nullable and Any payload
// attributes selected with Body or nullable and Any payloads. It sends
// absent, null, concrete and invalid params to the generated server and
// round-trips concrete values, including empty strings, arrays and objects,
// through the generated client, which omits params only when it has none.
func TestJSONRPCRequiredPresenceParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcRequiredPresenceParamsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcpresence", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_params_test.go"), []byte(jsonRPCRequiredPresenceParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// requestDecoderSource returns the source of the JSON-RPC request decoder
// named name in file.
func requestDecoderSource(t *testing.T, file *codegen.File, name string) string {
	t.Helper()
	for _, section := range file.AllSections() {
		if section.SectionName() != "jsonrpc-request-decoder" {
			continue
		}
		if source := renderSectionSource(section); strings.Contains(source, "func "+name+"(") {
			return source
		}
	}
	require.FailNowf(t, "missing request decoder", "expected %s in %s", name, file.Path)
	return ""
}

// jsonrpcRequiredPresenceParamsDSL is a JSON-RPC design whose methods map
// required nullable and Any payload attributes to the params with Body, take
// a nullable or Any payload, or, for the Object, Str and Arr methods, map a
// required object, string or array without explicit presence to the params.
// The Ping method has no payload.
func jsonrpcRequiredPresenceParamsDSL() {
	dsl.API("presence", func() {
		dsl.JSONRPC(func() {})
	})
	leaf := dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Attribute("count", dsl.Int)
	})
	open := dsl.Type("Open", func() {
		dsl.Attribute("note", dsl.String)
	})
	body := func(name string, typ any, nullable bool) {
		dsl.Method(name, func() {
			dsl.Payload(func() {
				dsl.ID("id", dsl.String)
				dsl.Attribute("v", typ, func() {
					if nullable {
						dsl.Nullable()
					}
				})
				dsl.Required("v")
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
		body("NullString", dsl.String, true)
		body("NullArray", dsl.ArrayOf(dsl.String), true)
		body("NullMap", dsl.MapOf(dsl.String, dsl.Int), true)
		body("NullObject", leaf, true)
		body("AnyValue", dsl.Any, false)
		body("NullUnion", dsl.OneOf(leaf, other), true)
		body("NullOpen", open, true)
		body("Object", leaf, false)
		body("Str", dsl.String, false)
		body("Arr", dsl.ArrayOf(dsl.String), false)
		dsl.Method("Ping", func() {
			dsl.JSONRPC(func() {})
		})
		dsl.Method("PayloadAny", func() {
			dsl.Payload(dsl.Any)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("PayloadNullString", func() {
			dsl.Payload(dsl.String, func() {
				dsl.Nullable()
			})
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCRequiredPresenceParamsHarness = `package jsonrpcpresence_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcpresence/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcpresence/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcpresence/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

// service records the state of the value that each method receives.
type service struct {
	mu   sync.Mutex
	seen []string
}

func (s *service) NullString(_ context.Context, p *picker.NullStringPayload) error {
	s.record(state(p.V))
	return nil
}

func (s *service) NullArray(_ context.Context, p *picker.NullArrayPayload) error {
	s.record(state(p.V))
	return nil
}

func (s *service) NullMap(_ context.Context, p *picker.NullMapPayload) error {
	s.record(state(p.V))
	return nil
}

func (s *service) NullObject(_ context.Context, p *picker.NullObjectPayload) error {
	s.record(state(p.V))
	return nil
}

func (s *service) AnyValue(_ context.Context, p *picker.AnyValuePayload) error {
	s.record(anyState(p.V))
	return nil
}

func (s *service) NullUnion(_ context.Context, p *picker.NullUnionPayload) error {
	s.record(state(p.V))
	return nil
}

func (s *service) NullOpen(_ context.Context, p *picker.NullOpenPayload) error {
	s.record(state(p.V))
	return nil
}

func (s *service) Str(_ context.Context, p *picker.StrPayload) error {
	s.record(marshal(p.V))
	return nil
}

func (s *service) Arr(_ context.Context, p *picker.ArrPayload) error {
	s.record(marshal(p.V))
	return nil
}

func (s *service) Ping(context.Context) error {
	s.record("ping")
	return nil
}

func (s *service) Object(_ context.Context, p *picker.ObjectPayload) error {
	if p.V == nil {
		s.record("absent")
		return nil
	}
	s.record(marshal(p.V))
	return nil
}

func (s *service) PayloadAny(_ context.Context, p loom.JSONValue) error {
	s.record(anyState(p))
	return nil
}

func (s *service) PayloadNullString(_ context.Context, p loom.Nullable[string]) error {
	s.record(state(p))
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

func marshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("marshal error: %v", err)
	}
	return string(b)
}

// state describes an absent, null or concrete nullable value.
func state[T any](n loom.Nullable[T]) string {
	switch {
	case !n.Present():
		return "absent"
	case n.IsNull():
		return "null"
	}
	v, _ := n.Value()
	return marshal(v)
}

// anyState describes an absent or concrete Any value.
func anyState(v loom.JSONValue) string {
	if v == nil {
		return "absent"
	}
	return string(v)
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

// TestWire sends absent, null, concrete and invalid params. Absent params
// are a missing payload for every method but Object, whose params are a
// required object without explicit presence and decode as {}.
func TestWire(t *testing.T) {
	svc, _, hs := serve(t)
	const (
		missing = "validation error"
		leaf    = "{\"type\":\"Leaf\",\"value\":{\"name\":\"a\"}}"
	)
	cases := []struct {
		method  string
		params  string
		code    int
		message string
		errName string
		want    string
	}{
		{"NullString", "", -32602, missing, "missing_payload", ""},
		{"NullString", "null", 0, "", "", "null"},
		{"NullString", "\"a\"", 0, "", "", "\"a\""},
		{"NullString", "{}", -32602, "", "decode_payload", ""},
		{"NullArray", "", -32602, missing, "missing_payload", ""},
		{"NullArray", "null", 0, "", "", "null"},
		{"NullArray", "[]", 0, "", "", "[]"},
		{"NullArray", "[\"a\"]", 0, "", "", "[\"a\"]"},
		{"NullMap", "", -32602, missing, "missing_payload", ""},
		{"NullMap", "null", 0, "", "", "null"},
		{"NullMap", "{}", 0, "", "", "{}"},
		{"NullObject", "", -32602, missing, "missing_payload", ""},
		{"NullObject", "null", 0, "", "", "null"},
		{"NullObject", "{\"name\":\"b\"}", 0, "", "", "{\"name\":\"b\"}"},
		{"NullObject", "{}", -32602, "Missing required field: name", "missing_field", ""},
		{"AnyValue", "", -32602, missing, "missing_payload", ""},
		{"AnyValue", "null", 0, "", "", "null"},
		{"AnyValue", "{}", 0, "", "", "{}"},
		{"AnyValue", "[1,2.50]", 0, "", "", "[1,2.50]"},
		{"NullUnion", "", -32602, missing, "missing_payload", ""},
		{"NullUnion", "null", 0, "", "", "null"},
		{"NullUnion", leaf, 0, "", "", leaf},
		{"NullUnion", "{}", -32602, "invalid value for \"type\": got \"\", expected one of \"Leaf\", \"Other\"", "invalid_enum_value", ""},
		{"PayloadAny", "", -32602, missing, "missing_payload", ""},
		{"PayloadAny", "null", 0, "", "", "null"},
		{"PayloadAny", "{}", 0, "", "", "{}"},
		{"PayloadNullString", "", -32602, missing, "missing_payload", ""},
		{"PayloadNullString", "null", 0, "", "", "null"},
		{"PayloadNullString", "\"a\"", 0, "", "", "\"a\""},
		{"Object", "", -32602, "Missing required field: name", "missing_field", ""},
		{"Object", "{\"name\":\"b\"}", 0, "", "", "{\"name\":\"b\"}"},
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
			if len(seen) != 1 || seen[0] != tc.want {
				t.Errorf("%s: server received %v, want one call with %s", name, seen, tc.want)
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
			t.Errorf("%s: service invoked with %v", name, seen)
		}
	}
}

// TestClient round-trips concrete values through the generated client and
// checks the params that it sends. Empty strings, arrays, objects and null
// values are sent as params; a method without payload sends none, and a
// request without an ID is sent as a notification without "id".
func TestClient(t *testing.T) {
	svc, wire, hs := serve(t)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	leaf := picker.NewLeafOrOtherLeaf(&picker.Leaf{Name: "a"})
	cases := []struct {
		name   string
		call   func() (any, error)
		params string
		// seen is what the service receives when it differs from params.
		seen string
		// notification is true when the request has no ID.
		notification bool
	}{
		{"NullString", func() (any, error) {
			return c.NullString()(ctx, &picker.NullStringPayload{ID: ptr("1"), V: loom.NullableValue("a")})
		}, "\"a\"", "", false},
		{"NullArray", func() (any, error) {
			return c.NullArray()(ctx, &picker.NullArrayPayload{ID: ptr("2"), V: loom.NullableValue([]string{"a"})})
		}, "[\"a\"]", "", false},
		{"NullMap", func() (any, error) {
			return c.NullMap()(ctx, &picker.NullMapPayload{ID: ptr("3"), V: loom.NullableValue(map[string]int{"k": 1})})
		}, "{\"k\":1}", "", false},
		{"NullObject", func() (any, error) {
			return c.NullObject()(ctx, &picker.NullObjectPayload{ID: ptr("4"), V: loom.NullableValue(picker.Leaf{Name: "a"})})
		}, "{\"name\":\"a\"}", "", false},
		{"AnyValue", func() (any, error) {
			return c.AnyValue()(ctx, &picker.AnyValuePayload{ID: ptr("5"), V: loom.JSONValue("[1,2.50]")})
		}, "[1,2.50]", "", false},
		{"NullUnion", func() (any, error) {
			return c.NullUnion()(ctx, &picker.NullUnionPayload{ID: ptr("6"), V: loom.NullableValue(leaf)})
		}, "{\"type\":\"Leaf\",\"value\":{\"name\":\"a\"}}", "", false},
		{"PayloadAny", func() (any, error) {
			return c.PayloadAny()(ctx, loom.JSONValue("{\"k\":1}"))
		}, "{\"k\":1}", "", false},
		{"PayloadNullString", func() (any, error) {
			return c.PayloadNullString()(ctx, loom.NullableValue("a"))
		}, "\"a\"", "", false},
		{"empty Str", func() (any, error) {
			return c.Str()(ctx, &picker.StrPayload{ID: ptr("7"), V: ""})
		}, "\"\"", "", false},
		{"empty Arr", func() (any, error) {
			return c.Arr()(ctx, &picker.ArrPayload{ID: ptr("8"), V: []string{}})
		}, "[]", "", false},
		{"empty object AnyValue", func() (any, error) {
			return c.AnyValue()(ctx, &picker.AnyValuePayload{ID: ptr("9"), V: loom.JSONValue("{}")})
		}, "{}", "", false},
		{"empty array AnyValue", func() (any, error) {
			return c.AnyValue()(ctx, &picker.AnyValuePayload{ID: ptr("10"), V: loom.JSONValue("[]")})
		}, "[]", "", false},
		{"empty string AnyValue", func() (any, error) {
			return c.AnyValue()(ctx, &picker.AnyValuePayload{ID: ptr("11"), V: loom.JSONValue("\"\"")})
		}, "\"\"", "", false},
		{"empty NullOpen", func() (any, error) {
			return c.NullOpen()(ctx, &picker.NullOpenPayload{ID: ptr("12"), V: loom.NullableValue(picker.Open{})})
		}, "{}", "", false},
		{"empty object PayloadAny", func() (any, error) {
			return c.PayloadAny()(ctx, loom.JSONValue("{}"))
		}, "{}", "", false},
		{"null NullString", func() (any, error) {
			return c.NullString()(ctx, &picker.NullStringPayload{ID: ptr("13"), V: loom.NullValue[string]()})
		}, "null", "", false},
		{"null NullUnion", func() (any, error) {
			return c.NullUnion()(ctx, &picker.NullUnionPayload{ID: ptr("14"), V: loom.NullValue[picker.LeafOrOther]()})
		}, "null", "", false},
		{"null AnyValue", func() (any, error) {
			return c.AnyValue()(ctx, &picker.AnyValuePayload{ID: ptr("15"), V: loom.JSONValue("null")})
		}, "null", "", false},
		{"nil AnyValue", func() (any, error) {
			return c.AnyValue()(ctx, &picker.AnyValuePayload{ID: ptr("16")})
		}, "null", "", false},
		{"null PayloadNullString", func() (any, error) {
			return c.PayloadNullString()(ctx, loom.NullValue[string]())
		}, "null", "", false},
		{"Ping", func() (any, error) {
			return c.Ping()(ctx, nil)
		}, "", "ping", false},
		{"notification", func() (any, error) {
			return c.AnyValue()(ctx, &picker.AnyValuePayload{V: loom.JSONValue("{}")})
		}, "{}", "", true},
	}
	for _, tc := range cases {
		// The server sends no response to a notification, which the
		// generated client reports as a success.
		if _, err := tc.call(); err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		want := tc.seen
		if want == "" {
			want = tc.params
		}
		if seen := svc.take(); len(seen) != 1 || seen[0] != want {
			t.Errorf("%s: server received %v, want %s", tc.name, seen, want)
		}
		sent := wire.take()
		if got := string(sentParams(t, sent)); got != tc.params {
			t.Errorf("%s: sent params %q, want %q", tc.name, got, tc.params)
		}
		var envelope map[string]jsontext.Value
		if err := json.Unmarshal(sent, &envelope); err != nil {
			t.Errorf("%s: decode request %q: %v", tc.name, sent, err)
			continue
		}
		if _, ok := envelope["id"]; ok == tc.notification {
			t.Errorf("%s: request %s has id %t, want %t", tc.name, sent, ok, !tc.notification)
		}
	}
}
`
