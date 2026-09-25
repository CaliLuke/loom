package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestWebSocketNamedPayloadRecv checks the server Recv of WebSocket streaming
// payloads of named non-object types. The payload constructor takes the body
// of a named primitive type by value and the body of a named collection as
// declared, and a body is validated inline without calling a Validate
// function of the named type.
func TestWebSocketNamedPayloadRecv(t *testing.T) {
	cases := []struct {
		service  string
		contains []string
		excludes []string
	}{
		{service: "token", contains: []string{"msg *string", "return NewTalkToken(*msg), nil"}, excludes: []string{"ValidateToken("}},
		{service: "count", contains: []string{"msg *int", "return NewTalkN(*msg), nil"}, excludes: []string{"ValidateN("}},
		{service: "param", contains: []string{"return NewTalkToken(*msg), nil"}},
		{service: "list", contains: []string{"body []loom.Nullable[*Item]", "return NewTalkL(body), nil"}, excludes: []string{"ValidateL("}},
		{service: "index", contains: []string{"body map[string]loom.Nullable[*Item]", "return NewTalkM(body), nil"}, excludes: []string{"ValidateM("}},
	}
	root := RunHTTPDSL(t, webSocketNamedPayloadDSL)
	services := CreateHTTPServices(root)
	for _, c := range cases {
		t.Run(c.service, func(t *testing.T) {
			data := services.Get(c.service)
			require.NotNil(t, data)
			require.Len(t, data.Endpoints, 1)
			ws := data.Endpoints[0].ServerWebSocket
			require.NotNil(t, ws)
			code := codegen.SectionCode(t, websocketRecvSection(ws))
			for _, s := range c.contains {
				require.Contains(t, code, s)
			}
			for _, s := range c.excludes {
				require.NotContains(t, code, s)
			}
		})
	}
}

// TestWebSocketNamedPayloadGeneratedIntegration generates services that
// stream named primitive, array and map payloads, compiles and vets them,
// then round-trips the payloads through the generated client and server and
// checks that the server rejects invalid messages.
func TestWebSocketNamedPayloadGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, webSocketNamedPayloadDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/namedpayloadws", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "named_payload_ws_test.go"), []byte(webSocketNamedPayloadHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func webSocketNamedPayloadDSL() {
	item := Type("Item", func() {
		Attribute("name", String, func() {
			MinLength(1)
		})
		Required("name")
	})
	located := Type("Located", func() {
		Meta("struct:pkg:path", "types")
		Attribute("name", String)
	})
	token := Type("Token", String, func() {
		MinLength(1)
	})
	n := Type("N", Int, func() {
		Minimum(1)
	})
	list := Type("L", ArrayOf(item))
	locatedList := Type("LocatedList", ArrayOf(located))
	index := Type("M", MapOf(String, item))
	stream := func(service, path string, method func()) {
		Service(service, func() {
			Method("talk", func() {
				method()
				HTTP(func() {
					GET(path)
					if service == "param" {
						Param("id")
					}
				})
			})
		})
	}
	stream("token", "/token", func() {
		StreamingPayload(token)
		StreamingResult(token)
	})
	stream("count", "/count", func() {
		StreamingPayload(n)
		Result(n)
	})
	stream("param", "/param", func() {
		Payload(func() {
			Attribute("id", String)
		})
		StreamingPayload(token)
		Result(String)
	})
	stream("list", "/list", func() {
		StreamingPayload(list)
		StreamingResult(list)
	})
	stream("index", "/index", func() {
		StreamingPayload(index)
		StreamingResult(index)
	})
	stream("located", "/located", func() {
		StreamingPayload(locatedList)
		Result(String)
	})
}

