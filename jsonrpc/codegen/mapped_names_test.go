package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCMappedNamesGeneratedModule compiles and vets a JSON-RPC service
// whose params and result declare attributes with a transport element name
// suffix, such as "n:m", round-trips values through the generated client and
// server, and sends raw requests to the generated server, checking that the
// params and the result use the suffix as the JSON name of the field, that
// union branches are identified by their attribute names and that missing
// required fields are reported by attribute name.
func TestJSONRPCMappedNamesGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcMappedNamesDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcmappednames", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapped_names_test.go"), []byte(jsonRPCMappedNamesHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcMappedNamesDSL() {
	API("mappednames", func() {
		JSONRPC(func() {})
	})
	var Leaf = Type("Leaf", func() {
		Attribute("leaf:l", String)
		Attribute("count:c", Int)
		Required("count:c")
	})
	var Envelope = Type("Envelope", func() {
		Attribute("n:m", String, func() {
			MinLength(2)
		})
		Attribute("req:r", Int)
		Attribute("def:d", Int, func() {
			Default(3)
		})
		Attribute("pick:p", OneOf(String, Int))
		Attribute("obj:o", Leaf)
		Attribute("list:ls", ArrayOf(String))
		OneOf("choice:ch", func() {
			Attribute("text:t", String)
			Attribute("leaf_branch:lb", Leaf)
		})
		Required("req:r", "pick:p", "obj:o")
	})
	Service("mappednames", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("echo", func() {
			Payload(Envelope)
			Result(Envelope)
			JSONRPC(func() {})
		})
	})
}

const jsonRPCMappedNamesHarness = `package jsonrpcmappednames_test

import (
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

	client "example.com/jsonrpcmappednames/gen/jsonrpc/mappednames/client"
	server "example.com/jsonrpcmappednames/gen/jsonrpc/mappednames/server"
	mappednames "example.com/jsonrpcmappednames/gen/mappednames"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []*mappednames.Envelope
}

func (s *service) Echo(_ context.Context, p *mappednames.Envelope) (*mappednames.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
	return p, nil
}

func (s *service) take() []*mappednames.Envelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func ptr[T any](v T) *T { return &v }

func TestMappedNames(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(mappednames.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	full := &mappednames.Envelope{N: ptr("name"), Req: 7, Def: 5, Obj: &mappednames.Leaf{Leaf: ptr("leaf"), Count: 2}, List: []string{"a"}}
	full.Pick.SetString("picked")
	full.Choice = &mappednames.Choice{}
	full.Choice.SetText("text")
	minimal := &mappednames.Envelope{Def: 3, Obj: &mappednames.Leaf{}}
	minimal.Pick.SetInt(4)
	for _, envelope := range []*mappednames.Envelope{full, minimal} {
		res, err := c.Echo()(context.Background(), envelope)
		if err != nil {
			t.Errorf("echo %+v: %v", envelope, err)
			continue
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], envelope) {
			t.Errorf("echo: server received %+v, want %+v", seen, envelope)
		}
		if !reflect.DeepEqual(res, envelope) {
			t.Errorf("echo: client received %+v, want %+v", res, envelope)
		}
	}

	want := &mappednames.Envelope{Req: 1, Def: 3, Obj: &mappednames.Leaf{Count: 2}}
	want.Pick.SetInt(4)
	branch := &mappednames.Envelope{Req: 1, Def: 3, Obj: &mappednames.Leaf{Count: 2}}
	branch.Pick.SetInt(4)
	branch.Choice = &mappednames.Choice{}
	branch.Choice.SetText("t")
	cases := []struct {
		name    string
		params  string
		code    int
		message string
		want    *mappednames.Envelope
		result  map[string]any
	}{
		{"element names", ` + "`" + `{"r":1,"p":{"type":"Int","value":4},"o":{"c":2}}` + "`" + `, 0, "", want,
			map[string]any{"r": 1.0, "d": 3.0, "p": map[string]any{"type": "Int", "value": 4.0}, "o": map[string]any{"c": 2.0}}},
		{"union branch", ` + "`" + `{"r":1,"p":{"type":"Int","value":4},"o":{"c":2},"ch":{"type":"text","value":"t"}}` + "`" + `, 0, "", branch,
			map[string]any{"r": 1.0, "d": 3.0, "p": map[string]any{"type": "Int", "value": 4.0}, "o": map[string]any{"c": 2.0}, "ch": map[string]any{"type": "text", "value": "t"}}},
		{"attribute names", ` + "`" + `{"req":1,"pick":{"type":"Int","value":4},"obj":{"count":2}}` + "`" + `, -32602, "Missing required field: req", nil, nil},
		{"missing nested required", ` + "`" + `{"r":1,"p":{"type":"Int","value":4},"o":{}}` + "`" + `, -32602, "Missing required field: count", nil, nil},
	}
	for _, tc := range cases {
		body := ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"echo","params":` + "`" + ` + tc.params + "}"
		resp, err := hs.Client().Post(hs.URL+"/rpc", "application/json", strings.NewReader(body))
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
		var envelope struct {
			Result jsontext.Value ` + "`" + `json:"result"` + "`" + `
			Error  *struct {
				Code    int    ` + "`" + `json:"code"` + "`" + `
				Message string ` + "`" + `json:"message"` + "`" + `
			} ` + "`" + `json:"error"` + "`" + `
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Errorf("%s: decode response %q: %v", tc.name, raw, err)
			continue
		}
		seen := svc.take()
		if tc.code != 0 {
			if envelope.Error == nil || envelope.Error.Code != tc.code || envelope.Error.Message != tc.message {
				t.Errorf("%s: response %s, want error (%d, %q)", tc.name, raw, tc.code, tc.message)
			}
			if len(seen) != 0 {
				t.Errorf("%s: service invoked with %+v", tc.name, seen)
			}
			continue
		}
		if envelope.Error != nil {
			t.Errorf("%s: unexpected error %s", tc.name, raw)
			continue
		}
		if len(seen) != 1 || !reflect.DeepEqual(seen[0], tc.want) {
			t.Errorf("%s: server received %+v, want %+v", tc.name, seen, tc.want)
		}
		var result map[string]any
		if err := json.Unmarshal(envelope.Result, &result); err != nil || !reflect.DeepEqual(result, tc.result) {
			t.Errorf("%s: result %s, want %v (%v)", tc.name, envelope.Result, tc.result, err)
		}
	}
}
`
