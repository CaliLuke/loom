package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONRPCWebSocketMessageLimitGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketPendingOrderDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsdemux", root)
	module, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	module = []byte(strings.ReplaceAll(string(module), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013014218-1eba09fe59d7"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), module, 0600))
	harness := strings.Replace(jsonRPCWebSocketDemuxHarness,
		`"github.com/gorilla/websocket"`,
		"\"github.com/gorilla/websocket\"\n\t\"github.com/stretchr/testify/require\"", 1) + jsonRPCWebSocketLimitHarness
	require.NoError(t, os.WriteFile(filepath.Join(dir, "message_limit_test.go"), []byte(harness), 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-run", "TestGeneratedMessageLimit", "-count=1", ".")
}

const jsonRPCWebSocketLimitHarness = `
// limitObservedConn records completion of native Close. A gate can hold Close
// after the protocol close frame has already reached the remote reader.
type limitObservedConn struct {
	net.Conn
	entered   chan struct{}
	closed    chan struct{}
	gate      <-chan struct{}
	enterOnce sync.Once
	closeOnce sync.Once
}

type limitObservedListener struct {
	net.Listener
	gate <-chan struct{}
}

type limitService struct {
	service
	calls atomic.Int32
	large bool
}

func (c *limitObservedConn) Close() error {
	c.enterOnce.Do(func() {
		close(c.entered)
	})
	if c.gate != nil {
		<-c.gate
	}
	err := c.Conn.Close()
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return err
}

func (l *limitObservedListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &limitObservedConn{Conn: conn, entered: make(chan struct{}), closed: make(chan struct{}), gate: l.gate}, nil
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
				serverConfig, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: serverLimit}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
					serverConn <- conn
					return conn
				})
				if err != nil {
					t.Fatal(err)
				}
				clientConfig, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: clientLimit}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
					clientConn <- conn
					return conn
				})
				if err != nil {
					t.Fatal(err)
				}
				svc := &limitService{large: large && receiver == "client"}
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(svc.HandleStream, echo.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{EnableCompression: true}, serverConfig))
				closeGate := make(chan struct{})
				var releaseOnce sync.Once
				release := func() {
					releaseOnce.Do(func() {
						close(closeGate)
					})
				}
				defer release()
				var serverGate <-chan struct{}
				if large && receiver == "server" {
					serverGate = closeGate
				}
				hs := httptest.NewUnstartedServer(mux)
				hs.Listener = &limitObservedListener{Listener: hs.Listener, gate: serverGate}
				hs.Start()
				defer hs.Close()
				c := client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, &websocket.Dialer{EnableCompression: true, NetDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
					if err != nil {
						return nil, err
					}
					return &limitObservedConn{Conn: conn, entered: make(chan struct{}), closed: make(chan struct{})}, nil
				}}, clientConfig)
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
					observed, ok := conn.NetConn().(*limitObservedConn)
					require.True(t, ok, "unexpected native connection %T", conn.NetConn())
					if serverGate != nil {
						// Peer receipt precedes native closure. Hold that exact
						// ordering deterministically, then observe completion.
						select {
						case <-observed.entered:
						case <-ctx.Done():
							require.NoError(t, ctx.Err(), "native close not started")
						}
						select {
						case <-observed.closed:
							require.FailNow(t, "native close completed before gate release")
						default:
						}
						release()
					}
					select {
					case <-observed.closed:
					case <-ctx.Done():
						require.NoError(t, ctx.Err(), "native close did not finish")
					}
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
