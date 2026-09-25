package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCWebSocketRecvAfterResponseGeneratedModule compiles a
// bidirectional JSON-RPC WebSocket client and checks, under the race
// detector, that Recv returns the response of every request sent with Send
// in send order: when the response arrived before Recv was called, when a
// Recv was canceled before the response arrived, and after a request timed
// out, whose late response must then be reported as orphaned. A Recv while a
// Send is still writing its request waits for that response, and a Recv
// after Close reports the closed stream and finds no held response. It also
// checks that the client Close succeeds however the shared connection was
// closed before, and reports a failure to close a live connection.
func TestJSONRPCWebSocketRecvAfterResponseGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketPendingOrderDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwspending", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pending_order_test.go"), []byte(jsonRPCWebSocketPendingOrderHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=3", "./...")
}

func jsonrpcWebSocketPendingOrderDSL() {
	dsl.API("wspending", func() {
		dsl.JSONRPC(func() {})
	})
	note := dsl.Type("Note", func() {
		dsl.Attribute("text", dsl.String)
		dsl.Required("text")
	})
	dsl.Service("echo", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("talk", func() {
			dsl.StreamingPayload(note)
			dsl.StreamingResult(note)
			dsl.JSONRPC(func() {})
		})
	})
}

const jsonRPCWebSocketPendingOrderHarness = `package jsonrpcwspending_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	echo "example.com/jsonrpcwspending/gen/echo"
	client "example.com/jsonrpcwspending/gen/jsonrpc/echo/client"
	server "example.com/jsonrpcwspending/gen/jsonrpc/echo/server"
	"github.com/CaliLuke/loom/jsonrpc"
	loomhttp "github.com/CaliLuke/loom/http"
)

// service echoes every note. It answers "slow" only once release is closed
// and "late" only once late is closed, after Talk returned; it reports the
// result of the late answer on lateSent. It signals responded after every
// other response.
type service struct {
	responded chan struct{}
	release   chan struct{}
	late      chan struct{}
	lateSent  chan error
}

func (s *service) HandleStream(ctx context.Context, stream echo.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (s *service) Talk(ctx context.Context, p *echo.Note, st echo.TalkServerStream) error {
	switch p.Text {
	case "late":
		go func() {
			<-s.late
			s.lateSent <- st.SendResponse(context.Background(), p)
		}()
		return nil
	case "slow":
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	err := st.SendResponse(ctx, p)
	s.responded <- struct{}{}
	return err
}

// gatedConn blocks its writes while gated until open is closed. It signals
// writing when a write starts blocking. Close closes the connection and also
// reports closeErr when it is set.
type gatedConn struct {
	net.Conn
	gated    chan struct{}
	open     chan struct{}
	writing  chan struct{}
	closeErr error
}

func newGatedConn() *gatedConn {
	return &gatedConn{gated: make(chan struct{}), open: make(chan struct{}), writing: make(chan struct{}, 1)}
}

func (c *gatedConn) Close() error {
	err := c.Conn.Close()
	if c.closeErr != nil {
		return errors.Join(c.closeErr, err)
	}
	return err
}

func (c *gatedConn) Write(b []byte) (int, error) {
	select {
	case <-c.gated:
		select {
		case c.writing <- struct{}{}:
		default:
		}
		<-c.open
	default:
	}
	return c.Conn.Write(b)
}

type fixture struct {
	svc      *service
	stream   *client.TalkClientStream
	orphaned chan *jsonrpc.RawResponse
	conn     *gatedConn
}

func (f *fixture) send(t *testing.T, ctx context.Context, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if err := f.stream.SendWithContext(ctx, &echo.Note{Text: text}); err != nil {
			t.Fatalf("send %s: %v", text, err)
		}
	}
}

func (f *fixture) awaitResponses(t *testing.T, ctx context.Context, n int) {
	t.Helper()
	for range n {
		select {
		case <-f.svc.responded:
		case <-ctx.Done():
			t.Fatal("server did not respond")
		}
	}
	// Give the client time to read the responses before the next Recv.
	time.Sleep(100 * time.Millisecond)
}

func (f *fixture) recv(t *testing.T, ctx context.Context, want string) {
	t.Helper()
	got, err := f.stream.RecvWithContext(ctx)
	if err != nil {
		t.Fatalf("recv %s: %v", want, err)
	}
	if got == nil || got.Text != want {
		t.Fatalf("recv: got %+v, want %s", got, want)
	}
}

func TestRecvOrder(t *testing.T) {
	cases := []struct {
		name    string
		timeout time.Duration
		run     func(t *testing.T, ctx context.Context, f *fixture)
	}{
		{"responses before recv", 0, func(t *testing.T, ctx context.Context, f *fixture) {
			const n = 8
			for i := range n {
				f.send(t, ctx, fmt.Sprint(i))
			}
			f.awaitResponses(t, ctx, n)
			for i := range n {
				f.recv(t, ctx, fmt.Sprint(i))
			}
		}},
		{"canceled recv", 0, func(t *testing.T, ctx context.Context, f *fixture) {
			f.send(t, ctx, "slow", "fast")
			canceled, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			if got, err := f.stream.RecvWithContext(canceled); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("canceled recv: got (%+v, %v), want context.DeadlineExceeded", got, err)
			}
			close(f.svc.release)
			f.recv(t, ctx, "slow")
			f.recv(t, ctx, "fast")
		}},
		{"recv while send writes", 0, func(t *testing.T, ctx context.Context, f *fixture) {
			close(f.conn.gated)
			sent := make(chan error, 1)
			go func() {
				sent <- f.stream.SendWithContext(ctx, &echo.Note{Text: "gated"})
			}()
			select {
			case <-f.conn.writing:
			case <-ctx.Done():
				t.Fatal("send did not write")
			}
			waiting, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			if got, err := f.stream.RecvWithContext(waiting); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("recv during send: got (%+v, %v), want context.DeadlineExceeded", got, err)
			}
			close(f.conn.open)
			if err := <-sent; err != nil {
				t.Fatalf("send: %v", err)
			}
			f.recv(t, ctx, "gated")
		}},
		{"recv after close", 0, func(t *testing.T, ctx context.Context, f *fixture) {
			f.send(t, ctx, "held")
			f.awaitResponses(t, ctx, 1)
			if err := f.stream.Close(); err != nil {
				t.Fatalf("close stream: %v", err)
			}
			if got, err := f.stream.RecvWithContext(ctx); err == nil || err.Error() != "stream closed" {
				t.Fatalf("recv after close: got (%+v, %v), want stream closed", got, err)
			}
		}},
		{"timed out request", 200 * time.Millisecond, func(t *testing.T, ctx context.Context, f *fixture) {
			f.send(t, ctx, "late")
			got, err := f.stream.RecvWithContext(ctx)
			if err == nil || !strings.Contains(err.Error(), "request timeout") {
				t.Fatalf("late recv: got (%+v, %v), want a request timeout", got, err)
			}
			close(f.svc.late)
			if err := <-f.svc.lateSent; err != nil {
				t.Fatalf("late response: %v", err)
			}
			select {
			case response := <-f.orphaned:
				if !strings.Contains(string(response.Result), ` + "`" + `"late"` + "`" + `) {
					t.Errorf("orphaned response %s, want the late one", response.Result)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("the late response was not reported as orphaned")
			}
			f.send(t, ctx, "after")
			f.recv(t, ctx, "after")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService()
			orphaned := make(chan *jsonrpc.RawResponse, 16)
			opts := []jsonrpc.StreamConfigOption{
				jsonrpc.WithErrorHandler(func(_ context.Context, kind jsonrpc.StreamErrorType, _ error, response *jsonrpc.RawResponse) {
					if kind == jsonrpc.StreamErrorOrphaned {
						orphaned <- response
					}
				}),
			}
			if tc.timeout > 0 {
				opts = append(opts, jsonrpc.WithRequestTimeout(tc.timeout))
			}
			conn := newGatedConn()
			c := newClient(t, svc, conn, nil, opts...)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			raw, err := c.Talk()(ctx, nil)
			if err != nil {
				t.Fatalf("open stream: %v", err)
			}
			f := &fixture{svc: svc, stream: raw.(*client.TalkClientStream), orphaned: orphaned, conn: conn}
			tc.run(t, ctx, f)
			if err := f.stream.Close(); err != nil {
				t.Errorf("close stream: %v", err)
			}
			if err := c.Close(); err != nil {
				t.Errorf("close client: %v", err)
			}
		})
	}
}

// TestClientClose checks that the client Close succeeds before any stream,
// when called twice, and after, before, or concurrently with a stream that
// closes the shared connection, and that it reports a failure to close a
// live connection.
func TestClientClose(t *testing.T) {
	errClose := errors.New("injected close failure")
	cases := []struct {
		name string
		// run gets an open stream unless noStream is set.
		noStream bool
		closeErr error
		run      func(t *testing.T, ctx context.Context, cancel context.CancelFunc, c *client.Client, stream *client.TalkClientStream)
	}{
		{"before any stream", true, nil, func(t *testing.T, _ context.Context, _ context.CancelFunc, c *client.Client, _ *client.TalkClientStream) {
			closeClientTwice(t, c)
		}},
		{"after stream close", false, nil, func(t *testing.T, ctx context.Context, _ context.CancelFunc, c *client.Client, stream *client.TalkClientStream) {
			roundTrip(t, ctx, stream)
			if err := stream.Close(); err != nil {
				t.Errorf("close stream: %v", err)
			}
			closeClientTwice(t, c)
		}},
		{"after stream context cancellation", false, nil, func(t *testing.T, ctx context.Context, cancel context.CancelFunc, c *client.Client, stream *client.TalkClientStream) {
			if err := stream.SendWithContext(ctx, &echo.Note{Text: "slow"}); err != nil {
				t.Fatalf("send: %v", err)
			}
			cancel()
			if got, err := stream.RecvWithContext(context.Background()); err == nil {
				t.Fatalf("recv after cancel: got %+v, want an error", got)
			}
			closeClientTwice(t, c)
			if err := stream.Close(); err != nil {
				t.Errorf("close stream: %v", err)
			}
		}},
		{"before stream close", false, nil, func(t *testing.T, ctx context.Context, _ context.CancelFunc, c *client.Client, stream *client.TalkClientStream) {
			roundTrip(t, ctx, stream)
			closeClientTwice(t, c)
			if err := stream.Close(); err != nil {
				t.Errorf("close stream: %v", err)
			}
		}},
		{"concurrent with stream close", false, nil, func(t *testing.T, ctx context.Context, _ context.CancelFunc, c *client.Client, stream *client.TalkClientStream) {
			roundTrip(t, ctx, stream)
			errs := make(chan error, 2)
			start := make(chan struct{})
			go func() {
				<-start
				errs <- c.Close()
			}()
			go func() {
				<-start
				errs <- stream.Close()
			}()
			close(start)
			for range 2 {
				if err := <-errs; err != nil {
					t.Errorf("concurrent close: %v", err)
				}
			}
			closeClientTwice(t, c)
		}},
		{"live connection close failure", false, errClose, func(t *testing.T, ctx context.Context, _ context.CancelFunc, c *client.Client, stream *client.TalkClientStream) {
			roundTrip(t, ctx, stream)
			if err := c.Close(); !errors.Is(err, errClose) {
				t.Errorf("close client: got %v, want %v", err, errClose)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newService()
			defer close(svc.release)
			conn := newGatedConn()
			conn.closeErr = tc.closeErr
			c := newClient(t, svc, conn, nil)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			var stream *client.TalkClientStream
			if !tc.noStream {
				raw, err := c.Talk()(ctx, nil)
				if err != nil {
					t.Fatalf("open stream: %v", err)
				}
				stream = raw.(*client.TalkClientStream)
			}
			tc.run(t, ctx, cancel, c, stream)
		})
	}
}

// TestConfigureReturningNil checks that a stream fails to open with an error,
// without keeping a connection, every time the connection configure function
// returns nil, that the dialed connection is closed, and that the client
// still closes cleanly.
func TestConfigureReturningNil(t *testing.T) {
	svc := newService()
	defer close(svc.release)
	conn := newGatedConn()
	var calls int
	cfn := func(*websocket.Conn, context.CancelFunc) *websocket.Conn {
		calls++
		return nil
	}
	c := newClient(t, svc, conn, cfn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := range 2 {
		if _, err := c.Talk()(ctx, nil); err == nil || !strings.Contains(err.Error(), "returned a nil connection") {
			t.Errorf("open stream (call %d): got %v, want a nil connection error", i+1, err)
		}
		if err := conn.Conn.Close(); !errors.Is(err, net.ErrClosed) {
			t.Errorf("dialed connection (call %d) was not closed: %v", i+1, err)
		}
	}
	if calls != 2 {
		t.Errorf("configure calls: got %d, want 2", calls)
	}
	closeClientTwice(t, c)
}

func newService() *service {
	return &service{
		responded: make(chan struct{}, 16),
		release:   make(chan struct{}),
		late:      make(chan struct{}),
		lateSent:  make(chan error, 1),
	}
}

// newClient serves svc and returns a client whose connection to it is conn,
// configured with cfn when it is not nil.
func newClient(t *testing.T, svc *service, conn *gatedConn, cfn loomhttp.ConnConfigureFunc, opts ...jsonrpc.StreamConfigOption) *client.Client {
	t.Helper()
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.HandleStream, echo.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	dialer := &websocket.Dialer{NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		c, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		conn.Conn = c
		return conn, nil
	}}
	return client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, dialer, cfn, opts...)
}

// roundTrip sends one note on stream and receives its echo.
func roundTrip(t *testing.T, ctx context.Context, stream *client.TalkClientStream) {
	t.Helper()
	if err := stream.SendWithContext(ctx, &echo.Note{Text: "ping"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got, err := stream.RecvWithContext(ctx); err != nil || got == nil || got.Text != "ping" {
		t.Fatalf("recv: got (%+v, %v), want ping", got, err)
	}
}

func closeClientTwice(t *testing.T, c *client.Client) {
	t.Helper()
	for i := range 2 {
		if err := c.Close(); err != nil {
			t.Errorf("close client (call %d): %v", i+1, err)
		}
	}
}
`
