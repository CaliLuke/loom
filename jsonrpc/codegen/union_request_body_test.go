package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCUnionRequestBodyDeclaration asserts that the JSON-RPC server
// declares params whose type is a union by value and decodes into its
// address, over HTTP and over WebSocket.
func TestJSONRPCUnionRequestBodyDeclaration(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"http", jsonrpcUnionRequestBodyDSL(false)},
		{"websocket", jsonrpcUnionRequestBodyDSL(true)},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, c.DSL)
			file := requireEncodeDecodeFile(t, ServerFiles("", CreateJSONRPCServices(root)), "server")
			code := sectionSourceByName(t, file, "jsonrpc-request-decoder")
			assert.Contains(t, code, "body PickRequestBody\n")
			assert.Contains(t, code, "err = decoder(r).Decode(&body)")
			assert.Contains(t, code, "payload = NewPickLeafOrOther(&body,")
			assert.NotContains(t, code, "body *Pick")
		})
	}
}

// TestJSONRPCUnionRequestBodyGeneratedModuleRoundTrip compiles and vets
// JSON-RPC services whose params or WebSocket streaming payload are a
// constructor OneOf union and round-trips every branch through the generated
// client and server. Over HTTP it also sends empty, whitespace, malformed,
// truncated and invalid requests and checks the HTTP status, the JSON-RPC
// error code, message and error name, and that the service is not invoked.
func TestJSONRPCUnionRequestBodyGeneratedModuleRoundTrip(t *testing.T) {
	cases := []struct {
		Name      string
		WebSocket bool
		Harness   string
	}{
		{"http", false, jsonRPCUnionParamsHarness},
		{"websocket", true, jsonRPCUnionStreamHarness},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, jsonrpcUnionRequestBodyDSL(c.WebSocket))
			dir := t.TempDir()
			renderJSONRPCModule(t, dir, "example.com/jsonrpcunionparams", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "union_params_test.go"), []byte(c.Harness), 0o600))
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
			runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}

// jsonrpcUnionRequestBodyDSL returns a JSON-RPC design whose method takes a
// constructor OneOf union as params, or as its streaming payload over
// WebSocket when websocket is true. The result is not a union.
func jsonrpcUnionRequestBodyDSL(websocket bool) func() {
	return func() {
		dsl.API("unionparams", func() {
			dsl.JSONRPC(func() {})
		})
		leaf := dsl.Type("Leaf", func() {
			dsl.Attribute("name", dsl.String)
			dsl.Required("name")
		})
		other := dsl.Type("Other", func() {
			dsl.Attribute("count", dsl.Int)
		})
		ack := dsl.Type("Ack", func() {
			dsl.Attribute("kind", dsl.String)
			dsl.Required("kind")
		})
		dsl.Service("Picker", func() {
			dsl.JSONRPC(func() {
				if websocket {
					dsl.GET("/rpc")
				} else {
					dsl.POST("/rpc")
				}
			})
			dsl.Method("Pick", func() {
				if websocket {
					dsl.StreamingPayload(dsl.OneOf(leaf, other))
					dsl.StreamingResult(ack)
				} else {
					dsl.Payload(dsl.OneOf(leaf, other))
					dsl.Result(ack)
				}
				dsl.JSONRPC(func() {})
			})
		})
	}
}

