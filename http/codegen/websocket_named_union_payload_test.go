package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestWebSocketNamedUnionStreamingPayload checks that the WebSocket
// streaming body of a named union gets the "<endpoint>StreamingBody" union
// type, like an unnamed union streaming body, that the payload constructor
// keeps the name of the union and that the server Recv validates the union
// branches. Without the name the union collided with the unions nested in
// the other bodies of the service.
func TestWebSocketNamedUnionStreamingPayload(t *testing.T) {
	root := RunHTTPDSL(t, webSocketNamedUnionDSL)
	services := CreateHTTPServices(root)
	for _, name := range []string{"echo", "collect"} {
		t.Run(name, func(t *testing.T) {
			data := services.Get(name)
			require.NotNil(t, data)
			var talk *EndpointData
			for _, endpoint := range data.Endpoints {
				if endpoint.Method.Name == "talk" {
					talk = endpoint
				}
			}
			require.NotNil(t, talk)
			server, client := talk.ServerWebSocket, talk.ClientWebSocket
			require.NotNil(t, server)
			require.NotNil(t, client)
			require.Equal(t, "talkStreamingBody", server.Payload.Name)
			require.Equal(t, "talkStreamingBody", client.Payload.Name)
			require.NotNil(t, server.Payload.Init)
			require.Equal(t, "NewTalkChoice", server.Payload.Init.Name)
			require.NotEmpty(t, serverWebSocketPayloadValidation(server))
		})
	}
}

// TestWebSocketNamedUnionStreamingPayloadGeneratedIntegration generates the
// services, compiles and vets them, then streams every union branch through
// the generated client and server and checks that the server rejects a
// branch that fails validation.
func TestWebSocketNamedUnionStreamingPayloadGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, webSocketNamedUnionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/namedunionws", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "named_union_ws_test.go"), []byte(webSocketNamedUnionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func webSocketNamedUnionDSL() {
	leaf := Type("Leaf", func() {
		Attribute("name", String)
		Required("name")
	})
	other := Type("Other", func() {
		Attribute("count", Int)
	})
	choice := Type("Choice", OneOf(leaf, other))
	holder := Type("Holder", func() {
		Attribute("choice", choice)
	})
	Service("echo", func() {
		Method("talk", func() {
			StreamingPayload(choice)
			StreamingResult(holder)
			HTTP(func() {
				GET("/echo/talk")
			})
		})
		Method("watch", func() {
			StreamingResult(choice)
			HTTP(func() {
				GET("/echo/watch")
			})
		})
	})
	Service("collect", func() {
		Method("talk", func() {
			StreamingPayload(choice)
			Result(String)
			HTTP(func() {
				GET("/collect/talk")
			})
		})
		Method("put", func() {
			Payload(holder)
			Result(holder)
			HTTP(func() {
				POST("/collect/put")
			})
		})
	})
}

const webSocketNamedUnionHarness = `package namedunionws

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

	echo "example.com/namedunionws/gen/echo"
	echoclient "example.com/namedunionws/gen/http/echo/client"
	echoserver "example.com/namedunionws/gen/http/echo/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type echoService struct {
	done chan error
}

func (s *echoService) Talk(ctx context.Context, stream echo.TalkServerStream) error {
	err := s.talk(ctx, stream)
	s.done <- err
	return err
}

func (s *echoService) talk(ctx context.Context, stream echo.TalkServerStream) error {
	for {
		msg, err := stream.RecvWithContext(ctx)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.SendWithContext(ctx, &echo.Holder{Choice: msg}); err != nil {
			return err
		}
	}
}

func (s *echoService) Watch(_ context.Context, stream echo.WatchServerStream) error {
	leaf := echo.NewChoiceLeaf(&echo.Leaf{Name: "watched"})
	if err := stream.Send(&leaf); err != nil {
		return err
	}
	return stream.Close()
}

func ptr[T any](v T) *T { return &v }

func newServer(t *testing.T) (*echoService, *echoclient.Client, string) {
	t.Helper()
	svc := &echoService{done: make(chan error, 1)}
	mux := loomhttp.NewMuxer()
	echoserver.Mount(mux, echoserver.New(echo.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	host := strings.TrimPrefix(hs.URL, "http://")
	c := echoclient.NewClient("ws", host, hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	return svc, c, host
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

func TestTalkRoundTrip(t *testing.T) {
	svc, c, _ := newServer(t)
	raw, err := c.Talk()(context.Background(), nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(echo.TalkClientStream)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}
	payloads := []echo.Choice{
		echo.NewChoiceLeaf(&echo.Leaf{Name: "a"}),
		echo.NewChoiceOther(&echo.Other{Count: ptr(3)}),
		echo.NewChoiceOther(&echo.Other{}),
	}
	for _, p := range payloads {
		if err := stream.Send(&p); err != nil {
			t.Fatalf("send %s: %v", p.Kind(), err)
		}
		got, err := stream.Recv()
		if err != nil {
			t.Fatalf("recv %s: %v", p.Kind(), err)
		}
		if got.Choice == nil || !reflect.DeepEqual(*got.Choice, p) {
			t.Errorf("%s: got %+v, want %+v", p.Kind(), got.Choice, p)
		}
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := wait(t, svc.done); err != nil {
		t.Errorf("server recv: %v", err)
	}
}

func TestTalkRejectsInvalidBranch(t *testing.T) {
	svc, _, host := newServer(t)
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+host+"/echo/talk", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(` + "`" + `{"type":"Leaf","value":{}}` + "`" + `)); err != nil {
		t.Fatalf("write: %v", err)
	}
	err = wait(t, svc.done)
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("got %v, want a missing name error", err)
	}
}

func TestWatch(t *testing.T) {
	_, c, _ := newServer(t)
	raw, err := c.Watch()(context.Background(), nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(echo.WatchClientStream)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}
	got, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	want := echo.NewChoiceLeaf(&echo.Leaf{Name: "watched"})
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("got %+v, want %+v", *got, want)
	}
}
`
