package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCAliasParamsGeneratedModule compiles and vets a JSON-RPC service
// whose params are an optional or required alias of a primitive, an array
// or a map with validations, or a union with an alias of a primitive
// branch, selected with Body. It round-trips values through the generated
// client and server, checks that a nil optional value is sent without
// params, and sends absent, null, valid, invalid and malformed params to the
// generated server, checking the error code, message and name, whether the
// service is invoked and the decoded value.
func TestJSONRPCAliasParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcAliasParamsModuleDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcalias", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alias_params_test.go"), []byte(jsonRPCAliasParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// jsonrpcAliasParamsModuleDSL declares a JSON-RPC service whose methods map
// an optional and a required alias of String, Int, an array and a map with
// validations, and a union with an alias of String branch, to the params
// with Body("v").
func jsonrpcAliasParamsModuleDSL() {
	dsl.API("alias", func() {
		dsl.JSONRPC(func() {
			// Enable JSON-RPC for this API.
		})
	})
	var Code = dsl.Type("Code", dsl.String, func() {
		dsl.MinLength(1)
	})
	var Amount = dsl.Type("Amount", dsl.Int, func() {
		dsl.Minimum(1)
	})
	var Tags = dsl.Type("Tags", dsl.ArrayOf(dsl.String), func() {
		dsl.MinLength(1)
	})
	var Dict = dsl.Type("Dict", dsl.MapOf(dsl.String, dsl.Int), func() {
		dsl.MaxLength(2)
	})
	var Leaf = dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	methods := []struct {
		name string
		att  any
	}{
		{"Code", Code},
		{"Amount", Amount},
		{"Tags", Tags},
		{"Dict", Dict},
		{"Union", dsl.OneOf(Leaf, Code)},
	}
	dsl.Service("Picker", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		for _, m := range methods {
			for _, required := range []bool{false, true} {
				name := "Send" + m.name
				if required {
					name += "Req"
				}
				dsl.Method(name, func() {
					dsl.Payload(func() {
						dsl.ID("id", dsl.String)
						dsl.Attribute("v", m.att)
						if required {
							dsl.Required("v")
						}
					})
					dsl.JSONRPC(func() {
						dsl.Body("v")
					})
				})
			}
		}
	})
}

