package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
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

	// WebSocketStream owns one physical reader, a bounded inbox and terminal
	// shutdown for a WebSocket connection. Call SetConn to attach it. Only one
	// application goroutine may receive at a time; writes are serialized.
	WebSocketStream struct {
		observation      *loomtransport.StreamObserver
		connLock         sync.Mutex
		conn             *websocket.Conn
		closed           bool
		closeErr         error
		readErr          error
		policy           StreamWritePolicy
		config           WebSocketConfig
		inbox            [][]byte
		bufferedBytes    int64
		bufferedMessages int
		notify           chan struct{}
		terminalDone     chan struct{}
		pumpDone         chan struct{}
		stopLifetime     func() bool
		cancel           context.CancelFunc
		attachLock       sync.Mutex
		writeToken       chan struct{}
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
	stream := &WebSocketStream{
		config:       config,
		policy:       firstStreamWritePolicy(policies),
		notify:       make(chan struct{}),
		terminalDone: make(chan struct{}),
		writeToken:   make(chan struct{}, 1),
	}
	stream.writeToken <- struct{}{}
	return stream
}

// NewServerWebSocketStream allocates a server connection owner with immutable
// observation identity. Client connections use NewWebSocketStream and never
// inherit inbound request observation through SetConn.
func NewServerWebSocketStream(ctx context.Context, kind loomtransport.TransportKind, config WebSocketConfig, policies ...StreamWritePolicy) *WebSocketStream {
	stream := NewWebSocketStream(config, policies...)
	stream.observation = loomtransport.NewStreamObserver(ctx, kind)
	return stream
}

// Conn returns the wrapped Gorilla WebSocket connection. The stream owns all
// physical reads; install read handlers in the configuration hook before
// attachment instead of reading or mutating reader state through Conn.
func (s *WebSocketStream) Conn() *websocket.Conn {
	if s == nil {
		return nil
	}
	s.connLock.Lock()
	defer s.connLock.Unlock()
	return s.conn
}

// SetConn configures and adopts conn for ctx's connection lifetime. It invokes
// the hook before starting the sole physical reader. cancel, when nonnil, is
// called on terminal shutdown and is also passed to the hook. The hook owns
// releasing its input when it returns a replacement. A nil result closes input.
// Reattaching the same connection is a no-op. A different or late connection is
// rejected and closed. Cancellation during configuration cannot revive a stream.
func (s *WebSocketStream) SetConn(ctx context.Context, conn *websocket.Conn, cancel context.CancelFunc) error {
	if conn == nil {
		return fmt.Errorf("loom http websocket connection is nil")
	}
	defer s.observation.Drain()
	s.attachLock.Lock()
	defer s.attachLock.Unlock()
	s.connLock.Lock()
	if conn == s.conn {
		s.connLock.Unlock()
		return nil
	}
	s.observation.RecordOpen()
	if s.closed {
		s.conn = conn
		s.closeErr = errors.Join(s.closeErr, conn.Close())
		s.observation.RecordClose()
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
	var configureErr error
	if ctx.Err() == nil && s.config.configure != nil {
		configured := s.config.configure(conn, cancel)
		if configured == nil {
			configureErr = fmt.Errorf("connection configure function returned a nil connection")
		} else {
			conn = configured
		}
	}
	s.connLock.Lock()
	s.conn = conn
	if s.closed {
		s.closeErr = errors.Join(s.closeErr, conn.Close())
		s.observation.RecordClose()
		err := errors.Join(ErrWebSocketStreamClosed, s.closeErr)
		s.connLock.Unlock()
		return err
	}
	s.cancel = cancel
	if err := errors.Join(configureErr, ctx.Err()); err != nil {
		s.connLock.Unlock()
		s.stop(nil, 0, nil, false)
		return errors.Join(err, s.closeResult())
	}
	s.pumpDone = make(chan struct{})
	s.stopLifetime = context.AfterFunc(ctx, func() {
		s.stop(nil, 0, nil, false)
	})
	s.connLock.Unlock()
	go s.readPump(conn)
	return nil
}

// WriteJSON writes one JSON WebSocket frame while honoring ctx cancellation.
// Concurrent calls are serialized. It returns ErrWebSocketStreamClosed when no
// connection is attached or the stream has been closed.
func (s *WebSocketStream) WriteJSON(ctx context.Context, v any) (err error) {
	defer func() {
		s.observation.Drain()
		if err != nil && s.observation != nil {
			s.observation.ForContext(ctx).Failure(streamFailureReason(loomtransport.ReasonStreamWriteFailed, loomtransport.ReasonStreamWriteTimeout, err))
		}
	}()
	if err := s.acquireWrite(ctx); err != nil {
		return err
	}
	defer s.releaseWrite()

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

func (s *WebSocketStream) activeConn() (*websocket.Conn, error) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	if s.closed || s.conn == nil {
		return nil, ErrWebSocketStreamClosed
	}
	return s.conn, nil
}

func (s *WebSocketStream) acquireWrite(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.writeToken:
		return nil
	}
}

func (s *WebSocketStream) releaseWrite() {
	s.writeToken <- struct{}{}
}

func (s *WebSocketStream) withContext(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	closec := make(chan error, 1)
	stop := context.AfterFunc(ctx, func() {
		s.stop(nil, 0, nil, false)
		closec <- s.closeResult()
	})
	defer stop()

	err := fn()
	if err != nil && ctx.Err() != nil {
		s.stop(nil, 0, nil, false)
	}
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
