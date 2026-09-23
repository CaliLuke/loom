package http

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type (
	// Upgrader is an HTTP connection that is able to upgrade to websocket.
	Upgrader interface {
		// Upgrade upgrades the HTTP connection to the websocket protocol.
		Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
	}

	// Dialer creates a websocket connection to a given URL.
	Dialer interface {
		// DialContext creates a client connection to the websocket server.
		DialContext(ctx context.Context, url string, h http.Header) (*websocket.Conn, *http.Response, error)
	}

	// ConnConfigureFunc is used to configure a websocket connection with
	// custom handlers. The cancel function cancels the request context when
	// invoked in the configure function.
	ConnConfigureFunc func(conn *websocket.Conn, cancel context.CancelFunc) *websocket.Conn

	// WebSocketStream owns the lifecycle for a WebSocket connection used by
	// generated HTTP streaming clients and servers.
	//
	// Close is terminal: once it has been called, the stream never becomes
	// usable again, even when the connection is attached later with SetConn.
	// A failed ReadMessage is also terminal for reads.
	WebSocketStream struct {
		// connLock guards conn, closed, closeErr, and readErr.
		connLock sync.RWMutex
		conn     *websocket.Conn
		closed   bool
		closeErr error
		readErr  error
		policy   StreamWritePolicy

		writeLock sync.Mutex
	}
)

var (
	// ErrWebSocketStreamClosed is returned when generated code uses a stream
	// after its WebSocket connection has been closed.
	ErrWebSocketStreamClosed = errors.New("loom http websocket stream closed")
)

// NewWebSocketStream wraps conn with shared generated-stream lifecycle
// behavior. A nil conn is allowed so generated server streams can be allocated
// before the WebSocket upgrade happens.
func NewWebSocketStream(conn *websocket.Conn, policies ...StreamWritePolicy) *WebSocketStream {
	return &WebSocketStream{
		conn:   conn,
		policy: firstStreamWritePolicy(policies),
	}
}

// Conn returns the wrapped Gorilla WebSocket connection.
func (s *WebSocketStream) Conn() *websocket.Conn {
	if s == nil {
		return nil
	}
	s.connLock.RLock()
	defer s.connLock.RUnlock()
	return s.conn
}

// SetConn replaces the wrapped Gorilla WebSocket connection. Generated servers
// call it once the lazy upgrade succeeds.
//
// If the stream has already been closed, SetConn still records conn so Conn
// reports that the HTTP connection was upgraded, but it closes conn
// immediately. Later reads and writes return ErrWebSocketStreamClosed, and any
// error from closing conn is reported by subsequent Close calls.
func (s *WebSocketStream) SetConn(conn *websocket.Conn) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	if s.closed && conn != nil && conn != s.conn {
		if err := conn.Close(); err != nil {
			s.closeErr = errors.Join(s.closeErr, err)
		}
	}
	s.conn = conn
}

// ReadJSON reads one JSON WebSocket message into v while honoring ctx
// cancellation. The message is decoded with encoding/json/v2, the same
// decoder used for HTTP request bodies. An empty or whitespace-only message
// returns io.ErrUnexpectedEOF. A decode error leaves the stream readable, but
// a failed connection read is terminal, as described for ReadMessage. It
// returns ErrWebSocketStreamClosed when no connection is attached or the
// stream has been closed. At most one goroutine may read from the stream at a
// time.
func (s *WebSocketStream) ReadJSON(ctx context.Context, v any) error {
	data, err := s.ReadMessage(ctx)
	if err != nil {
		return err
	}
	if jsonWhitespaceOnly(data) {
		return io.ErrUnexpectedEOF
	}
	return json.Unmarshal(data, v)
}