const jsonRPCAliasParamsHarness = `package jsonrpcalias_test

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"encoding/json/jsontext"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcalias/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcalias/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcalias/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []any
}

func (s *service) SendCode(_ context.Context, p *picker.SendCodePayload) error {
	return s.record(p)
}
func (s *service) SendCodeReq(_ context.Context, p *picker.SendCodeReqPayload) error {
	return s.record(p)
}
func (s *service) SendAmount(_ context.Context, p *picker.SendAmountPayload) error {
	return s.record(p)
}
func (s *service) SendAmountReq(_ context.Context, p *picker.SendAmountReqPayload) error {
	return s.record(p)
}
func (s *service) SendTags(_ context.Context, p *picker.SendTagsPayload) error {
	return s.record(p)
}
func (s *service) SendTagsReq(_ context.Context, p *picker.SendTagsReqPayload) error {
	return s.record(p)
}
func (s *service) SendDict(_ context.Context, p *picker.SendDictPayload) error {
	return s.record(p)
}
func (s *service) SendDictReq(_ context.Context, p *picker.SendDictReqPayload) error {
	return s.record(p)
}
func (s *service) SendUnion(_ context.Context, p *picker.SendUnionPayload) error {
	return s.record(p)
}
func (s *service) SendUnionReq(_ context.Context, p *picker.SendUnionReqPayload) error {
	return s.record(p)
}

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

func ptr[T any](v T) *T {
	return &v
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

// TestClient round-trips values through the generated client and checks the
// params that it sends: none for a nil optional value.
func TestClient(t *testing.T) {
	svc, wire, hs := serve(t)
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	code := picker.NewCodeOrLeafCode(picker.Code("ab"))
	leaf := picker.NewCodeOrLeafLeaf(&picker.Leaf{Name: "a"})
	cases := []struct {
		name    string
		call    func(context.Context, any) (any, error)
		payload any
		params  string
	}{
		{"SendCode", c.SendCode(), &picker.SendCodePayload{ID: ptr("1"), V: ptr(picker.Code("ab"))}, "\"ab\""},
		{"SendCode nil", c.SendCode(), &picker.SendCodePayload{ID: ptr("2")}, ""},
		{"SendCodeReq", c.SendCodeReq(), &picker.SendCodeReqPayload{ID: ptr("3"), V: "ab"}, "\"ab\""},
		{"SendAmount", c.SendAmount(), &picker.SendAmountPayload{ID: ptr("4"), V: ptr(picker.Amount(2))}, "2"},
		{"SendAmount nil", c.SendAmount(), &picker.SendAmountPayload{ID: ptr("5")}, ""},
		{"SendAmountReq", c.SendAmountReq(), &picker.SendAmountReqPayload{ID: ptr("6"), V: 2}, "2"},
		{"SendTags", c.SendTags(), &picker.SendTagsPayload{ID: ptr("7"), V: picker.Tags{"a"}}, "[\"a\"]"},
		{"SendTags nil", c.SendTags(), &picker.SendTagsPayload{ID: ptr("8")}, ""},
		{"SendTagsReq", c.SendTagsReq(), &picker.SendTagsReqPayload{ID: ptr("9"), V: picker.Tags{"a"}}, "[\"a\"]"},
		{"SendDict", c.SendDict(), &picker.SendDictPayload{ID: ptr("10"), V: picker.Dict{"k": 1}}, "{\"k\":1}"},
		{"SendDict nil", c.SendDict(), &picker.SendDictPayload{ID: ptr("11")}, ""},
		{"SendDictReq", c.SendDictReq(), &picker.SendDictReqPayload{ID: ptr("12"), V: picker.Dict{"k": 1}}, "{\"k\":1}"},
		{"SendUnion code", c.SendUnion(), &picker.SendUnionPayload{ID: ptr("13"), V: &code}, "{\"type\":\"Code\",\"value\":\"ab\"}"},
		{"SendUnion leaf", c.SendUnion(), &picker.SendUnionPayload{ID: ptr("14"), V: &leaf}, "{\"type\":\"Leaf\",\"value\":{\"name\":\"a\"}}"},
		{"SendUnion nil", c.SendUnion(), &picker.SendUnionPayload{ID: ptr("15")}, ""},
		{"SendUnionReq", c.SendUnionReq(), &picker.SendUnionReqPayload{ID: ptr("16"), V: code}, "{\"type\":\"Code\",\"value\":\"ab\"}"},
	}
	for _, tc := range cases {
		if _, err := tc.call(context.Background(), tc.payload); err != nil {
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

// TestWire sends absent, null, valid, invalid and malformed params and checks
// the error or the value that the service receives.
func TestWire(t *testing.T) {
	svc, _, hs := serve(t)
	const (
		malformed = "invalid request body"
		invalid   = "validation error"
	)
	code := picker.NewCodeOrLeafCode(picker.Code("ab"))
	cases := []struct {
		method  string
		params  string
		code    int
		message string
		errName string
		want    any
	}{
		{"SendCode", "", 0, "", "", (*picker.Code)(nil)},
		{"SendCode", "\"ab\"", 0, "", "", ptr(picker.Code("ab"))},
		{"SendCode", "\"\"", -32602, invalid, "invalid_length", nil},
		{"SendCode", "null", -32602, invalid, "invalid_length", nil},
		{"SendCode", "{}", -32602, malformed, "decode_payload", nil},
		{"SendCode", "\"ab", -32700, "", "", nil},
		{"SendCodeReq", "", -32602, malformed, "decode_payload", nil},
		{"SendCodeReq", "\"ab\"", 0, "", "", picker.Code("ab")},
		{"SendCodeReq", "\"\"", -32602, invalid, "invalid_length", nil},
		{"SendCodeReq", "1", -32602, malformed, "decode_payload", nil},
		{"SendAmount", "", 0, "", "", (*picker.Amount)(nil)},
		{"SendAmount", "2", 0, "", "", ptr(picker.Amount(2))},
		{"SendAmount", "0", -32602, invalid, "invalid_range", nil},
		{"SendAmountReq", "2", 0, "", "", picker.Amount(2)},
		{"SendAmountReq", "0", -32602, invalid, "invalid_range", nil},
		{"SendAmountReq", "\"x\"", -32602, malformed, "decode_payload", nil},
		{"SendTags", "", 0, "", "", picker.Tags(nil)},
		{"SendTags", "null", 0, "", "", picker.Tags(nil)},
		{"SendTags", "[\"a\"]", 0, "", "", picker.Tags{"a"}},
		{"SendTags", "[]", -32602, invalid, "invalid_length", nil},
		{"SendTagsReq", "[\"a\"]", 0, "", "", picker.Tags{"a"}},
		{"SendTagsReq", "[]", -32602, invalid, "invalid_length", nil},
		{"SendTagsReq", "{}", -32602, malformed, "decode_payload", nil},
		{"SendDict", "", 0, "", "", picker.Dict(nil)},
		{"SendDict", "{\"k\":1}", 0, "", "", picker.Dict{"k": 1}},
		{"SendDict", "{\"a\":1,\"b\":2,\"c\":3}", -32602, invalid, "invalid_length", nil},
		{"SendDictReq", "{}", 0, "", "", picker.Dict{}},
		{"SendDictReq", "{\"a\":1,\"b\":2,\"c\":3}", -32602, invalid, "invalid_length", nil},
		{"SendDictReq", "[]", -32602, malformed, "decode_payload", nil},
		{"SendUnion", "", 0, "", "", (*picker.CodeOrLeaf)(nil)},
		{"SendUnion", "{\"type\":\"Code\",\"value\":\"ab\"}", 0, "", "", &code},
		{"SendUnion", "{\"type\":\"Code\",\"value\":\"\"}", -32602, invalid, "invalid_length", nil},
		{"SendUnionReq", "{\"type\":\"Code\",\"value\":\"ab\"}", 0, "", "", code},
		{"SendUnionReq", "{\"type\":\"Code\",\"value\":\"\"}", -32602, invalid, "invalid_length", nil},
		{"SendUnionReq", "[]", -32602, malformed, "decode_payload", nil},
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
