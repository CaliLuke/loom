package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebSocketMessageLimitGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, webSocketDecodeDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/websocketdecode", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "websocket_decode_test.go"), []byte(webSocketDecodeHarness), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "websocket_limit_test.go"), []byte(webSocketLimitHarness), 0600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

const webSocketLimitHarness = `package websocketdecode

import (
	"context"
	"errors"
	"fmt"
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

type limitService struct {
	result chan observation
}

func (s *limitService) Talk(ctx context.Context, stream chat.TalkServerStream) error {
	msg, err := stream.RecvWithContext(ctx)
	s.result <- observation{msg: msg, err: err}
	return err
}

func TestGeneratedMessageLimit(t *testing.T) {
	for _, receiver := range []string{"server", "client"} {
		for _, size := range []int{128, 129} {
			t.Run(fmt.Sprintf("%s/%d", receiver, size), func(t *testing.T) {
				config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: 128}, nil)
				if err != nil {
					t.Fatal(err)
				}
				result := make(chan observation, 1)
				frame := []byte(fmt.Sprintf("{\"text\":%q}", strings.Repeat("x", size-11)))
				if len(frame) != size {
					t.Fatalf("frame length %d", len(frame))
				}
				handlerDone := make(chan struct{})
				var handler http.Handler
				if receiver == "server" {
					mux := loomhttp.NewMuxer()
					srv := chatserver.New(chat.NewEndpoints(&limitService{result: result}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{EnableCompression: true}, chatserver.NewConnConfigurer(config))
					chatserver.Mount(mux, srv)
					handler = mux
				} else {
					handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						defer close(handlerDone)
						upgrader := websocket.Upgrader{EnableCompression: true}
						conn, err := upgrader.Upgrade(w, r, nil)
						if err != nil {
							t.Error(err)
							return
						}
						defer func() {
							if err := conn.Close(); err != nil {
								t.Error(err)
							}
						}()
						if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
							t.Error(err)
						}
						_, _, err = conn.ReadMessage()
						if err != nil && !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
							t.Error(err)
						}
					})
				}
				hs := httptest.NewServer(handler)
				defer hs.Close()
				dialer := &websocket.Dialer{EnableCompression: true}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if receiver == "server" {
					conn, resp, err := dialer.DialContext(ctx, "ws"+strings.TrimPrefix(hs.URL, "http")+"/chat", nil)
					if err != nil {
						t.Fatal(err)
					}
					if err := resp.Body.Close(); err != nil {
						t.Fatal(err)
					}
					defer func() {
						if err := conn.Close(); err != nil {
							t.Error(err)
						}
					}()
					if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
						t.Fatal(err)
					}
					if size > 128 {
						if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
							t.Fatal(err)
						}
						_, _, err = conn.ReadMessage()
						if !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
							t.Errorf("peer close: %v", err)
						}
					}
				} else {
					c := chatclient.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, dialer, chatclient.NewConnConfigurer(config))
					value, err := c.Talk()(ctx, nil)
					if err != nil {
						t.Fatal(err)
					}
					stream := value.(*chatclient.TalkClientStream)
					msg, err := stream.RecvWithContext(ctx)
					result <- observation{msg: msg, err: err}
					if err := stream.Close(); err != nil && size <= 128 {
						t.Error(err)
					}
					select {
					case <-handlerDone:
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					}
				}
				select {
				case got := <-result:
					if size > 128 {
						wantErr := websocket.ErrReadLimit
						if receiver == "server" {
							// The shared owner cancels the service lifetime on overflow;
							// an operation using that context observes cancellation.
							wantErr = context.Canceled
						}
						if !errors.Is(got.err, wantErr) || got.msg != nil {
							t.Errorf("oversize decoded: %#v, %v", got.msg, got.err)
						}
					} else if got.err != nil || got.msg == nil || got.msg.Text != strings.Repeat("x", size-11) {
						t.Errorf("exact limit: %#v, %v", got.msg, got.err)
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			})
		}
	}
}
func TestGeneratedNilHookRetainsUpgradeState(t *testing.T) {
	calls := make(chan string, 2)
	result := make(chan observation, 1)
	config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: 128}, func(*websocket.Conn, context.CancelFunc) *websocket.Conn {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := loomhttp.NewMuxer()
	encoder := func(ctx context.Context, w http.ResponseWriter) loomhttp.Encoder {
		calls <- "http encoder"
		return loomhttp.ResponseEncoder(ctx, w)
	}
	errhandler := func(_ context.Context, _ http.ResponseWriter, err error) {
		if err == nil {
			t.Error("missing hook error")
		}
		calls <- "transport error handler"
	}
	srv := chatserver.New(chat.NewEndpoints(&limitService{result: result}), mux, loomhttp.RequestDecoder, encoder, errhandler, nil, &websocket.Upgrader{}, chatserver.NewConnConfigurer(config))
	chatserver.Mount(mux, srv)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(hs.URL, "http")+"/chat", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()
	select {
	case got := <-calls:
		if got != "transport error handler" {
			t.Errorf("post-upgrade hook failure invoked %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("hook failure was not reported")
	}
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	_, _, err = conn.ReadMessage()
	if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
		t.Errorf("input socket remains live: %v", err)
	}
}
`
