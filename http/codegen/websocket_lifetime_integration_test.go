package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestWebSocketLifetimeGeneratedIntegration exercises generated cleanup with
// real sockets. LOOM_WEBSOCKET_TEST_OUTPUT retains each generated module for
// source-to-source byte comparison; ordinary runs use the test's temp directory.
func TestWebSocketLifetimeGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("lifetime", func() {
			Method("echo", func() {
				StreamingPayload(String)
				StreamingResult(String)
				HTTP(func() {
					GET("/echo")
				})
			})
			Method("upload", func() {
				StreamingPayload(String)
				Result(String)
				HTTP(func() {
					GET("/upload")
				})
			})
			Method("sub", func() {
				StreamingResult(String)
				HTTP(func() {
					GET("/sub")
				})
			})
		})
	})
	dir := t.TempDir()
	if output := os.Getenv("LOOM_WEBSOCKET_TEST_OUTPUT"); output != "" {
		require.NoError(t, os.MkdirAll(output, 0o700))
		var err error
		dir, err = os.MkdirTemp(output, "http-")
		require.NoError(t, err)
		t.Logf("retained generated module: %s", dir)
	}
	renderHTTPModule(t, dir, "example.com/wslifetime", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lifetime_test.go"), []byte(webSocketLifetimeHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

const webSocketLifetimeHarness = `package wslifetime_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	lifetime "example.com/wslifetime/gen/lifetime"
	client "example.com/wslifetime/gen/http/lifetime/client"
	server "example.com/wslifetime/gen/http/lifetime/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	echo func(context.Context, lifetime.EchoServerStream) error
	sub func(context.Context, lifetime.SubServerStream) error
	upload func(context.Context, lifetime.UploadServerStream) error
}

func (s *service) Echo(ctx context.Context, stream lifetime.EchoServerStream) error {
	return s.echo(ctx, stream)
}

func (s *service) Upload(ctx context.Context, stream lifetime.UploadServerStream) error {
	return s.upload(ctx, stream)
}

func (s *service) Sub(ctx context.Context, stream lifetime.SubServerStream) error {
	return s.sub(ctx, stream)
}

func newEnv(t *testing.T, svc *service) (*client.Client, <-chan *websocket.Conn, <-chan *websocket.Conn) {
	t.Helper()
	serverConns := make(chan *websocket.Conn, 1)
	clientConns := make(chan *websocket.Conn, 1)
	mux := loomhttp.NewMuxer()
	serverConfig, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: loomhttp.DefaultMaxRequestBodyBytes}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
		serverConns <- conn
		t.Cleanup(func() {
			if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				t.Errorf("server cleanup: %v", err)
			}
		})
		return conn
	})
	require.NoError(t, err)
	config := server.NewConnConfigurer(serverConfig)
	server.Mount(mux, server.New(lifetime.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, config))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	clientConfig, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: loomhttp.DefaultMaxRequestBodyBytes}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
		clientConns <- conn
		t.Cleanup(func() {
			if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
				t.Errorf("client cleanup: %v", err)
			}
		})
		return conn
	})
	require.NoError(t, err)
	cconfig := client.NewConnConfigurer(clientConfig)
	c := client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, cconfig)
	return c, serverConns, clientConns
}

func assertClosed(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	if err := conn.NetConn().SetDeadline(time.Now()); !errors.Is(err, net.ErrClosed) {
		t.Errorf("socket remains open: SetDeadline returned %v", err)
	}
}

func TestGeneratedCloseAfterPeerClose(t *testing.T) {
	c, _, clients := newEnv(t, &service{echo: func(_ context.Context, st lifetime.EchoServerStream) error {
		if err := st.Send("last"); err != nil {
			return err
		}
		return st.Close()
	}})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	raw, err := c.Echo()(ctx, nil)
	require.NoError(t, err)
	st := raw.(*client.EchoClientStream)
	value, err := st.RecvWithContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "last", value)
	_, err = st.RecvWithContext(ctx)
	require.ErrorIs(t, err, io.EOF)
	closeErr := st.Close()
	t.Logf("Close after peer close: %v", closeErr)
	assertClosed(t, <-clients)
}

func TestGeneratedServiceReturnClosesStream(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name := "success"
		if failed {
			name = "error"
		}
		t.Run(name, func(t *testing.T) {
			returned := make(chan struct{})
			c, servers, _ := newEnv(t, &service{sub: func(_ context.Context, st lifetime.SubServerStream) error {
				defer close(returned)
				if err := st.Send("last"); err != nil {
					return err
				}
				if failed {
					return errors.New("service stopped")
				}
				return nil
			}})
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			raw, err := c.Sub()(ctx, nil)
			require.NoError(t, err)
			st := raw.(*client.SubClientStream)
			_, err = st.RecvWithContext(ctx)
			require.NoError(t, err)
			<-returned
			recvCtx, recvCancel := context.WithTimeout(ctx, 300*time.Millisecond)
			defer recvCancel()
			_, err = st.RecvWithContext(recvCtx)
			if errors.Is(err, context.DeadlineExceeded) {
				t.Error("Recv blocked after service return")
			} else if !failed {
				require.ErrorIs(t, err, io.EOF)
			} else {
				require.Error(t, err)
			}
			assertClosed(t, <-servers)
		})
	}
}

func TestGeneratedDisconnectCancelsService(t *testing.T) {
	ended := make(chan error, 1)
	escape := make(chan struct{})
	defer close(escape)
	c, _, clients := newEnv(t, &service{sub: func(ctx context.Context, st lifetime.SubServerStream) error {
		if err := st.Send("ready"); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			ended <- ctx.Err()
		case <-escape:
			ended <- nil
		}
		return nil
	}})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	raw, err := c.Sub()(ctx, nil)
	require.NoError(t, err)
	st := raw.(*client.SubClientStream)
	_, err = st.RecvWithContext(ctx)
	require.NoError(t, err)
	require.NoError(t, (<-clients).Close())
	select {
	case err := <-ended:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(300*time.Millisecond):
		t.Error("disconnect did not cancel the service")
	}
}

func TestGeneratedHalfCloseFinalReply(t *testing.T) {
	c, _, _ := newEnv(t, &service{upload: func(ctx context.Context, stream lifetime.UploadServerStream) error {
		value, err := stream.Recv()
		if err != nil { return err }
		if _, err := stream.Recv(); !errors.Is(err, io.EOF) { return errors.New("missing input half-close") }
		if ctx.Err() != nil { return errors.New("half-close canceled service") }
		return stream.SendAndClose("final:"+value)
	}})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	raw, err := c.Upload()(ctx, nil)
	require.NoError(t, err)
	stream := raw.(*client.UploadClientStream)
	require.NoError(t, stream.Send("payload"))
	result, err := stream.CloseAndRecvWithContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "final:payload", result)
}

func TestGeneratedPanicClosesWithSafeReason(t *testing.T) {
	c, _, _ := newEnv(t, &service{sub: func(_ context.Context, stream lifetime.SubServerStream) error {
		if err := stream.Send("ready"); err != nil { return err }
		panic("private panic text")
	}})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	raw, err := c.Sub()(ctx, nil)
	require.NoError(t, err)
	stream := raw.(*client.SubClientStream)
	value, err := stream.RecvWithContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "ready", value)
	_, err = stream.RecvWithContext(ctx)
	var closeErr *websocket.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, websocket.CloseInternalServerErr, closeErr.Code)
	require.Equal(t, "handler failed", closeErr.Text)
}

func TestGeneratedErrorBeforeUpgrade(t *testing.T) {
	c, servers, _ := newEnv(t, &service{sub: func(context.Context, lifetime.SubServerStream) error {
		return errors.New("private pre-upgrade error")
	}})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	_, err := c.Sub()(ctx, nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private pre-upgrade error")
	select {
	case <-servers: t.Fatal("error cleanup triggered upgrade")
	default:
	}
}
`
