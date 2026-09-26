package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJSONRPCWebSocketStreamsShareConnectionGeneratedModule compiles a
// bidirectional JSON-RPC WebSocket client and checks, under the race
// detector, that several streams opened on one client share its connection
// without stealing each other's responses: each stream receives exactly the
// responses to its own requests, in send order. Closing a stream or canceling
// its context ends only that stream, the connection closes once the last
// stream and the client are done with it, and a connection failure ends every
// open stream instead of leaving one waiting.
func TestJSONRPCWebSocketStreamsShareConnectionGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcWebSocketPendingOrderDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcwsdemux", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "demux_test.go"), []byte(jsonRPCWebSocketDemuxHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=3", "./...")
}

const jsonRPCWebSocketDemuxHarness = `package jsonrpcwsdemux_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	echo "example.com/jsonrpcwsdemux/gen/echo"
	client "example.com/jsonrpcwsdemux/gen/jsonrpc/echo/client"
	server "example.com/jsonrpcwsdemux/gen/jsonrpc/echo/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
)

// service echoes every note except those whose text starts with "drop",
// which it never answers. The generated server handles the requests of a
// connection one at a time, so a note that blocked would stall every stream.
type service struct{}

func (s *service) HandleStream(ctx context.Context, stream echo.Stream) error {
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (s *service) Talk(ctx context.Context, p *echo.Note, st echo.TalkServerStream) error {
	if strings.HasPrefix(p.Text, "drop") {
		return nil
	}
	return st.SendResponse(ctx, p)
}

// countingConn counts the dialed connections and records the last one.
type countingConn struct {
	mu    sync.Mutex
	dials int
	last  net.Conn
	started chan struct{}
	resume chan struct{}
}

func (c *countingConn) lastConn() net.Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last
}

func (c *countingConn) dialCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dials
}

func newClient(t *testing.T, svc *service, dials *countingConn, opts ...jsonrpc.StreamConfigOption) *client.Client {
	t.Helper()
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(svc.HandleStream, echo.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	dialer := &websocket.Dialer{NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		if dials.started != nil {
			select {
			case dials.started <- struct{}{}:
			default:
			}
			select {
			case <-dials.resume:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		c, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		dials.mu.Lock()
		dials.dials++
		dials.last = c
		dials.mu.Unlock()
		return c, nil
	}}
	return client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, dialer, nil, opts...)
}

func open(t *testing.T, ctx context.Context, c *client.Client) *client.TalkClientStream {
	t.Helper()
	raw, err := c.Talk()(ctx, nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	return raw.(*client.TalkClientStream)
}

func roundTrip(t *testing.T, ctx context.Context, stream *client.TalkClientStream, text string) {
	t.Helper()
	if err := stream.SendWithContext(ctx, &echo.Note{Text: text}); err != nil {
		t.Fatalf("send %s: %v", text, err)
	}
	got, err := stream.RecvWithContext(ctx)
	if err != nil {
		t.Fatalf("recv %s: %v", text, err)
	}
	if got == nil || got.Text != text {
		t.Fatalf("recv: got %+v, want %s", got, text)
	}
}

// TestStreamsReceiveOwnResponses runs several streams on one client
// concurrently. Each sends a burst of notes and then receives them; every
// stream must get back exactly its own notes, in send order.
func TestStreamsReceiveOwnResponses(t *testing.T) {
	const streams, notes = 4, 25
	var orphaned atomic.Int32
	svc := &service{}
	dials := &countingConn{}
	c := newClient(t, svc, dials, jsonrpc.WithErrorHandler(func(_ context.Context, kind jsonrpc.StreamErrorType, _ error, _ *jsonrpc.RawResponse) {
		if kind == jsonrpc.StreamErrorOrphaned {
			orphaned.Add(1)
		}
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	opened := make([]*client.TalkClientStream, streams)
	for i := range opened {
		opened[i] = open(t, ctx, c)
	}
	var wg sync.WaitGroup
	errs := make(chan error, streams)
	for i, stream := range opened {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range notes {
				if err := stream.SendWithContext(ctx, &echo.Note{Text: fmt.Sprintf("s%d-%d", i, n)}); err != nil {
					errs <- fmt.Errorf("stream %d send %d: %w", i, n, err)
					return
				}
			}
			for n := range notes {
				want := fmt.Sprintf("s%d-%d", i, n)
				got, err := stream.RecvWithContext(ctx)
				if err != nil {
					errs <- fmt.Errorf("stream %d recv %s: %w", i, want, err)
					return
				}
				if got == nil || got.Text != want {
					errs <- fmt.Errorf("stream %d recv: got %+v, want %s", i, got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if n := orphaned.Load(); n != 0 {
		t.Errorf("%d responses reported as orphaned", n)
	}
	if n := dials.dialCount(); n != 1 {
		t.Errorf("streams dialed %d connections, want 1", n)
	}
	for i, stream := range opened {
		if err := stream.Close(); err != nil {
			t.Errorf("close stream %d: %v", i, err)
		}
	}
	if err := c.Close(); err != nil {
		t.Errorf("close client: %v", err)
	}
}

// TestStreamEndsAlone checks that closing one stream, or canceling the
// context it was opened with, ends that stream only: its held request fails,
// and the other stream on the connection keeps working. The connection closes
// once no stream uses it, and the next stream dials a new one.
func TestStreamEndsAlone(t *testing.T) {
	cases := []struct {
		name string
		end  func(t *testing.T, cancel context.CancelFunc, stream *client.TalkClientStream)
	}{
		{"close", func(t *testing.T, _ context.CancelFunc, stream *client.TalkClientStream) {
			if err := stream.Close(); err != nil {
				t.Errorf("close stream: %v", err)
			}
		}},
		{"context canceled", func(_ *testing.T, cancel context.CancelFunc, _ *client.TalkClientStream) {
			cancel()
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &service{}
			dials := &countingConn{}
			c := newClient(t, svc, dials)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			endCtx, endCancel := context.WithCancel(ctx)
			defer endCancel()
			ending := open(t, endCtx, c)
			staying := open(t, ctx, c)
			if err := ending.SendWithContext(ctx, &echo.Note{Text: "drop"}); err != nil {
				t.Fatalf("send: %v", err)
			}
			received := make(chan error, 1)
			go func() {
				_, err := ending.RecvWithContext(ctx)
				received <- err
			}()
			roundTrip(t, ctx, staying, "before")
			tc.end(t, endCancel, ending)
			select {
			case err := <-received:
				if err == nil {
					t.Error("the held request of the ended stream succeeded")
				}
			case <-ctx.Done():
				t.Fatal("the held request of the ended stream is still waiting")
			}
			roundTrip(t, ctx, staying, "after")
			if err := staying.Close(); err != nil {
				t.Errorf("close staying stream: %v", err)
			}
			if err := ending.Close(); err != nil {
				t.Errorf("close ended stream: %v", err)
			}
			if _, err := dials.lastConn().Write([]byte{0}); !errors.Is(err, net.ErrClosed) {
				t.Errorf("connection still open after its streams closed: %v", err)
			}
			next := open(t, ctx, c)
			roundTrip(t, ctx, next, "next")
			if n := dials.dialCount(); n != 2 {
				t.Errorf("dialed %d connections, want 2", n)
			}
			if err := next.Close(); err != nil {
				t.Errorf("close next stream: %v", err)
			}
			if err := c.Close(); err != nil {
				t.Errorf("close client: %v", err)
			}
		})
	}
}

// TestConnectionFailureEndsEveryStream checks that when the connection fails
// every stream waiting on it returns an error instead of waiting forever, and
// that later sends and receives fail too.
func TestConnectionFailureEndsEveryStream(t *testing.T) {
	svc := &service{}
	dials := &countingConn{}
	var connectionErrors atomic.Int32
	c := newClient(t, svc, dials, jsonrpc.WithErrorHandler(func(_ context.Context, kind jsonrpc.StreamErrorType, _ error, _ *jsonrpc.RawResponse) {
		if kind == jsonrpc.StreamErrorConnection {
			connectionErrors.Add(1)
		}
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opened := []*client.TalkClientStream{open(t, ctx, c), open(t, ctx, c)}
	received := make(chan error, len(opened))
	for i, stream := range opened {
		if err := stream.SendWithContext(ctx, &echo.Note{Text: fmt.Sprintf("drop-%d", i)}); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
		go func() {
			_, err := stream.RecvWithContext(ctx)
			received <- err
		}()
	}
	// Close the socket under the client, as a network failure would.
	if err := dials.lastConn().Close(); err != nil {
		t.Fatalf("close socket: %v", err)
	}
	for range opened {
		select {
		case err := <-received:
			if err == nil {
				t.Error("a request succeeded after the connection failed")
			}
		case <-ctx.Done():
			t.Fatal("a stream is still waiting after the connection failed")
		}
	}
	for i, stream := range opened {
		if err := stream.SendWithContext(ctx, &echo.Note{Text: "late"}); err == nil {
			t.Errorf("stream %d: send after the connection failed succeeded", i)
		}
		if _, err := stream.RecvWithContext(ctx); err == nil {
			t.Errorf("stream %d: recv after the connection failed succeeded", i)
		}
	}
	if n := connectionErrors.Load(); n == 0 {
		t.Error("the connection failure was not reported")
	}
	for i, stream := range opened {
		if err := stream.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close stream %d: %v", i, err)
		}
	}
	if err := c.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		t.Errorf("close client: %v", err)
	}
}

func assertClientClosed(t *testing.T, ctx context.Context, c *client.Client, dials *countingConn) {
	t.Helper()
	before := dials.dialCount()
	for range 3 {
		raw, err := c.Talk()(ctx, nil)
		if err == nil || raw != nil {
			t.Errorf("call after Close: stream=%v error=%v", raw, err)
		}
		if raw != nil {
			if err := raw.(*client.TalkClientStream).Close(); err != nil {
				t.Errorf("cleanup unexpected stream: %v", err)
			}
		}
		if err := c.Close(); err != nil {
			t.Errorf("repeated Close: %v", err)
		}
	}
	if got := dials.dialCount(); got != before {
		t.Errorf("calls after Close dialed %d new connections", got-before)
	}
}

func TestClientClosePreventsNewStreams(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(fmt.Sprintf("active=%t", active), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			dials := &countingConn{}
			c := newClient(t, &service{}, dials)
			if active {
				stream := open(t, ctx, c)
				roundTrip(t, ctx, stream, "before close")
			}
			if err := c.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			assertClientClosed(t, ctx, c, dials)
		})
	}
}

func TestClientCloseDuringDial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dials := &countingConn{started: make(chan struct{}, 1), resume: make(chan struct{})}
	c := newClient(t, &service{}, dials)
	opened := make(chan error, 1)
	go func() {
		_, err := c.Talk()(ctx, nil)
		opened <- err
	}()
	select {
	case <-dials.started:
	case <-ctx.Done():
		t.Fatal("dial did not start")
	}
	closed := make(chan error, 1)
	go func() {
		closed <- c.Close()
	}()
	for !c.IsClosed() {
		select {
		case <-ctx.Done():
			t.Fatal("Close did not mark the client")
		case <-time.After(time.Millisecond):
		}
	}
	close(dials.resume)
	select {
	case err := <-closed:
		if err != nil {
			t.Errorf("close during dial: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Close did not return")
	}
	select {
	case <-opened: // A racing call can either fail or return a closed stream.
	case <-ctx.Done():
		t.Fatal("opening stream did not return")
	}
	if conn := dials.lastConn(); conn == nil {
		t.Error("no connection was dialed")
	} else if _, err := conn.Write([]byte{0}); !errors.Is(err, net.ErrClosed) {
		t.Errorf("connection survived Close: %v", err)
	}
	assertClientClosed(t, ctx, c, dials)
}
`
