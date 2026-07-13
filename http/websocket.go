package http

import (
	"context"
	"errors"
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
	WebSocketStream struct {
		conn     *websocket.Conn
		connLock sync.RWMutex
		policy   StreamWritePolicy

		writeLock sync.Mutex
		closeOnce sync.Once
		closeErr  error
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

// SetConn replaces the wrapped Gorilla WebSocket connection.
func (s *WebSocketStream) SetConn(conn *websocket.Conn) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	s.conn = conn
}

// ReadJSON reads one JSON WebSocket frame while honoring ctx cancellation.
func (s *WebSocketStream) ReadJSON(ctx context.Context, v any) error {
	return s.withContext(ctx, func() error {
		conn := s.Conn()
		if conn == nil {
			return ErrWebSocketStreamClosed
		}
		return conn.ReadJSON(v)
	})
}

// WriteJSON writes one JSON WebSocket frame while honoring ctx cancellation.
func (s *WebSocketStream) WriteJSON(ctx context.Context, v any) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	return s.withContext(ctx, func() error {
		conn := s.Conn()
		if conn == nil {
			return ErrWebSocketStreamClosed
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

// Close closes the WebSocket connection at most once.
func (s *WebSocketStream) Close() error {
	if s == nil {
		return nil
	}
	conn := s.Conn()
	if conn == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		s.closeErr = conn.Close()
	})
	return s.closeErr
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