const jsonRPCUnionParamsHarness = `package jsonrpcunionparams_test

import (
	"context"
	json "encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	client "example.com/jsonrpcunionparams/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcunionparams/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcunionparams/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	mu   sync.Mutex
	seen []*picker.LeafOrOther
}

func (s *service) Pick(_ context.Context, p *picker.LeafOrOther) (*picker.Ack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, p)
	return &picker.Ack{Kind: string(p.Kind())}, nil
}

func (s *service) take() []*picker.LeafOrOther {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func ptr[T any](v T) *T { return &v }

func TestUnionParamsRoundTrip(t *testing.T) {
	svc := &service{}
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	server.Mount(mux, server.New(picker.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)

	payloads := []picker.LeafOrOther{
		picker.NewLeafOrOtherLeaf(&picker.Leaf{Name: "a"}),
		picker.NewLeafOrOtherOther(&picker.Other{Count: ptr(3)}),
		picker.NewLeafOrOtherOther(&picker.Other{}),
	}
	for _, p := range payloads {
		res, err := c.Pick()(context.Background(), &p)
		if err != nil {
			t.Fatalf("pick %s: %v", p.Kind(), err)
		}
		if ack, ok := res.(*picker.Ack); !ok || ack.Kind != string(p.Kind()) {
			t.Errorf("pick %s: result %#v", p.Kind(), res)
		}
		seen := svc.take()
		if len(seen) != 1 || !reflect.DeepEqual(*seen[0], p) {
			t.Errorf("pick %s: server received %+v, want %+v", p.Kind(), seen, p)
		}
	}

	cases := []struct {
		name    string
		body    string
		status  int
		code    int
		message string
		errName string
		want    *picker.LeafOrOther
	}{
		{"leaf", request(` + "`" + `{"type":"Leaf","value":{"name":"b"}}` + "`" + `), http.StatusOK, 0, "", "", ptr(picker.NewLeafOrOtherLeaf(&picker.Leaf{Name: "b"}))},
		{"other", request(` + "`" + `{"type":"Other","value":{}}` + "`" + `), http.StatusOK, 0, "", "", ptr(picker.NewLeafOrOtherOther(&picker.Other{}))},
		{"empty", "", http.StatusOK, -32700, "Parse error", "", nil},
		{"whitespace", " \n\t", http.StatusOK, -32700, "Parse error", "", nil},
		{"malformed", "{x}", http.StatusOK, -32700, "Parse error", "", nil},
		{"truncated", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":{"type":"Leaf"` + "`" + `, http.StatusOK, -32700, "Parse error", "", nil},
		{"absent params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick"}` + "`" + `, http.StatusOK, -32602, ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, "invalid_enum_value", nil},
		{"null params", request("null"), http.StatusOK, -32602, ` + "`" + `invalid value for "type": got "", expected one of "Leaf", "Other"` + "`" + `, "invalid_enum_value", nil},
		{"unknown branch", request(` + "`" + `{"type":"Nope","value":{}}` + "`" + `), http.StatusOK, -32602, ` + "`" + `invalid value for "type": got "Nope", expected one of "Leaf", "Other"` + "`" + `, "invalid_enum_value", nil},
		{"missing value", request(` + "`" + `{"type":"Leaf"}` + "`" + `), http.StatusOK, -32602, "Missing required field: value", "missing_field", nil},
		{"null value", request(` + "`" + `{"type":"Leaf","value":null}` + "`" + `), http.StatusOK, -32602, "Missing required field: value", "missing_field", nil},
		{"invalid branch", request(` + "`" + `{"type":"Leaf","value":{}}` + "`" + `), http.StatusOK, -32602, "Missing required field: name", "missing_field", nil},
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
		if resp.StatusCode != tc.status {
			t.Errorf("%s: status %d, want %d (%s)", tc.name, resp.StatusCode, tc.status, raw)
		}
		var envelope struct {
			Result *picker.Ack ` + "`" + `json:"result"` + "`" + `
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
		if tc.want == nil {
			switch {
			case envelope.Error == nil:
				t.Errorf("%s: no error in %s", tc.name, raw)
			case envelope.Error.Code != tc.code || envelope.Error.Message != tc.message || envelope.Error.Data.Name != tc.errName:
				t.Errorf("%s: error (%d, %q, %q), want (%d, %q, %q)", tc.name, envelope.Error.Code, envelope.Error.Message, envelope.Error.Data.Name, tc.code, tc.message, tc.errName)
			}
			if len(seen) != 0 {
				t.Errorf("%s: service invoked with %+v", tc.name, seen)
			}
			continue
		}
		if envelope.Error != nil || envelope.Result == nil || envelope.Result.Kind != string(tc.want.Kind()) {
			t.Errorf("%s: response %s", tc.name, raw)
		}
		if len(seen) != 1 || !reflect.DeepEqual(seen[0], tc.want) {
			t.Errorf("%s: server received %+v, want %+v", tc.name, seen, tc.want)
		}
	}
}

// request returns a JSON-RPC request envelope for the Pick method with the
// given raw params.
func request(params string) string {
	return ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"Pick","params":` + "`" + ` + params + "}"
}
`

const jsonRPCUnionStreamHarness = `package jsonrpcunionparams_test

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	client "example.com/jsonrpcunionparams/gen/jsonrpc/picker/client"
	server "example.com/jsonrpcunionparams/gen/jsonrpc/picker/server"
	picker "example.com/jsonrpcunionparams/gen/picker"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	seen chan *picker.LeafOrOther
}

func (s *service) HandleStream(ctx context.Context, stream picker.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (s *service) Pick(ctx context.Context, p *picker.LeafOrOther, stream picker.PickServerStream) error {
	s.seen <- p
	return stream.SendResponse(ctx, &picker.Ack{Kind: string(p.Kind())})
}

func ptr[T any](v T) *T { return &v }

func TestUnionStreamingPayloadRoundTrip(t *testing.T) {
	svc := &service{seen: make(chan *picker.LeafOrOther, 8)}
	mux := loomhttp.NewMuxer()
	endpoints := picker.NewEndpoints(svc)
	server.Mount(mux, server.New(svc.HandleStream, endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	defer func() {
		if err := c.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := c.Pick()(ctx, nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(*client.PickClientStream)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}
	payloads := []picker.LeafOrOther{
		picker.NewLeafOrOtherLeaf(&picker.Leaf{Name: "a"}),
		picker.NewLeafOrOtherOther(&picker.Other{Count: ptr(3)}),
		picker.NewLeafOrOtherOther(&picker.Other{}),
	}
	for _, p := range payloads {
		if err := stream.SendWithContext(ctx, &p); err != nil {
			t.Fatalf("send %s: %v", p.Kind(), err)
		}
		ack, err := stream.RecvWithContext(ctx)
		if err != nil {
			t.Fatalf("recv %s: %v", p.Kind(), err)
		}
		if ack == nil || ack.Kind != string(p.Kind()) {
			t.Errorf("recv %s: got %+v", p.Kind(), ack)
		}
		select {
		case got := <-svc.seen:
			if !reflect.DeepEqual(*got, p) {
				t.Errorf("send %s: server received %+v, want %+v", p.Kind(), got, p)
			}
		case <-ctx.Done():
			t.Fatalf("send %s: service not invoked", p.Kind())
		}
	}
}
`
