package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCUnionBodyNamesGeneratedModuleRoundTrip compiles and vets
// JSON-RPC services whose params, results and SSE and WebSocket streaming
// results are anonymous and named unions, and round-trips every branch
// through the generated client and server. It sends empty, whitespace,
// malformed, truncated and invalid requests and checks the HTTP status, the
// JSON-RPC error code, message and error name, and that the service is not
// invoked. It also serves valid and invalid union results to the generated
// client and checks how they decode.
func TestJSONRPCUnionBodyNamesGeneratedModuleRoundTrip(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcUnionBodyNamesDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcunionnames", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_names_test.go"), []byte(jsonRPCUnionBodyNamesHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcUnionBodyNamesDSL() {
	dsl.API("unionnames", func() {
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
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("echo", func() {
			dsl.Payload(dsl.OneOf(leaf, other))
			dsl.Result(dsl.OneOf(leaf, other))
			dsl.JSONRPC(func() {})
		})
		dsl.Method("named", func() {
			dsl.Payload(choice)
			dsl.Result(choice)
			dsl.JSONRPC(func() {})
		})
	})
	dsl.Service("events", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/events")
		})
		dsl.Method("watch", func() {
			dsl.Payload(dsl.OneOf(leaf, other))
			dsl.StreamingResult(choice)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
	dsl.Service("socket", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/socket")
		})
		dsl.Method("talk", func() {
			dsl.StreamingPayload(leaf)
			dsl.StreamingResult(choice)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCUnionBodyNamesHarness = `package jsonrpcunionnames_test

import (
	"context"
	json "encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	events "example.com/jsonrpcunionnames/gen/events"
	eventsclient "example.com/jsonrpcunionnames/gen/jsonrpc/events/client"
	eventsserver "example.com/jsonrpcunionnames/gen/jsonrpc/events/server"
	rpcclient "example.com/jsonrpcunionnames/gen/jsonrpc/rpc/client"
	rpcserver "example.com/jsonrpcunionnames/gen/jsonrpc/rpc/server"
	socketclient "example.com/jsonrpcunionnames/gen/jsonrpc/socket/client"
	socketserver "example.com/jsonrpcunionnames/gen/jsonrpc/socket/server"
	rpc "example.com/jsonrpcunionnames/gen/rpc"
	socket "example.com/jsonrpcunionnames/gen/socket"
	loomhttp "github.com/CaliLuke/loom/http"
)

type rpcService struct {
	mu   sync.Mutex
	seen []string
}

func (s *rpcService) record(kind string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, kind)
}

func (s *rpcService) take() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen
	s.seen = nil
	return seen
}

func (s *rpcService) Echo(_ context.Context, p *rpc.LeafOrOther) (*rpc.LeafOrOther, error) {
	s.record(string(p.Kind()))
	return p, nil
}

func (s *rpcService) Named(_ context.Context, p *rpc.Choice) (*rpc.Choice, error) {
	s.record(string(p.Kind()))
	return p, nil
}

type eventsService struct{}

func (eventsService) Watch(ctx context.Context, p *events.LeafOrOther, st events.WatchServerStream) error {
	if err := st.Send(ctx, ptr(events.NewChoiceLeaf(&events.Leaf{Name: string(p.Kind())}))); err != nil {
		return err
	}
	return st.SendAndClose(ctx, ptr(events.NewChoiceOther(&events.Other{Count: ptr(1)})))
}

type socketService struct{}

func (socketService) HandleStream(ctx context.Context, stream socket.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (socketService) Talk(ctx context.Context, p *socket.Leaf, st socket.TalkServerStream) error {
	if p.Name == "other" {
		return st.SendResponse(ctx, ptr(socket.NewChoiceOther(&socket.Other{})))
	}
	return st.SendResponse(ctx, ptr(socket.NewChoiceLeaf(p)))
}

func ptr[T any](v T) *T { return &v }

func host(hs *httptest.Server) string { return strings.TrimPrefix(hs.URL, "http://") }

func TestUnaryUnionRoundTrip(t *testing.T) {
	svc := &rpcService{}
	mux := loomhttp.NewMuxer()
	rpcserver.Mount(mux, rpcserver.New(rpc.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := rpcclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()

	for _, p := range []rpc.LeafOrOther{
		rpc.NewLeafOrOtherLeaf(&rpc.Leaf{Name: "a"}),
		rpc.NewLeafOrOtherOther(&rpc.Other{Count: ptr(3)}),
		rpc.NewLeafOrOtherOther(&rpc.Other{}),
	} {
		res, err := c.Echo()(ctx, &p)
		if err != nil {
			t.Fatalf("echo %s: %v", p.Kind(), err)
		}
		if got, ok := res.(*rpc.LeafOrOther); !ok || !reflect.DeepEqual(*got, p) {
			t.Errorf("echo %s: result %#v, want %#v", p.Kind(), res, p)
		}
		if seen := svc.take(); !reflect.DeepEqual(seen, []string{string(p.Kind())}) {
			t.Errorf("echo %s: service received %v", p.Kind(), seen)
		}
	}
	for _, p := range []rpc.Choice{
		rpc.NewChoiceLeaf(&rpc.Leaf{Name: "a"}),
		rpc.NewChoiceOther(&rpc.Other{Count: ptr(3)}),
	} {
		res, err := c.Named()(ctx, &p)
		if err != nil {
			t.Fatalf("named %s: %v", p.Kind(), err)
		}
		if got, ok := res.(*rpc.Choice); !ok || !reflect.DeepEqual(*got, p) {
			t.Errorf("named %s: result %#v, want %#v", p.Kind(), res, p)
		}
		if seen := svc.take(); !reflect.DeepEqual(seen, []string{string(p.Kind())}) {
			t.Errorf("named %s: service received %v", p.Kind(), seen)
		}
	}

	enum := func(got string) string {
		return ` + "`" + `invalid value for "type": got "` + "`" + ` + got + ` + "`" + `", expected one of "Leaf", "Other"` + "`" + `
	}
	for _, method := range []string{"echo", "named"} {
		request := func(params string) string {
			return ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"` + "`" + ` + method + ` + "`" + `","params":` + "`" + ` + params + "}"
		}
		cases := []struct {
			name    string
			body    string
			code    int
			message string
			errName string
			kind    string
		}{
			{"leaf", request(` + "`" + `{"type":"Leaf","value":{"name":"b"}}` + "`" + `), 0, "", "", "Leaf"},
			{"other", request(` + "`" + `{"type":"Other","value":{}}` + "`" + `), 0, "", "", "Other"},
			{"empty", "", -32700, "Parse error", "", ""},
			{"whitespace", " \n\t", -32700, "Parse error", "", ""},
			{"malformed", "{x}", -32700, "Parse error", "", ""},
			{"truncated", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"` + "`" + ` + method + ` + "`" + `","params":{"type":"Leaf"` + "`" + `, -32700, "Parse error", "", ""},
			{"absent params", ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"` + "`" + ` + method + ` + "`" + `"}` + "`" + `, -32602, enum(""), "invalid_enum_value", ""},
			{"null params", request("null"), -32602, enum(""), "invalid_enum_value", ""},
			{"unknown branch", request(` + "`" + `{"type":"Nope","value":{}}` + "`" + `), -32602, enum("Nope"), "invalid_enum_value", ""},
			{"missing value", request(` + "`" + `{"type":"Leaf"}` + "`" + `), -32602, "Missing required field: value", "missing_field", ""},
			{"null value", request(` + "`" + `{"type":"Leaf","value":null}` + "`" + `), -32602, "Missing required field: value", "missing_field", ""},
			{"invalid branch", request(` + "`" + `{"type":"Leaf","value":{}}` + "`" + `), -32602, "Missing required field: name", "missing_field", ""},
		}
		for _, tc := range cases {
			name := method + " " + tc.name
			resp, err := hs.Client().Post(hs.URL+"/rpc", "application/json", strings.NewReader(tc.body))
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
				t.Errorf("%s: status %d (%s)", name, resp.StatusCode, raw)
			}
			var envelope struct {
				Result *struct {
					Type string ` + "`" + `json:"type"` + "`" + `
				} ` + "`" + `json:"result"` + "`" + `
				Error *struct {
					Code    int    ` + "`" + `json:"code"` + "`" + `
					Message string ` + "`" + `json:"message"` + "`" + `
					Data    struct {
						Name string ` + "`" + `json:"name"` + "`" + `
					} ` + "`" + `json:"data"` + "`" + `
				} ` + "`" + `json:"error"` + "`" + `
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Errorf("%s: decode response %q: %v", name, raw, err)
				continue
			}
			seen := svc.take()
			if tc.kind == "" {
				switch {
				case envelope.Error == nil:
					t.Errorf("%s: no error in %s", name, raw)
				case envelope.Error.Code != tc.code || envelope.Error.Message != tc.message || envelope.Error.Data.Name != tc.errName:
					t.Errorf("%s: error (%d, %q, %q), want (%d, %q, %q)", name, envelope.Error.Code, envelope.Error.Message, envelope.Error.Data.Name, tc.code, tc.message, tc.errName)
				}
				if len(seen) != 0 {
					t.Errorf("%s: service invoked with %v", name, seen)
				}
				continue
			}
			if envelope.Error != nil || envelope.Result == nil || envelope.Result.Type != tc.kind {
				t.Errorf("%s: response %s", name, raw)
			}
			if !reflect.DeepEqual(seen, []string{tc.kind}) {
				t.Errorf("%s: service received %v, want %s", name, seen, tc.kind)
			}
		}
	}
}

// TestClientDecodesUnionResults serves raw JSON-RPC results to the generated
// client and checks that valid unions decode to the right branch and invalid
// ones fail.
func TestClientDecodesUnionResults(t *testing.T) {
	var result string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID any ` + "`" + `json:"id"` + "`" + `
		}
		if err := json.UnmarshalRead(r.Body, &req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		id, err := json.Marshal(req.ID)
		if err != nil {
			t.Errorf("encode id: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, ` + "`" + `{"jsonrpc":"2.0","id":` + "`" + `+string(id)+` + "`" + `,"result":` + "`" + `+result+"}"); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer hs.Close()
	c := rpcclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx := context.Background()
	payload := rpc.NewLeafOrOtherOther(&rpc.Other{})
	choice := rpc.NewChoiceOther(&rpc.Other{})
	endpoints := []struct {
		name string
		call func() (any, error)
		kind func(any) string
	}{
		{"echo", func() (any, error) { return c.Echo()(ctx, &payload) }, func(v any) string { return string(v.(*rpc.LeafOrOther).Kind()) }},
		{"named", func() (any, error) { return c.Named()(ctx, &choice) }, func(v any) string { return string(v.(*rpc.Choice).Kind()) }},
	}
	cases := []struct {
		name string
		body string
		kind string
	}{
		{"leaf", ` + "`" + `{"type":"Leaf","value":{"name":"x"}}` + "`" + `, "Leaf"},
		{"other", ` + "`" + `{"type":"Other","value":{"count":1}}` + "`" + `, "Other"},
		{"unknown branch", ` + "`" + `{"type":"Nope","value":{}}` + "`" + `, ""},
		{"no type", "{}", ""},
		{"missing value", ` + "`" + `{"type":"Leaf"}` + "`" + `, ""},
		{"invalid branch", ` + "`" + `{"type":"Leaf","value":{}}` + "`" + `, ""},
	}
	for _, e := range endpoints {
		for _, tc := range cases {
			result = tc.body
			res, err := e.call()
			if tc.kind == "" {
				if err == nil {
					t.Errorf("%s %s: decoded %#v, want an error", e.name, tc.name, res)
				}
				continue
			}
			if err != nil {
				t.Errorf("%s %s: %v", e.name, tc.name, err)
				continue
			}
			if got := e.kind(res); got != tc.kind {
				t.Errorf("%s %s: kind %s, want %s", e.name, tc.name, got, tc.kind)
			}
		}
	}
}

func TestSSEUnionResultRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	eventsserver.Mount(mux, eventsserver.New(events.NewEndpoints(eventsService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := eventsclient.NewClient("http", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, p := range []events.LeafOrOther{
		events.NewLeafOrOtherLeaf(&events.Leaf{Name: "a"}),
		events.NewLeafOrOtherOther(&events.Other{}),
	} {
		res, err := c.Watch()(ctx, &p)
		if err != nil {
			t.Fatalf("watch %s: %v", p.Kind(), err)
		}
		stream, ok := res.(*eventsclient.WatchClientStream)
		if !ok {
			t.Fatalf("watch %s: stream type %T", p.Kind(), res)
		}
		want := []events.Choice{
			events.NewChoiceLeaf(&events.Leaf{Name: string(p.Kind())}),
			events.NewChoiceOther(&events.Other{Count: ptr(1)}),
		}
		for i, w := range want {
			got, err := stream.Recv(ctx)
			if err != nil {
				t.Fatalf("watch %s[%d]: %v", p.Kind(), i, err)
			}
			if !reflect.DeepEqual(*got, w) {
				t.Errorf("watch %s[%d]: got %#v, want %#v", p.Kind(), i, got, w)
			}
		}
		if _, err := stream.Recv(ctx); !errors.Is(err, io.EOF) {
			t.Errorf("watch %s: expected io.EOF, got %v", p.Kind(), err)
		}
		if err := stream.Close(); err != nil {
			t.Errorf("watch %s: close: %v", p.Kind(), err)
		}
	}
}

func TestWebSocketUnionResultRoundTrip(t *testing.T) {
	svc := socketService{}
	mux := loomhttp.NewMuxer()
	socketserver.Mount(mux, socketserver.New(svc.HandleStream, socket.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := socketclient.NewClient("ws", host(hs), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	defer func() {
		if err := c.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := c.Talk()(ctx, nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(*socketclient.TalkClientStream)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}
	cases := []struct {
		sent *socket.Leaf
		want socket.Choice
	}{
		{&socket.Leaf{Name: "a"}, socket.NewChoiceLeaf(&socket.Leaf{Name: "a"})},
		{&socket.Leaf{Name: "other"}, socket.NewChoiceOther(&socket.Other{})},
	}
	for _, tc := range cases {
		if err := stream.SendWithContext(ctx, tc.sent); err != nil {
			t.Fatalf("send %s: %v", tc.sent.Name, err)
		}
		got, err := stream.RecvWithContext(ctx)
		if err != nil {
			t.Fatalf("recv %s: %v", tc.sent.Name, err)
		}
		if got == nil || !reflect.DeepEqual(*got, tc.want) {
			t.Errorf("recv %s: got %#v, want %#v", tc.sent.Name, got, tc.want)
		}
	}
}
`
