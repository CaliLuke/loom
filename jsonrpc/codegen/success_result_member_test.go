package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCSuccessResultMemberGeneratedModule covers the result member of
// the success responses that generated JSON-RPC servers send over HTTP, as
// the final response of an SSE stream and over WebSocket. JSON-RPC 2.0
// requires the member in every success response: it is null for a method
// without a result and keeps empty results such as "", [] and {}. Error
// responses have no result member, and requests without an ID get no
// response.
func TestJSONRPCSuccessResultMemberGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcSuccessResultMemberDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcresultmember", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "result_member_test.go"), []byte(jsonRPCSuccessResultMemberHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

func jsonrpcSuccessResultMemberDSL() {
	dsl.API("resultmember", func() {
		dsl.JSONRPC(func() {})
	})
	info := dsl.Type("Info", func() {
		dsl.Attribute("note", dsl.String)
	})
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("touch", func() {
			dsl.Payload(dsl.String)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("name", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.String)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("names", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.ArrayOf(dsl.String))
			dsl.JSONRPC(func() {})
		})
		dsl.Method("detail", func() {
			dsl.Payload(dsl.String)
			dsl.Result(info)
			dsl.JSONRPC(func() {})
		})
	})
	dsl.Service("feed", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/feed")
		})
		dsl.Method("watch", func() {
			dsl.Payload(dsl.String)
			dsl.StreamingResult(info)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
		dsl.Method("words", func() {
			dsl.Payload(dsl.String)
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
	dsl.Service("sock", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("push", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("echo", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.StreamingResult(dsl.String)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCSuccessResultMemberHarness = `package jsonrpcresultmember_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	feed "example.com/jsonrpcresultmember/gen/feed"
	feedclient "example.com/jsonrpcresultmember/gen/jsonrpc/feed/client"
	feedserver "example.com/jsonrpcresultmember/gen/jsonrpc/feed/server"
	rpcclient "example.com/jsonrpcresultmember/gen/jsonrpc/rpc/client"
	rpcserver "example.com/jsonrpcresultmember/gen/jsonrpc/rpc/server"
	sockclient "example.com/jsonrpcresultmember/gen/jsonrpc/sock/client"
	sockserver "example.com/jsonrpcresultmember/gen/jsonrpc/sock/server"
	rpc "example.com/jsonrpcresultmember/gen/rpc"
	sock "example.com/jsonrpcresultmember/gen/sock"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

// q returns s with its single quotes replaced by double quotes, so that the
// JSON documents of the cases read without escapes.
func q(s string) string {
	return strings.ReplaceAll(s, "'", "\"")
}

// Every service fails the payload "fail" and otherwise answers with an empty
// value of its result type.

func rejected() error {
	return loom.PermanentError("rejected", "payload rejected")
}

type rpcService struct{}

func (rpcService) Touch(_ context.Context, p string) error {
	if p == "fail" {
		return rejected()
	}
	return nil
}

func (rpcService) Name(_ context.Context, p string) (string, error) {
	if p == "fail" {
		return "", rejected()
	}
	return "", nil
}

func (rpcService) Names(_ context.Context, p string) ([]string, error) {
	if p == "fail" {
		return nil, rejected()
	}
	return []string{}, nil
}

func (rpcService) Detail(_ context.Context, p string) (*rpc.Info, error) {
	if p == "fail" {
		return nil, rejected()
	}
	return &rpc.Info{}, nil
}

type feedService struct{}

func (feedService) Watch(ctx context.Context, p string, st feed.WatchServerStream) error {
	note := "first"
	if err := st.Send(ctx, &feed.Info{Note: &note}); err != nil {
		return err
	}
	if p == "fail" {
		return rejected()
	}
	return st.SendAndClose(ctx, &feed.Info{})
}

func (feedService) Words(ctx context.Context, p string, st feed.WordsServerStream) error {
	if err := st.Send(ctx, "first"); err != nil {
		return err
	}
	if p == "fail" {
		return rejected()
	}
	return st.SendAndClose(ctx, "")
}

type sockService struct{}

func (sockService) HandleStream(ctx context.Context, stream sock.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (sockService) Push(_ context.Context, p string) error {
	if p == "fail" {
		return rejected()
	}
	return nil
}

func (sockService) Echo(ctx context.Context, p string, st sock.EchoServerStream) error {
	if p == "fail" {
		return rejected()
	}
	return st.SendResponse(ctx, p)
}

func serve(t *testing.T) string {
	t.Helper()
	mux := loomhttp.NewMuxer()
	// The handler also reports the endpoint errors of notifications, which
	// the cases check through the absence of a response.
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	rpcserver.Mount(mux, rpcserver.New(rpc.NewEndpoints(rpcService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	feedserver.Mount(mux, feedserver.New(feed.NewEndpoints(feedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	svc := sockService{}
	sockserver.Mount(mux, sockserver.New(svc.HandleStream, sock.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return hs.URL
}

func post(t *testing.T, url, accept, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", body, err)
	}
	return resp
}

// requireError checks that data is an error response to id without a result
// member.
func requireError(t *testing.T, name, id string, data []byte) {
	t.Helper()
	var members map[string]jsontext.Value
	if err := json.Unmarshal(data, &members); err != nil {
		t.Errorf("%s: decode %s: %v", name, data, err)
		return
	}
	if _, ok := members["result"]; ok {
		t.Errorf("%s: error response has a result member: %s", name, data)
	}
	if _, ok := members["error"]; !ok || string(members["id"]) != id {
		t.Errorf("%s: want an error response to %s, got %s", name, id, data)
	}
}

func TestHTTPResultMember(t *testing.T) {
	url := serve(t) + "/rpc"
	cases := []struct {
		name, body, want string
	}{
		{"no result", q("{'jsonrpc':'2.0','id':1,'method':'touch','params':'x'}"), q("{'jsonrpc':'2.0','result':null,'id':1}")},
		{"empty string", q("{'jsonrpc':'2.0','id':'s','method':'name','params':'x'}"), q("{'jsonrpc':'2.0','result':'','id':'s'}")},
		{"empty array", q("{'jsonrpc':'2.0','id':3,'method':'names','params':'x'}"), q("{'jsonrpc':'2.0','result':[],'id':3}")},
		{"empty object", q("{'jsonrpc':'2.0','id':4,'method':'detail','params':'x'}"), q("{'jsonrpc':'2.0','result':{},'id':4}")},
		{"null id", q("{'jsonrpc':'2.0','id':null,'method':'touch','params':'x'}"), q("{'jsonrpc':'2.0','result':null,'id':null}")},
		{"notification", q("{'jsonrpc':'2.0','method':'touch','params':'x'}"), ""},
		{"failed notification", q("{'jsonrpc':'2.0','method':'touch','params':'fail'}"), ""},
		{"batch", q("[{'jsonrpc':'2.0','id':5,'method':'touch','params':'x'},{'jsonrpc':'2.0','method':'touch','params':'x'},{'jsonrpc':'2.0','id':6,'method':'name','params':'x'}]"), q("[{'jsonrpc':'2.0','result':null,'id':5},{'jsonrpc':'2.0','result':'','id':6}]")},
	}
	for _, c := range cases {
		resp := post(t, url, "", c.body)
		data, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("%s: close: %v", c.name, closeErr)
		}
		if err != nil {
			t.Errorf("%s: read: %v", c.name, err)
			continue
		}
		if got := strings.TrimSpace(string(data)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	for _, method := range []string{"touch", "name", "names", "detail"} {
		resp := post(t, url, "", q("{'jsonrpc':'2.0','id':9,'method':'")+method+q("','params':'fail'}"))
		data, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("%s: close: %v", method, closeErr)
		}
		if err != nil {
			t.Errorf("%s: read: %v", method, err)
			continue
		}
		requireError(t, method, "9", data)
	}
}

func TestSSEFinalResultMember(t *testing.T) {
	url := serve(t) + "/feed"
	events := func(t *testing.T, body string) []loomhttp.SSEEvent {
		t.Helper()
		resp := post(t, url, "text/event-stream", body)
		events, err := loomhttp.ParseSSEStream(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Errorf("close: %v", closeErr)
		}
		if err != nil {
			t.Fatalf("parse %s: %v", body, err)
		}
		return events
	}
	cases := []struct {
		name, body, notification, final string
	}{
		{"empty object", q("{'jsonrpc':'2.0','id':'w','method':'watch','params':'x'}"), q("{'note':'first'}"), q("{'jsonrpc':'2.0','result':{},'id':'w'}")},
		{"empty string", q("{'jsonrpc':'2.0','id':2,'method':'words','params':'x'}"), q("'first'"), q("{'jsonrpc':'2.0','result':'','id':2}")},
		{"object notification", q("{'jsonrpc':'2.0','method':'watch','params':'x'}"), q("{'note':'first'}"), ""},
		{"string notification", q("{'jsonrpc':'2.0','method':'words','params':'x'}"), q("'first'"), ""},
		{"failed notification", q("{'jsonrpc':'2.0','method':'words','params':'fail'}"), q("'first'"), ""},
	}
	for _, c := range cases {
		got := events(t, c.body)
		want := 1
		if c.final != "" {
			want = 2
		}
		if len(got) != want {
			t.Errorf("%s: got %d events %#v, want %d", c.name, len(got), got, want)
			continue
		}
		var notification map[string]jsontext.Value
		if err := json.Unmarshal([]byte(got[0].Data), &notification); err != nil || string(notification["params"]) != c.notification {
			t.Errorf("%s: got notification %s (%v), want params %s", c.name, got[0].Data, err, c.notification)
		}
		if c.final != "" && got[1].Data != c.final {
			t.Errorf("%s: got final response %q, want %q", c.name, got[1].Data, c.final)
		}
	}
	for _, method := range []string{"watch", "words"} {
		got := events(t, q("{'jsonrpc':'2.0','id':9,'method':'")+method+q("','params':'fail'}"))
		if len(got) != 2 {
			t.Errorf("%s: got %d events %#v, want 2", method, len(got), got)
			continue
		}
		requireError(t, method, "9", []byte(got[1].Data))
	}
}

// TestClientsDecodeEmptyResults checks that the generated clients decode the
// success responses without a result and with empty results.
func TestClientsDecodeEmptyResults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	host := strings.TrimPrefix(serve(t), "http://")
	rc := rpcclient.NewClient("http", host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	cases := []struct {
		name     string
		endpoint loom.Endpoint
		want     any
	}{
		{"touch", rc.Touch(), nil},
		{"name", rc.Name(), ""},
		{"names", rc.Names(), []string{}},
		{"detail", rc.Detail(), &rpc.Info{}},
	}
	for _, c := range cases {
		got, err := c.endpoint(ctx, "x")
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got (%#v, %v), want %#v", c.name, got, err, c.want)
		}
	}

	fc := feedclient.NewClient("http", host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	res, err := fc.Words()(ctx, "x")
	if err != nil {
		t.Fatalf("words: %v", err)
	}
	words := res.(*feedclient.WordsClientStream)
	for _, want := range []string{"first", ""} {
		if got, err := words.Recv(ctx); err != nil || got != want {
			t.Errorf("words: got (%q, %v), want %q", got, err, want)
		}
	}
	if err := words.Close(); err != nil {
		t.Errorf("words: close: %v", err)
	}
	res, err = fc.Watch()(ctx, "x")
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	watch := res.(*feedclient.WatchClientStream)
	if _, err := watch.Recv(ctx); err != nil {
		t.Errorf("watch: first: %v", err)
	}
	if got, err := watch.Recv(ctx); err != nil || got == nil || got.Note != nil {
		t.Errorf("watch: got (%#v, %v), want an empty Info", got, err)
	}
	if err := watch.Close(); err != nil {
		t.Errorf("watch: close: %v", err)
	}

	sc := sockclient.NewClient("ws", host, http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	defer func() {
		if err := sc.Close(); err != nil {
			t.Errorf("close sock client: %v", err)
		}
	}()
	res, err = sc.Echo()(ctx, nil)
	if err != nil {
		t.Fatalf("echo: %v", err)
	}
	echo := res.(*sockclient.EchoClientStream)
	if err := echo.SendWithContext(ctx, ""); err != nil {
		t.Fatalf("echo: send: %v", err)
	}
	if got, err := echo.RecvWithContext(ctx); err != nil || got != "" {
		t.Errorf("echo: got (%q, %v), want the empty string", got, err)
	}
	if err := echo.Close(); err != nil {
		t.Errorf("echo: close: %v", err)
	}
}

func TestWebSocketResultMember(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(serve(t), "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close handshake body: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, websocket.ErrCloseSent) {
			t.Errorf("close conn: %v", err)
		}
	}()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	// Each request is preceded by the same request without an ID, which
	// gets no response, so the next message answers the request.
	exchange := func(method, params, id string) []byte {
		t.Helper()
		for _, req := range []string{
			q("{'jsonrpc':'2.0','method':'") + method + q("','params':'") + params + q("'}"),
			q("{'jsonrpc':'2.0','id':") + id + q(",'method':'") + method + q("','params':'") + params + q("'}"),
		} {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(req)); err != nil {
				t.Fatalf("write %s: %v", req, err)
			}
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read the response to %s %s: %v", method, id, err)
		}
		return data
	}
	cases := []struct {
		method, params, id, want string
	}{
		{"push", "x", "1", q("{'jsonrpc':'2.0','result':null,'id':1}")},
		{"echo", "", "\"e\"", q("{'jsonrpc':'2.0','result':'','id':'e'}")},
		{"push", "x", "null", q("{'jsonrpc':'2.0','result':null,'id':null}")},
	}
	for _, c := range cases {
		if got := strings.TrimSpace(string(exchange(c.method, c.params, c.id))); got != c.want {
			t.Errorf("%s %s: got %q, want %q", c.method, c.id, got, c.want)
		}
	}
	for _, method := range []string{"push", "echo"} {
		requireError(t, method, "9", exchange(method, "fail", "9"))
	}
}
`
