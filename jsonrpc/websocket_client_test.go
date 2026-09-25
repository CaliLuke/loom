package jsonrpc

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

type (
	// scriptedServer is a WebSocket JSON-RPC server whose answers the test
	// writes itself. It hands every request it reads to requests.
	scriptedServer struct {
		requests chan *RawRequest
		conns    chan *websocket.Conn
		url      string
	}

	// recordedErrors collects the errors passed to a StreamErrorHandler.
	recordedErrors struct {
		mu    sync.Mutex
		kinds []StreamErrorType
	}
)

func TestWebSocketClientStreamsReceiveOwnResponses(t *testing.T) {
	srv := newScriptedServer(t)
	errs := &recordedErrors{}
	conn, serverConn := dialScripted(t, srv, errs.handle)
	const streams, requests = 3, 10
	opened := make([]*WebSocketClientStream, streams)
	for i := range opened {
		opened[i] = openStream(t, t.Context(), conn, errs.handle)
	}
	var wg sync.WaitGroup
	for i, s := range opened {
		wg.Go(func() {
			for n := range requests {
				if err := s.Send(t.Context(), fmt.Sprintf("s%d-%d", i, n)); err != nil {
					t.Errorf("stream %d send %d: %v", i, n, err)
					return
				}
			}
		})
	}
	// Answer every request in reverse arrival order, echoing its params.
	received := make([]*RawRequest, 0, streams*requests)
	for range streams * requests {
		received = append(received, srv.next(t))
	}
	wg.Wait()
	ids := make(map[string]bool)
	for i := len(received) - 1; i >= 0; i-- {
		req := received[i]
		id := IDToString(req.ID)
		require.False(t, ids[id], "request id %s reused on the connection", id)
		ids[id] = true
		writeRaw(t, serverConn, fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":%s}`, id, req.Params))
	}
	for i, s := range opened {
		wg.Go(func() {
			for n := range requests {
				want := fmt.Sprintf("%q", fmt.Sprintf("s%d-%d", i, n))
				response, err := s.Recv(t.Context())
				if err != nil {
					t.Errorf("stream %d recv %d: %v", i, n, err)
					return
				}
				if string(response.Result) != want {
					t.Errorf("stream %d recv %d: got %s, want %s", i, n, response.Result, want)
				}
			}
		})
	}
	wg.Wait()
	require.Empty(t, errs.kindsSeen())
}

func TestWebSocketClientConnRoutesByID(t *testing.T) {
	cases := []struct {
		name      string
		reply     func(id string) string
		wantKinds []StreamErrorType
		wantOK    bool
	}{
		{"numeric id", func(id string) string {
			return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":"ok"}`, id)
		}, nil, true},
		{"unknown id then answer", func(id string) string {
			return `{"jsonrpc":"2.0","id":"999","result":"stray"}` + "\n" + fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"ok"}`, id)
		}, []StreamErrorType{StreamErrorOrphaned}, true},
		{"notification then answer", func(id string) string {
			return `{"jsonrpc":"2.0","method":"tick","params":{}}` + "\n" + fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"ok"}`, id)
		}, []StreamErrorType{StreamErrorNotification}, true},
		{"error response", func(id string) string {
			return fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"error":{"code":-32000,"message":"boom"}}`, id)
		}, []StreamErrorType{StreamErrorProtocol}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newScriptedServer(t)
			errs := &recordedErrors{}
			conn, serverConn := dialScripted(t, srv, errs.handle)
			s := openStream(t, t.Context(), conn, errs.handle)
			require.NoError(t, s.Send(t.Context(), "x"))
			req := srv.next(t)
			for _, msg := range strings.Split(tc.reply(IDToString(req.ID)), "\n") {
				writeRaw(t, serverConn, msg)
			}
			response, err := s.Recv(t.Context())
			if tc.wantOK {
				require.NoError(t, err)
				require.JSONEq(t, `"ok"`, string(response.Result))
			} else {
				var rpcErr *RawErrorResponse
				require.ErrorAs(t, err, &rpcErr)
				require.Equal(t, "boom", rpcErr.Message)
			}
			require.Equal(t, tc.wantKinds, errs.kindsSeen())
			require.NoError(t, s.Close())
		})
	}
}

func TestWebSocketClientConnFailureFailsEveryWaiter(t *testing.T) {
	srv := newScriptedServer(t)
	errs := &recordedErrors{}
	conn, serverConn := dialScripted(t, srv, errs.handle)
	a := openStream(t, t.Context(), conn, errs.handle)
	b := openStream(t, t.Context(), conn, errs.handle)
	require.NoError(t, a.Send(t.Context(), "a"))
	callErr := make(chan error, 1)
	go func() {
		_, err := b.Call(t.Context(), "b")
		callErr <- err
	}()
	srv.next(t)
	srv.next(t)
	require.NoError(t, serverConn.Close())
	_, err := a.Recv(t.Context())
	require.ErrorContains(t, err, "failed to read response")
	require.ErrorContains(t, <-callErr, "failed to read response")
	<-conn.Done()
	// No request registers after the waiters were failed.
	require.ErrorContains(t, a.Send(t.Context(), "late"), "failed to read response")
	require.ErrorContains(t, b.Send(t.Context(), "late"), "failed to read response")
	_, err = a.Recv(t.Context())
	require.Error(t, err)
	require.False(t, conn.Acquire(), "an ended connection must not take new streams")
	require.Equal(t, []StreamErrorType{StreamErrorConnection}, errs.kindsSeen())
	require.NoError(t, a.Close())
	require.NoError(t, b.Close())
}

func TestWebSocketClientStreamEndsAlone(t *testing.T) {
	cases := []struct {
		name string
		end  func(s *WebSocketClientStream, cancel context.CancelFunc) error
		want error
	}{
		{"close", func(s *WebSocketClientStream, _ context.CancelFunc) error {
			return s.Close()
		}, ErrWebSocketClientStreamClosed},
		{"context canceled", func(_ *WebSocketClientStream, cancel context.CancelFunc) error {
			cancel()
			return nil
		}, context.Canceled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newScriptedServer(t)
			errs := &recordedErrors{}
			conn, serverConn := dialScripted(t, srv, errs.handle)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			ending := openStream(t, ctx, conn, errs.handle)
			staying := openStream(t, t.Context(), conn, errs.handle)
			require.NoError(t, ending.Send(t.Context(), "held"))
			held := srv.next(t)
			waiting := make(chan error, 1)
			go func() {
				_, err := ending.Recv(t.Context())
				waiting <- err
			}()
			require.NoError(t, tc.end(ending, cancel))
			require.ErrorIs(t, <-waiting, tc.want)
			require.ErrorIs(t, ending.Send(t.Context(), "after"), ErrWebSocketClientStreamClosed)
			// The late answer of the ended stream is an orphan; the other
			// stream keeps working on the same connection.
			writeRaw(t, serverConn, fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"late"}`, IDToString(held.ID)))
			require.NoError(t, staying.Send(t.Context(), "live"))
			live := srv.next(t)
			writeRaw(t, serverConn, fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"live"}`, IDToString(live.ID)))
			response, err := staying.Recv(t.Context())
			require.NoError(t, err)
			require.JSONEq(t, `"live"`, string(response.Result))
			require.Equal(t, []StreamErrorType{StreamErrorOrphaned}, errs.kindsSeen())
			require.NoError(t, conn.Err())
			// Releasing the last stream closes the connection.
			require.NoError(t, staying.Close())
			require.ErrorIs(t, conn.Err(), ErrWebSocketClientConnClosed)
			<-conn.Done()
			require.NoError(t, ending.Close())
			require.NoError(t, ending.Close())
		})
	}
}

func TestWebSocketClientStreamRecvCancelAndTimeout(t *testing.T) {
	srv := newScriptedServer(t)
	errs := &recordedErrors{}
	conn, serverConn := dialScripted(t, srv, errs.handle)
	config := NewStreamConfig(WithRequestTimeout(300*time.Millisecond), WithErrorHandler(errs.handle))
	require.True(t, conn.Acquire())
	s := NewWebSocketClientStream(t.Context(), conn, "echo", config)
	require.NoError(t, s.Send(t.Context(), "first"))
	first := srv.next(t)
	canceled, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	_, err := s.Recv(canceled)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// The canceled Recv left the request first in line.
	writeRaw(t, serverConn, fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"first"}`, IDToString(first.ID)))
	response, err := s.Recv(t.Context())
	require.NoError(t, err)
	require.JSONEq(t, `"first"`, string(response.Result))
	// A timed out request is forgotten: its late answer is an orphan.
	require.NoError(t, s.Send(t.Context(), "slow"))
	slow := srv.next(t)
	_, err = s.Recv(t.Context())
	require.ErrorContains(t, err, "request timeout")
	writeRaw(t, serverConn, fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"slow"}`, IDToString(slow.ID)))
	require.Eventually(t, func() bool {
		return len(errs.kindsSeen()) == 2
	}, 5*time.Second, 10*time.Millisecond)
	require.Equal(t, []StreamErrorType{StreamErrorTimeout, StreamErrorOrphaned}, errs.kindsSeen())
	_, err = s.Recv(t.Context())
	require.ErrorContains(t, err, "no pending requests")
	require.NoError(t, s.Close())
	_, err = s.Recv(t.Context())
	require.ErrorIs(t, err, ErrWebSocketClientStreamClosed)
}

func TestWebSocketClientStreamNotify(t *testing.T) {
	srv := newScriptedServer(t)
	conn, _ := dialScripted(t, srv, nil)
	s := openStream(t, t.Context(), conn, nil)
	require.NoError(t, s.Notify(t.Context(), "note"))
	req := srv.next(t)
	require.False(t, req.HasID)
	require.Equal(t, "echo", req.Method)
	require.NoError(t, s.Close())
	require.ErrorIs(t, s.Notify(t.Context(), "after"), ErrWebSocketClientStreamClosed)
}

func TestWebSocketClientConnCloseFailsWaitersAndStreams(t *testing.T) {
	srv := newScriptedServer(t)
	conn, _ := dialScripted(t, srv, nil)
	s := openStream(t, t.Context(), conn, nil)
	require.NoError(t, s.Send(t.Context(), "held"))
	srv.next(t)
	require.NoError(t, conn.Close())
	_, err := s.Recv(t.Context())
	require.ErrorIs(t, err, ErrWebSocketClientConnClosed)
	require.ErrorIs(t, s.Send(t.Context(), "after"), ErrWebSocketClientConnClosed)
	require.False(t, conn.Acquire())
	require.NoError(t, s.Close())
	require.NoError(t, conn.Close())
}

// TestWebSocketClientConnAcquireAfterLastRelease checks that an Acquire that
// runs after the last Release dropped its reference, but before that release
// closed the socket, is refused: the release marks the connection closed in
// the same critical section, so it never closes a connection that a new
// stream was just given.
func TestWebSocketClientConnAcquireAfterLastRelease(t *testing.T) {
	srv := newScriptedServer(t)
	conn, _ := dialScripted(t, srv, nil)
	require.True(t, conn.Acquire())
	var acquired bool
	conn.afterLastRelease = func() {
		acquired = conn.Acquire()
	}
	require.NoError(t, conn.Release())
	require.False(t, acquired, "Acquire succeeded on a connection the last release was closing")
	require.ErrorIs(t, conn.Err(), ErrWebSocketClientConnClosed)
}

// TestWebSocketClientConnReleaseRacesAcquire runs the last Release and an
// Acquire concurrently: an Acquire that returns true always leaves a usable
// connection.
func TestWebSocketClientConnReleaseRacesAcquire(t *testing.T) {
	srv := newScriptedServer(t)
	for i := range 50 {
		conn, _ := dialScripted(t, srv, nil)
		require.True(t, conn.Acquire())
		start := make(chan struct{})
		acquired := make(chan bool, 1)
		released := make(chan error, 1)
		go func() {
			<-start
			acquired <- conn.Acquire()
		}()
		go func() {
			<-start
			released <- conn.Release()
		}()
		close(start)
		ok := <-acquired
		require.NoError(t, <-released)
		if ok {
			require.NoError(t, conn.Err(), "iteration %d: Acquire succeeded on a connection the last release closed", i)
			require.NoError(t, conn.Release())
		}
		require.ErrorIs(t, conn.Err(), ErrWebSocketClientConnClosed, "iteration %d", i)
	}
}

// TestWebSocketClientConnRefusedAcquireClosesSocket checks that Acquire on a
// connection whose read already failed, and that no stream holds, closes the
// socket instead of leaking it.
func TestWebSocketClientConnRefusedAcquireClosesSocket(t *testing.T) {
	srv := newScriptedServer(t)
	ws, resp, err := websocket.DefaultDialer.DialContext(t.Context(), srv.url, nil)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	errs := &recordedErrors{}
	conn := NewWebSocketClientConn(loomhttp.NewWebSocketStream(ws), errs.handle)
	serverConn := <-srv.conns
	require.NoError(t, serverConn.Close())
	<-conn.Done()
	require.False(t, conn.Acquire())
	_, err = ws.UnderlyingConn().Write([]byte{0})
	require.ErrorIs(t, err, net.ErrClosed, "the refused connection kept its socket open")
	require.Equal(t, []StreamErrorType{StreamErrorConnection}, errs.kindsSeen())
	require.NoError(t, conn.Close())
}

func newScriptedServer(t *testing.T) *scriptedServer {
	t.Helper()
	srv := &scriptedServer{requests: make(chan *RawRequest, 64), conns: make(chan *websocket.Conn, 1)}
	upgrader := &websocket.Upgrader{}
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		srv.conns <- conn
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req RawRequest
			if err := json.Unmarshal(data, &req); err != nil {
				t.Errorf("decode request: %v", err)
				return
			}
			srv.requests <- &req
		}
	}))
	t.Cleanup(hs.Close)
	srv.url = "ws" + strings.TrimPrefix(hs.URL, "http")
	return srv
}

// next returns the next request the server read.
func (srv *scriptedServer) next(t *testing.T) *RawRequest {
	t.Helper()
	select {
	case req := <-srv.requests:
		return req
	case <-time.After(5 * time.Second):
		t.Fatal("the server read no request")
		return nil
	}
}

// dialScripted connects a client connection to srv and returns it with the
// server side of the connection.
func dialScripted(t *testing.T, srv *scriptedServer, handler StreamErrorHandler) (*WebSocketClientConn, *websocket.Conn) {
	t.Helper()
	ws, resp, err := websocket.DefaultDialer.DialContext(t.Context(), srv.url, nil)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	conn := NewWebSocketClientConn(loomhttp.NewWebSocketStream(ws), handler)
	t.Cleanup(func() {
		if err := conn.Close(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Errorf("close connection: %v", err)
		}
	})
	return conn, <-srv.conns
}

// openStream acquires conn for a new stream of the "echo" method.
func openStream(t *testing.T, ctx context.Context, conn *WebSocketClientConn, handler StreamErrorHandler) *WebSocketClientStream {
	t.Helper()
	require.True(t, conn.Acquire())
	return NewWebSocketClientStream(ctx, conn, "echo", NewStreamConfig(WithErrorHandler(handler)))
}

func writeRaw(t *testing.T, conn *websocket.Conn, msg string) {
	t.Helper()
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(msg)))
}

func (r *recordedErrors) handle(_ context.Context, kind StreamErrorType, _ error, _ *RawResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.kinds = append(r.kinds, kind)
}

func (r *recordedErrors) kindsSeen() []StreamErrorType {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.kinds) == 0 {
		return nil
	}
	return append([]StreamErrorType(nil), r.kinds...)
}
