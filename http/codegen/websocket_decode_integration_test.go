package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestWebSocketRecvDecodingGeneratedIntegration generates a bidirectional
// WebSocket service, compiles it in a temporary module, and sends empty,
// whitespace-only, malformed, truncated, and valid messages to both the
// generated server Recv and the generated client Recv.
func TestWebSocketRecvDecodingGeneratedIntegration(t *testing.T) {
	const modulePath = "example.com/websocketdecode"

	root := RunHTTPDSL(t, webSocketDecodeDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "websocket_decode_test.go"), []byte(webSocketDecodeHarness), 0o600))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

func webSocketDecodeDSL() {
	var Message = Type("Message", func() {
		Attribute("text", String)
		OneOf("value", func() {
			Attribute("name", String)
			Attribute("count", Int)
		})
		Attribute("extra", Any)
		Required("text")
	})
	Service("chat", func() {
		Method("talk", func() {
			StreamingPayload(Message)
			StreamingResult(Message)
			HTTP(func() {
				GET("/chat")
			})
		})
	})
}

const webSocketDecodeHarness = `package websocketdecode

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	chat "example.com/websocketdecode/gen/chat"
	chatclient "example.com/websocketdecode/gen/http/chat/client"
	chatserver "example.com/websocketdecode/gen/http/chat/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type observation struct {
	msg *chat.Message
	err error
}

// frames are sent in order; each one is followed by one Recv call.
var frames = []string{
	"",
	" \n\t ",
	` + "`" + `{"text":x}` + "`" + `,
	` + "`" + `{"text":"a"` + "`" + `,
	` + "`" + `{"text":"hi","value":{"type":"count","value":3},"extra":{"k":[1,true]}}` + "`" + `,
}

type talkService struct {
	observations chan observation
}

func (s *talkService) Talk(ctx context.Context, stream chat.TalkServerStream) error {
	// One Recv per frame, plus one for the trailing null end-of-stream frame.
	for range len(frames) + 1 {
		msg, err := stream.RecvWithContext(ctx)
		s.observations <- observation{msg: msg, err: err}
	}
	return nil
}

func TestGeneratedServerRecvDecoding(t *testing.T) {
	svc := &talkService{observations: make(chan observation, len(frames)+1)}
	mux := loomhttp.NewMuxer()
	server := chatserver.New(chat.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil)
	chatserver.Mount(mux, server)
	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http")+"/chat", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close handshake body: %v", err)
		}
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	}()
	for _, frame := range append(frames, "null") {
		if err := conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
			t.Fatalf("write %q: %v", frame, err)
		}
	}

	observations := collect(t, svc.observations, len(frames)+1)
	checkDecoding(t, observations[:len(frames)])
	if last := observations[len(frames)]; !errors.Is(last.err, io.EOF) || last.msg != nil {
		t.Errorf("null frame: got (%v, %v), want (nil, io.EOF)", last.msg, last.err)
	}
}

func TestGeneratedClientRecvDecoding(t *testing.T) {
	upgrader := websocket.Upgrader{}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer func() {
			if err := conn.Close(); err != nil {
				t.Errorf("close server: %v", err)
			}
		}()
		for _, frame := range frames {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
				t.Errorf("write %q: %v", frame, err)
				return
			}
		}
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
			t.Errorf("write close: %v", err)
			return
		}
		if _, _, err := conn.ReadMessage(); err == nil {
			t.Error("expected the client to close the connection")
		}
	}))
	defer httpServer.Close()

	client := chatclient.NewClient("ws", strings.TrimPrefix(httpServer.URL, "http://"), http.DefaultClient,
		loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	ctx := context.Background()
	raw, err := client.Talk()(ctx, nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(chat.TalkClientStream)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}

	observations := make([]observation, 0, len(frames)+1)
	for range len(frames) + 1 {
		msg, err := stream.RecvWithContext(ctx)
		observations = append(observations, observation{msg: msg, err: err})
	}
	checkDecoding(t, observations[:len(frames)])
	if last := observations[len(frames)]; !errors.Is(last.err, io.EOF) || last.msg != nil {
		t.Errorf("normal closure: got (%v, %v), want (nil, io.EOF)", last.msg, last.err)
	}
}

func collect(t *testing.T, observations chan observation, n int) []observation {
	t.Helper()
	collected := make([]observation, 0, n)
	for range n {
		select {
		case o := <-observations:
			collected = append(collected, o)
		case <-time.After(5 * time.Second):
			t.Fatalf("received %d of %d observations", len(collected), n)
		}
	}
	return collected
}

// checkDecoding asserts one observation per entry of frames. Decode errors
// must not end the stream: the valid frame after them still decodes.
func checkDecoding(t *testing.T, observations []observation) {
	t.Helper()
	names := []string{"empty", "whitespace only", "malformed", "truncated"}
	wantUnexpectedEOF := []bool{true, true, false, true}
	for i, name := range names {
		o := observations[i]
		var closeErr *websocket.CloseError
		switch {
		case o.err == nil:
			t.Errorf("%s: got nil error", name)
		case errors.As(o.err, &closeErr):
			t.Errorf("%s: decode failure reported as close error %v", name, o.err)
		case errors.Is(o.err, io.ErrUnexpectedEOF) != wantUnexpectedEOF[i]:
			t.Errorf("%s: errors.Is(err, io.ErrUnexpectedEOF) = %t, want %t (err %v)", name, !wantUnexpectedEOF[i], wantUnexpectedEOF[i], o.err)
		}
		if o.msg != nil {
			t.Errorf("%s: got message %+v, want nil", name, o.msg)
		}
	}

	valid := observations[len(names)]
	if valid.err != nil {
		t.Fatalf("valid: %v", valid.err)
	}
	if valid.msg == nil || valid.msg.Text != "hi" {
		t.Fatalf("valid: got %+v", valid.msg)
	}
	if valid.msg.Value == nil || valid.msg.Value.Kind() != chat.ValueKindCount || valid.msg.Value.Count != 3 {
		t.Errorf("valid union: got %+v", valid.msg.Value)
	}
	want := jsontext.Value(` + "`" + `{"k":[1,true]}` + "`" + `)
	if got := jsontext.Value(valid.msg.Extra); !got.IsValid() || string(got.Clone()) != string(want) {
		t.Errorf("valid any: got %s, want %s", got, want)
	}
}
`
