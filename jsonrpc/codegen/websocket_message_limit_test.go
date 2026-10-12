package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONRPCWebSocketMessageLimitGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketPendingOrderDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsdemux", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "message_limit_test.go"), []byte(jsonRPCWebSocketDemuxHarness+jsonRPCWebSocketLimitHarness), 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-run", "TestGeneratedMessageLimit", "-count=1", ".")
}

const jsonRPCWebSocketLimitHarness = `
type limitService struct {
	service
	calls atomic.Int32
	large bool
}

func (s *limitService) Talk(ctx context.Context, p *echo.Note, st echo.TalkServerStream) error {
	s.calls.Add(1)
	if s.large {
		p = &echo.Note{Text: strings.Repeat("x", 512)}
	}
	return st.SendResponse(ctx, p)
}
func TestGeneratedMessageLimit(t *testing.T) {
	for _, receiver := range []string{"server", "client"} {
		for _, large := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/large=%v", receiver, large), func(t *testing.T) {
				serverLimit, clientLimit := int64(1024), int64(1024)
				if receiver == "server" {
					serverLimit = 128
				} else {
					clientLimit = 128
				}
				serverConn := make(chan *websocket.Conn, 1)
				clientConn := make(chan *websocket.Conn, 1)
				serverConfig, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: serverLimit}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn { serverConn <- conn; return conn })
				if err != nil {
					t.Fatal(err)
				}
				clientConfig, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: clientLimit}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn { clientConn <- conn; return conn })
				if err != nil {
					t.Fatal(err)
				}
				svc := &limitService{large: large && receiver == "client"}
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(svc.HandleStream, echo.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{EnableCompression: true}, serverConfig))
				hs := httptest.NewServer(mux)
				defer hs.Close()
				c := client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, &websocket.Dialer{EnableCompression: true}, clientConfig)
				defer func() {
					if err := c.Close(); err != nil {
						t.Error(err)
					}
				}()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				value, err := c.Talk()(ctx, nil)
				if err != nil {
					t.Fatal(err)
				}
				stream := value.(*client.TalkClientStream)
				text := "ok"
				if large && receiver == "server" {
					text = strings.Repeat("x", 512)
				}
				if err := stream.SendWithContext(ctx, &echo.Note{Text: text}); err != nil {
					t.Fatal(err)
				}
				got, err := stream.RecvWithContext(ctx)
				if large {
					if got != nil || err == nil {
						t.Errorf("oversize result %#v, %v", got, err)
					}
					if receiver == "client" && !errors.Is(err, websocket.ErrReadLimit) {
						t.Errorf("client error: %v", err)
					}
					if receiver == "server" {
                        var closeErr *websocket.CloseError
                        if !errors.As(err, &closeErr) || closeErr.Code != websocket.CloseMessageTooBig {
                            t.Errorf("server peer close: %v", err)
                        }
                    }
					selected := clientConn
					if receiver == "server" {
						selected = serverConn
					}
					conn := <-selected
					if err := conn.NetConn().SetDeadline(time.Now()); !errors.Is(err, net.ErrClosed) {
						t.Errorf("limited socket not released: %v", err)
					}
				} else if err != nil || got == nil || got.Text != "ok" {
					t.Errorf("valid result %#v, %v", got, err)
				}
				wantCalls := int32(1)
				if large && receiver == "server" {
					wantCalls = 0
				}
				if got := svc.calls.Load(); got != wantCalls {
					t.Errorf("service calls %d, want %d", got, wantCalls)
				}
			})
		}
	}
}
`
