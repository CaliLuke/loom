package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCNestedUnionParamsGeneratedModuleRoundTrip compiles and vets a
// JSON-RPC service whose params and result hold a named union in a user
// type, directly and in an array and a map, next to an alias user type with
// a validation. It round-trips every branch through the generated client and
// server, sends invalid params and checks the JSON-RPC error code, message
// and error name, and that the service is not invoked.
func TestJSONRPCNestedUnionParamsGeneratedModuleRoundTrip(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcNestedUnionParamsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcnestedunion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested_union_test.go"), []byte(jsonRPCNestedUnionParamsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcNestedUnionParamsDSL() {
	dsl.API("nestedunion", func() {
		dsl.JSONRPC(func() {})
	})
	leaf := dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Attribute("count", dsl.Int)
	})
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	name := dsl.Type("Name", dsl.String, func() {
		dsl.MinLength(2)
	})
	holder := dsl.Type("Holder", func() {
		dsl.Attribute("choice", choice)
		dsl.Attribute("nm", name)
		dsl.Attribute("list", dsl.ArrayOf(choice))
		dsl.Attribute("dict", dsl.MapOf(dsl.String, choice))
		dsl.Required("choice")
	})
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("put", func() {
			dsl.Payload(func() {
				dsl.Attribute("holder", holder)
				dsl.Required("holder")
			})
			dsl.Result(holder)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCNestedUnionParamsHarness = `package jsonrpcnestedunion_test

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	rpcclient "example.com/jsonrpcnestedunion/gen/jsonrpc/rpc/client"
	rpcserver "example.com/jsonrpcnestedunion/gen/jsonrpc/rpc/server"
	rpc "example.com/jsonrpcnestedunion/gen/rpc"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []*rpc.Holder
}

func (s *service) Put(_ context.Context, p *rpc.PutPayload) (*rpc.Holder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p.Holder)
	return p.Holder, nil
}

func (s *service) take() []*rpc.Holder {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func ptr[T any](v T) *T { return &v }

func TestNestedUnionRoundTrip(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	rpcserver.Mount(mux, rpcserver.New(rpc.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := rpcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()

	name := rpc.Name("ab")
	for i, h := range []*rpc.Holder{
		{Choice: rpc.NewChoiceLeaf(&rpc.Leaf{Name: "a"})},
		{
			Choice: rpc.NewChoiceOther(&rpc.Other{Count: ptr(3)}),
			Nm:     &name,
			List:   []rpc.Choice{rpc.NewChoiceLeaf(&rpc.Leaf{Name: "b"}), rpc.NewChoiceOther(&rpc.Other{})},
			Dict:   map[string]rpc.Choice{"k": rpc.NewChoiceLeaf(&rpc.Leaf{Name: "c"})},
		},
	} {
		res, err := c.Put()(ctx, &rpc.PutPayload{Holder: h})
		if err != nil {
			t.Fatalf("put %d: %v", i, err)
		}
		if !reflect.DeepEqual(res, h) {
			t.Errorf("put %d: result %#v, want %#v", i, res, h)
		}
		if seen := svc.take(); len(seen) != 1 || !reflect.DeepEqual(seen[0], h) {
			t.Errorf("put %d: service received %+v, want %+v", i, seen, h)
		}
	}

	cases := []struct {
		name    string
		holder  string
		code    int
		message string
		errName string
	}{
		{"valid", ` + "`" + `{"choice":{"type":"Leaf","value":{"name":"x"}},"nm":"ab","list":[{"type":"Other","value":{}}],"dict":{"k":{"type":"Leaf","value":{"name":"y"}}}}` + "`" + `, 0, "", ""},
		{"missing choice", ` + "`" + `{"nm":"ab"}` + "`" + `, -32602, "Missing required field: choice", "missing_field"},
		{"unknown branch", ` + "`" + `{"choice":{"type":"Nope","value":{}}}` + "`" + `, -32602, ` + "`" + `invalid value for "type": got "Nope", expected one of "Leaf", "Other"` + "`" + `, "invalid_enum_value"},
		{"invalid branch", ` + "`" + `{"choice":{"type":"Leaf","value":{}}}` + "`" + `, -32602, "Missing required field: name", "missing_field"},
		{"invalid list branch", ` + "`" + `{"choice":{"type":"Other","value":{}},"list":[{"type":"Leaf","value":{}}]}` + "`" + `, -32602, "Missing required field: name", "missing_field"},
		{"invalid dict branch", ` + "`" + `{"choice":{"type":"Other","value":{}},"dict":{"k":{"type":"Leaf"}}}` + "`" + `, -32602, "Missing required field: value", "missing_field"},
	}
	for _, tc := range cases {
		body := ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"put","params":{"holder":` + "`" + ` + tc.holder + "}}"
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
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d (%s)", tc.name, resp.StatusCode, raw)
		}
		var envelope struct {
			Result jsontext.Value ` + "`" + `json:"result"` + "`" + `
			Error  *struct {
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
			if envelope.Error != nil || string(envelope.Result) != tc.holder {
				t.Errorf("%s: response %s", tc.name, raw)
			}
			if len(seen) != 1 || seen[0].Choice.Kind() != rpc.ChoiceKindLeaf || len(seen[0].List) != 1 || len(seen[0].Dict) != 1 {
				t.Errorf("%s: service received %+v", tc.name, seen)
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
`