const webSocketNamedPayloadHarness = `package namedpayloadws

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	count "example.com/namedpayloadws/gen/count"
	countclient "example.com/namedpayloadws/gen/http/count/client"
	countserver "example.com/namedpayloadws/gen/http/count/server"
	indexclient "example.com/namedpayloadws/gen/http/index/client"
	indexserver "example.com/namedpayloadws/gen/http/index/server"
	listclient "example.com/namedpayloadws/gen/http/list/client"
	listserver "example.com/namedpayloadws/gen/http/list/server"
	tokenclient "example.com/namedpayloadws/gen/http/token/client"
	tokenserver "example.com/namedpayloadws/gen/http/token/server"
	index "example.com/namedpayloadws/gen/index"
	list "example.com/namedpayloadws/gen/list"
	token "example.com/namedpayloadws/gen/token"
	loomhttp "github.com/CaliLuke/loom/http"
)

type bidi[T any] interface {
	RecvWithContext(context.Context) (T, error)
	SendWithContext(context.Context, T) error
}

func echo[T any](ctx context.Context, stream bidi[T], done chan error) error {
	err := func() error {
		for {
			msg, err := stream.RecvWithContext(ctx)
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			if err := stream.SendWithContext(ctx, msg); err != nil {
				return err
			}
		}
	}()
	done <- err
	return err
}

type tokenService struct{ done chan error }

func (s *tokenService) Talk(ctx context.Context, stream token.TalkServerStream) error {
	return echo[token.Token](ctx, stream, s.done)
}

type listService struct{ done chan error }

func (s *listService) Talk(ctx context.Context, stream list.TalkServerStream) error {
	return echo[list.L](ctx, stream, s.done)
}

type indexService struct{ done chan error }

func (s *indexService) Talk(ctx context.Context, stream index.TalkServerStream) error {
	return echo[index.M](ctx, stream, s.done)
}

type countService struct{ done chan error }

func (s *countService) Talk(ctx context.Context, stream count.TalkServerStream) error {
	err := func() error {
		var sum count.N
		for {
			n, err := stream.RecvWithContext(ctx)
			if errors.Is(err, io.EOF) {
				return stream.SendAndCloseWithContext(ctx, sum)
			}
			if err != nil {
				return err
			}
			sum += n
		}
	}()
	s.done <- err
	return err
}

type server struct {
	host  string
	token *tokenService
	list  *listService
	index *indexService
	count *countService
}

func serve(t *testing.T) *server {
	t.Helper()
	s := &server{
		token: &tokenService{done: make(chan error, 1)},
		list:  &listService{done: make(chan error, 1)},
		index: &indexService{done: make(chan error, 1)},
		count: &countService{done: make(chan error, 1)},
	}
	mux := loomhttp.NewMuxer()
	dec, enc := loomhttp.RequestDecoder, loomhttp.ResponseEncoder
	up := &websocket.Upgrader{}
	tokenserver.Mount(mux, tokenserver.New(token.NewEndpoints(s.token), mux, dec, enc, nil, nil, up, nil))
	listserver.Mount(mux, listserver.New(list.NewEndpoints(s.list), mux, dec, enc, nil, nil, up, nil))
	indexserver.Mount(mux, indexserver.New(index.NewEndpoints(s.index), mux, dec, enc, nil, nil, up, nil))
	countserver.Mount(mux, countserver.New(count.NewEndpoints(s.count), mux, dec, enc, nil, nil, up, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	s.host = strings.TrimPrefix(hs.URL, "http://")
	return s
}

func wait(t *testing.T, done chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("server did not finish the stream")
		return nil
	}
}

func open[T any](t *testing.T, endpoint func(context.Context, any) (any, error)) T {
	t.Helper()
	raw, err := endpoint(context.Background(), nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(T)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}
	return stream
}

func TestRoundTrip(t *testing.T) {
	s := serve(t)
	enc, dec := loomhttp.RequestEncoder, loomhttp.ResponseDecoder

	tc := tokenclient.NewClient("ws", s.host, nil, enc, dec, false, websocket.DefaultDialer, nil)
	ts := open[token.TalkClientStream](t, tc.Talk())
	for _, v := range []token.Token{"a", "héllo"} {
		if err := ts.Send(v); err != nil {
			t.Fatalf("token send: %v", err)
		}
		got, err := ts.Recv()
		if err != nil || got != v {
			t.Errorf("token: got %q, %v, want %q", got, err, v)
		}
	}
	if err := ts.Close(); err != nil {
		t.Fatalf("token close: %v", err)
	}
	if err := wait(t, s.token.done); err != nil {
		t.Errorf("token server: %v", err)
	}

	lc := listclient.NewClient("ws", s.host, nil, enc, dec, false, websocket.DefaultDialer, nil)
	ls := open[list.TalkClientStream](t, lc.Talk())
	want := list.L{{Name: "a"}, {Name: "b"}}
	if err := ls.Send(want); err != nil {
		t.Fatalf("list send: %v", err)
	}
	if got, err := ls.Recv(); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("list: got %+v, %v, want %+v", got, err, want)
	}
	if err := ls.Close(); err != nil {
		t.Fatalf("list close: %v", err)
	}
	if err := wait(t, s.list.done); err != nil {
		t.Errorf("list server: %v", err)
	}

	ic := indexclient.NewClient("ws", s.host, nil, enc, dec, false, websocket.DefaultDialer, nil)
	is := open[index.TalkClientStream](t, ic.Talk())
	wantIndex := index.M{"k": {Name: "a"}}
	if err := is.Send(wantIndex); err != nil {
		t.Fatalf("index send: %v", err)
	}
	if got, err := is.Recv(); err != nil || !reflect.DeepEqual(got, wantIndex) {
		t.Errorf("index: got %+v, %v, want %+v", got, err, wantIndex)
	}
	if err := is.Close(); err != nil {
		t.Fatalf("index close: %v", err)
	}
	if err := wait(t, s.index.done); err != nil {
		t.Errorf("index server: %v", err)
	}

	cc := countclient.NewClient("ws", s.host, nil, enc, dec, false, websocket.DefaultDialer, nil)
	cs := open[count.TalkClientStream](t, cc.Talk())
	for _, v := range []count.N{1, 2} {
		if err := cs.Send(v); err != nil {
			t.Fatalf("count send: %v", err)
		}
	}
	if got, err := cs.CloseAndRecv(); err != nil || got != 3 {
		t.Errorf("count: got %d, %v, want 3", got, err)
	}
	if err := wait(t, s.count.done); err != nil {
		t.Errorf("count server: %v", err)
	}
}

func TestRejectsInvalidMessages(t *testing.T) {
	cases := []struct {
		path, message, want string
		done                func(*server) chan error
	}{
		{"/token", ` + "`" + `""` + "`" + `, "length", func(s *server) chan error { return s.token.done }},
		{"/count", ` + "`" + `0` + "`" + `, "body", func(s *server) chan error { return s.count.done }},
		{"/list", ` + "`" + `[{}]` + "`" + `, "name", func(s *server) chan error { return s.list.done }},
		{"/list", ` + "`" + `[{"name":""}]` + "`" + `, "length", func(s *server) chan error { return s.list.done }},
		{"/list", ` + "`" + `[null]` + "`" + `, "body[0]", func(s *server) chan error { return s.list.done }},
		{"/index", ` + "`" + `{"k":{}}` + "`" + `, "name", func(s *server) chan error { return s.index.done }},
	}
	for _, c := range cases {
		t.Run(c.path+" "+c.message, func(t *testing.T) {
			s := serve(t)
			conn, _, err := websocket.DefaultDialer.Dial("ws://"+s.host+c.path, nil)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close()
			if err := conn.WriteMessage(websocket.TextMessage, []byte(c.message)); err != nil {
				t.Fatalf("write: %v", err)
			}
			err = wait(t, c.done(s))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want an error containing %q", err, c.want)
			}
		})
	}
}
`
