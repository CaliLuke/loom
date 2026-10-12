package http

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
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
	// invoked in the configure function. The framing limit is installed before
	// this hook runs. A hook that returns a different connection owns releasing
	// the input without invalidating the returned connection. The stream owns
	// the returned connection; its expanded-message limit still applies. A nil
	// result is terminal: the stream records and closes the input connection.
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
		config   WebSocketConfig

		attachLock sync.Mutex
		writeLock  sync.Mutex
	}
)

var (
	// ErrWebSocketStreamClosed is returned when generated code uses a stream
	// after its WebSocket connection has been closed.
	ErrWebSocketStreamClosed = errors.New("loom http websocket stream closed")
)

// NewWebSocketStream allocates the shared connection owner with config. Attach
// a dialed or upgraded connection with SetConn before reading or writing.
func NewWebSocketStream(config WebSocketConfig, policies ...StreamWritePolicy) *WebSocketStream {
	return &WebSocketStream{
		config: config,
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

// SetConn configures and adopts conn. It installs the configured framing limit,
// invokes the connection hook with cancel, and adopts the hook's result. The
// hook owns releasing conn if it returns a different connection.
//
// Reattaching the same connection is a no-op. Attaching a different connection
// while one is owned fails and closes the incoming connection. After terminal
// Close, an incoming connection is recorded and closed without invoking the hook,
// and SetConn returns ErrWebSocketStreamClosed. Close racing the hook closes its
// returned connection on arrival. A nil connection is rejected.
func (s *WebSocketStream) SetConn(conn *websocket.Conn, cancel context.CancelFunc) error {
	if conn == nil {
		return fmt.Errorf("loom http websocket connection is nil")
	}
	s.attachLock.Lock()
	defer s.attachLock.Unlock()
	s.connLock.Lock()
	if conn == s.conn {
		s.connLock.Unlock()
		return nil
	}
	if s.closed {
		s.conn = conn
		s.closeErr = errors.Join(s.closeErr, conn.Close())
		err := errors.Join(ErrWebSocketStreamClosed, s.closeErr)
		s.connLock.Unlock()
		return err
	}
	if s.conn != nil {
		s.connLock.Unlock()
		return errors.Join(fmt.Errorf("loom http websocket stream already has a connection"), conn.Close())
	}
	s.connLock.Unlock()

	conn.SetReadLimit(s.config.MaxMessageBytes())
	if s.config.configure != nil {
		configured := s.config.configure(conn, cancel)
		if configured == nil {
			s.connLock.Lock()
			s.conn = conn
			s.closed = true
			s.closeErr = errors.Join(s.closeErr, conn.Close())
			err := errors.Join(fmt.Errorf("connection configure function returned a nil connection"), s.closeErr)
			s.connLock.Unlock()
			return err
		}
		conn = configured
	}
	s.connLock.Lock()
	defer s.connLock.Unlock()
	s.conn = conn
	if s.closed {
		s.closeErr = errors.Join(s.closeErr, conn.Close())
		return errors.Join(ErrWebSocketStreamClosed, s.closeErr)
	}
	return nil
}

// ReadJSON reads one JSON WebSocket message into v while honoring ctx
// cancellation. The message is decoded with encoding/json/v2, the same
// decoder used for HTTP request bodies. An empty or whitespace-only message
// returns io.ErrUnexpectedEOF. A decode error leaves the stream readable, but
// a failed connection read is terminal, as described for ReadMessage. It
// returns ErrWebSocketStreamClosed when no connection is attached. ReadMessage
// describes the terminal cause returned after closure. At most one goroutine
// may read from the stream at a time.
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
// stream records the failure and returns it from later ReadMessage and
// ReadJSON calls without reading the connection again, until Close replaces
// a nonclosing read failure with ErrWebSocketStreamClosed. A context that is
// already done returns its error without reading or recording anything. A
// context canceled during a read returns its error and closes the stream, so
// later calls return ErrWebSocketStreamClosed unless read-failure cleanup
// already closed the stream with its own terminal cause. The closed sentinel
// is also returned when no connection is attached. At most one goroutine
// may read from the stream at a time.
//
// Messages exceeding the configured expanded-byte limit are rejected before
// decoding. The stream sends close code 1009 with a bounded control write,
// closes the socket, and returns an error wrapping websocket.ErrReadLimit.
// The same terminal error is returned on subsequent reads.
func (s *WebSocketStream) ReadMessage(ctx context.Context) ([]byte, error) {
	var data []byte
	err := s.withContext(ctx, func() error {
		conn, err := s.readableConn()
		if err != nil {
			return err
		}
		_, reader, err := conn.NextReader()
		if err == nil {
			data, err = io.ReadAll(io.LimitReader(reader, s.config.MaxMessageBytes()+1))
		}
		if err == nil && int64(len(data)) > s.config.MaxMessageBytes() {
			closeErr := conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseMessageTooBig, ""), time.Now().Add(time.Second))
			err = errors.Join(websocket.ErrReadLimit, closeErr)
		}
		if err != nil {
			if errors.Is(err, websocket.ErrReadLimit) {
				err = errors.Join(err, s.close(err))
			}
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
// error. If this call closes the stream, later reads return
// ErrWebSocketStreamClosed, including after an earlier nonclosing read failure.
// If read-failure cleanup already closed it, later reads retain that terminal
// cause. The first close transition owns the cause; errors from reads unblocked
// by closing the socket cannot replace it.
func (s *WebSocketStream) Close() error {
	return s.close(nil)
}

func (s *WebSocketStream) close(readCause error) error {
	if s == nil {
		return nil
	}
	s.connLock.Lock()
	defer s.connLock.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	s.readErr = readCause
	if s.conn != nil {
		s.closeErr = s.conn.Close()
	}
	if readCause != nil {
		s.readErr = errors.Join(readCause, s.closeErr)
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
	if s.readErr != nil {
		return nil, s.readErr
	}
	if s.closed || s.conn == nil {
		return nil, ErrWebSocketStreamClosed
	}
	return s.conn, nil
}

func (s *WebSocketStream) recordReadErr(err error) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	if !s.closed && s.readErr == nil {
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