// ReadMessage reads the payload of one text or binary WebSocket message while
// honoring ctx cancellation. Every error it returns is a connection, close,
// or context failure, never a payload decoding failure. A failed connection
// read is terminal because a Gorilla connection cannot recover from one: the
// stream records the failure and returns it from every later ReadMessage and
// ReadJSON call without reading the connection again. A context that is
// already done returns its error without reading or recording anything. A
// context canceled during a read returns its error and closes the stream, so
// later calls return ErrWebSocketStreamClosed, which is also returned when no
// connection is attached or the stream has been closed. At most one goroutine
// may read from the stream at a time.
func (s *WebSocketStream) ReadMessage(ctx context.Context) ([]byte, error) {
	var data []byte
	err := s.withContext(ctx, func() error {
		conn, err := s.readableConn()
		if err != nil {
			return err
		}
		_, data, err = conn.ReadMessage()
		if err != nil {
			s.recordReadErr(err)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

// WriteJSON writes one JSON WebSocket frame while honoring ctx cancellation.
// Concurrent calls are serialized. It returns ErrWebSocketStreamClosed when no
// connection is attached or the stream has been closed.
func (s *WebSocketStream) WriteJSON(ctx context.Context, v any) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	return s.withContext(ctx, func() error {
		conn, err := s.activeConn()
		if err != nil {
			return err
		}
		return s.writeJSONWithDeadline(ctx, conn, v)
	})
}

func (s *WebSocketStream) writeJSONWithDeadline(ctx context.Context, conn *websocket.Conn, v any) (err error) {
	deadline, bounded := streamOperationDeadline(ctx, s.policy)
	if !bounded {
		return conn.WriteJSON(v)
	}
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	defer func() {
		if clearErr := conn.SetWriteDeadline(time.Time{}); clearErr != nil && err == nil {
			err = clearErr
		}
	}()
	return conn.WriteJSON(v)
}

func firstStreamWritePolicy(policies []StreamWritePolicy) StreamWritePolicy {
	if len(policies) == 0 {
		return StreamWritePolicy{}
	}
	return policies[0]
}

func streamOperationDeadline(ctx context.Context, policy StreamWritePolicy) (time.Time, bool) {
	var deadline time.Time
	if timeout := policy.Timeout(); timeout > 0 {
		deadline = time.Now().Add(timeout)
	}
	if ctxDeadline, ok := ctx.Deadline(); ok && (deadline.IsZero() || ctxDeadline.Before(deadline)) {
		deadline = ctxDeadline
	}
	return deadline, !deadline.IsZero()
}

// WriteClose writes a close control frame. It does not close the underlying
// connection; call Close after WriteClose to release the socket.
func (s *WebSocketStream) WriteClose(message string) error {
	conn := s.Conn()
	if conn == nil {
		return nil
	}
	return conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, message),
		time.Now().Add(time.Second),
	)
}

// Close marks the stream closed and closes the attached WebSocket connection,
// if any. The first call is terminal whether or not a connection is attached:
// a connection attached later with SetConn is closed on arrival. Repeated
// calls do not close the connection again and return the recorded close
// error.
func (s *WebSocketStream) Close() error {
	if s == nil {
		return nil
	}
	s.connLock.Lock()
	defer s.connLock.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	if s.conn != nil {
		s.closeErr = s.conn.Close()
	}
	return s.closeErr
}

func (s *WebSocketStream) activeConn() (*websocket.Conn, error) {
	s.connLock.RLock()
	defer s.connLock.RUnlock()
	if s.closed || s.conn == nil {
		return nil, ErrWebSocketStreamClosed
	}
	return s.conn, nil
}

func (s *WebSocketStream) readableConn() (*websocket.Conn, error) {
	s.connLock.RLock()
	defer s.connLock.RUnlock()
	if s.closed || s.conn == nil {
		return nil, ErrWebSocketStreamClosed
	}
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.conn, nil
}

func (s *WebSocketStream) recordReadErr(err error) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	if s.readErr == nil {
		s.readErr = err
	}
}

func (s *WebSocketStream) withContext(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	closec := make(chan error, 1)
	stop := context.AfterFunc(ctx, func() {
		closec <- s.Close()
	})

	err := fn()
	if !stop() {
		if closeErr := <-closec; closeErr != nil && err == nil {
			err = closeErr
		}
	}
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}
