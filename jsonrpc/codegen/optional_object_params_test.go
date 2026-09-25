package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCOptionalObjectParams asserts that the JSON-RPC client request
// encoder omits params when the optional object payload attribute mapped to
// the params with Body is nil, and that the server request decoder then
// decodes absent params as a nil object: it does not substitute {} and it
// decodes into a pointer that stays nil for an empty body. A required object
// attribute is still always encoded and absent params still decode as {}.
func TestJSONRPCOptionalObjectParams(t *testing.T) {
	cases := []struct {
		Name     string
		Required bool
		Contains string
		Decode   []string
	}{
		{
			Name: "optional",
			Contains: "\t\tbody := &jsonrpc.Request{\n" +
				"\t\t\tJSONRPC: \"2.0\",\n" +
				"\t\t\tMethod:  \"Find\",\n" +
				"\t\t}\n" +
				"\t\tif p.O != nil {\n" +
				"\t\t\tbody.Params = NewFindRequestBody(p)\n" +
				"\t\t}\n",
			Decode: []string{
				"\t\tparams := req.Params\n\t\tr.Body = io.NopCloser(bytes.NewReader(params))\n",
				"\t\t\tbody = &FindRequestBody{}\n",
				"\t\terr = decoder(r).Decode(body)\n",
				"\t\t\t\tbody = nil\n",
				"\t\tif body != nil {\n",
				"err = ValidateFindRequestBody(body)\n",
				"payload = NewFindPayload(body, ",
			},
		},
		{
			Name:     "required",
			Required: true,
			Contains: "\t\tb := NewFindRequestBody(p)\n" +
				"\t\tbody := &jsonrpc.Request{\n" +
				"\t\t\tJSONRPC: \"2.0\",\n" +
				"\t\t\tMethod:  \"Find\",\n" +
				"\t\t\tParams:  b,\n" +
				"\t\t}\n",
			Decode: []string{
				"\t\tif len(params) == 0 {\n\t\t\tparams = []byte(\"{}\")\n\t\t}\n",
				"\t\t\tbody FindRequestBody\n",
				"\t\terr = decoder(r).Decode(&body)\n",
				"\t\terr = ValidateFindRequestBody(&body)\n",
				"payload = NewFindPayload(&body, ",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, jsonrpcOptionalObjectParamsDSL(c.Required))
			services := CreateJSONRPCServices(root)
			data := services.Get("Finder").Endpoint("Find")
			require.NotNil(t, data)
			assert.Equal(t, !c.Required, data.Payload.Request.OptionalBodyAttribute)
			assert.Equal(t, !c.Required, data.Payload.Request.OptionalObjectBody)

			file := requireEncodeDecodeFile(t, ClientFiles("", services), "client")
			code := sectionSourceByName(t, file, "jsonrpc-request-encoder")
			assert.Contains(t, code, c.Contains)
			if c.Required {
				assert.NotContains(t, code, "if p.O != nil {")
			}

			decoder := sectionSourceByName(t, requireEncodeDecodeFile(t, ServerFiles("", services), "server"), "jsonrpc-request-decoder")
			for _, want := range c.Decode {
				assert.Contains(t, decoder, want)
			}
			if !c.Required {
				assert.NotContains(t, decoder, "params = []byte(\"{}\")")
			}
		})
	}
}

// TestJSONRPCOptionalObjectParamsGeneratedModule compiles and vets a JSON-RPC
// service whose params are an optional object payload attribute selected with
// Body, round-trips a nil and a non-nil object through the generated client
// and server, checks that a nil object is sent without params, and sends
// absent, null, empty-object, valid and invalid params to the generated
// server.
func TestJSONRPCOptionalObjectParamsGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcOptionalObjectParamsDSL(false))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcoptobject", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "optional_object_params_test.go"), []byte(jsonRPCOptionalObjectParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

// jsonrpcOptionalObjectParamsDSL returns a JSON-RPC design whose method maps
// an object payload attribute with a required field to the params with Body.
// The attribute is required when required is true.
func jsonrpcOptionalObjectParamsDSL(required bool) func() {
	return func() {
		dsl.API("optobject", func() {
			dsl.JSONRPC(func() {})
		})
		filters := dsl.Type("Filters", func() {
			dsl.Attribute("name", dsl.String)
			dsl.Attribute("limit", dsl.Int)
			dsl.Required("name")
		})
		dsl.Service("Finder", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("Find", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
					dsl.Attribute("o", filters)
					if required {
						dsl.Required("o")
					}
				})
				dsl.JSONRPC(func() {
					dsl.Body("o")
				})
			})
		})
	}
}

const jsonRPCOptionalObjectParamsHarness = `package jsonrpcoptobject_test

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

	client "example.com/jsonrpcoptobject/gen/jsonrpc/finder/client"
	server "example.com/jsonrpcoptobject/gen/jsonrpc/finder/server"
	finder "example.com/jsonrpcoptobject/gen/finder"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []*finder.FindPayload
}

func (s *service) Find(_ context.Context, p *finder.FindPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
	return nil
}

func (s *service) take() []*finder.FindPayload {
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

func TestOptionalObjectParams(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(finder.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	wire := &wireRecorder{next: mux}
	hs := httptest.NewServer(wire)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []finder.FindPayload{
		{ID: ptr("1"), O: &finder.Filters{Name: "a", Limit: ptr(2)}},
		{ID: ptr("2")},
	}
	for _, p := range payloads {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("client panicked: %v", r)
				}
			}()
			_, err = c.Find()(context.Background(), &p)
			return err
		}()
		if err != nil {
			t.Errorf("find %+v: %v", p, err)
			continue
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(seen[0].O, p.O) {
			t.Errorf("find: server received %+v, want object %+v", seen, p.O)
		}
		params := sentParams(t, wire.take())
		if p.O == nil && params != nil {
			t.Errorf("find nil object: sent params %s, want none", params)
		}
		if p.O != nil && params == nil {
			t.Errorf("find %+v: request has no params", p.O)
		}
	}

	cases := []struct {
		name    string
		body    string
		code    int
		message string
		errName string
		want    *finder.Filters
	}{
		{"absent params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Find"}` + "`" + `, 0, "", "", nil},
		{"valid params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Find","params":{"name":"x"}}` + "`" + `, 0, "", "", &finder.Filters{Name: "x"}},
		{"null params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Find","params":null}` + "`" + `, -32602, "Missing required field: name", "missing_field", nil},
		{"empty object params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Find","params":{}}` + "`" + `, -32602, "Missing required field: name", "missing_field", nil},
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
			if len(seen) != 1 || !reflect.DeepEqual(seen[0].O, tc.want) {
				t.Errorf("%s: server received %+v, want one call with object %+v", tc.name, seen, tc.want)
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
