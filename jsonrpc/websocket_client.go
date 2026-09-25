package jsonrpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	loomhttp "github.com/CaliLuke/loom/http"
)

type (
	// WebSocketClientConn is the WebSocket connection that a generated
	// JSON-RPC client shares between its streams. One goroutine reads the
	// connection and routes every response to the request with the same id,
	// whichever stream sent it, because gorilla/websocket allows only one
	// concurrent reader. Request ids are unique on the connection. Writes go
	// through the write lock of the wrapped WebSocketStream.
	//
	// When the read fails, every request still waiting gets the read error
	// and no request can be sent any more. Close and the release of the last
	// stream close the connection the same way. The read failure, a
	// response with an unknown id and a server notification are reported,
	// with context.Background(), to the handler given to
	// NewWebSocketClientConn, as StreamErrorConnection, StreamErrorOrphaned
	// and StreamErrorNotification.
	//
	// A stream reports its own errors (protocol, parsing, timeout, write
	// failure) to its StreamConfig handler with its context. Protocol and
	// parsing errors are reported when Recv or Call consumes the response,
	// so an error response to a request that is never received is not
	// reported.
	//
	// The formal model of this design is jsonrpc/tla/WebSocketClientDemux.tla.
	WebSocketClientConn struct {
		ws      *loomhttp.WebSocketStream
		handler StreamErrorHandler
		done    chan struct{}
		// afterLastRelease, when set by a test, runs after the last
		// Release unlocked and before it closes the socket.
		afterLastRelease func()

		// mu guards the fields below. A stream lock is always taken
		// before mu, never after.
		mu     sync.Mutex
		calls  map[string]*webSocketCall
		nextID uint64
		refs   int
		err    error
	}

	// WebSocketClientStream is one stream of a generated JSON-RPC WebSocket
	// client on a shared WebSocketClientConn. Every request sent with Send
	// gets its own response, which Recv returns in send order; a response
	// that arrives before Recv is called is held until Recv returns it or
	// the stream ends. Generated code decodes the results.
	//
	// The stream ends when it is closed or when the context it was created
	// with is done: its waiting requests fail, their late responses become
	// orphans, and it releases the connection. The other streams on the
	// connection are not affected.
	WebSocketClientStream struct {
		conn   *WebSocketClientConn
		ctx    context.Context
		method string
		config *StreamConfig
		done   chan struct{}
		stop   func() bool

		// mu guards the fields below.
		mu       sync.Mutex
		own      map[string]*webSocketCall
		queue    []*webSocketCall
		closed   bool
		ended    bool
		doneSent bool
		err      error

		releaseOnce sync.Once
		releaseErr  error
	}

	// webSocketCall is one request waiting for its response.
	webSocketCall struct {
		id     string
		result chan webSocketResult
		timer  *time.Timer
	}

	// webSocketResult is the response to a request or the error that ended
	// the wait for it.
	webSocketResult struct {
		response *RawResponse
		err      error
	}
)

var (
	// ErrWebSocketClientConnClosed is the error of the requests that were
	// waiting on a WebSocketClientConn when it was closed, and of the
	// requests sent after.
	ErrWebSocketClientConnClosed = errors.New("jsonrpc: websocket client connection closed")

	// ErrWebSocketClientStreamClosed is returned by the operations of a
	// WebSocketClientStream after Close.
	ErrWebSocketClientStreamClosed = errors.New("stream closed")
)

// NewWebSocketClientConn starts routing the responses read from ws and
// returns the connection. handler, which may be nil, receives the events of
// the connection itself: the read failure (StreamErrorConnection), responses
// with an unknown id (StreamErrorOrphaned), server notifications
// (StreamErrorNotification), and a failure to close a connection that
// Acquire refused. They belong to no stream, so handler gets
// context.Background() rather than a stream context. Generated clients pass
// the ErrorHandler of their StreamConfig, the handler their streams report to.
func NewWebSocketClientConn(ws *loomhttp.WebSocketStream, handler StreamErrorHandler) *WebSocketClientConn {
	c := &WebSocketClientConn{
		ws:      ws,
		handler: handler,
		done:    make(chan struct{}),
		calls:   make(map[string]*webSocketCall),
	}
	go c.read()
	return c
}

// NewWebSocketClientStream returns a stream that sends the requests of the
// JSON-RPC method on conn. The caller must hold a reference on conn from
// Acquire, which the stream releases when it ends. The stream ends when ctx
// is done, and config sets its request timeout and error handler.
func NewWebSocketClientStream(ctx context.Context, conn *WebSocketClientConn, method string, config *StreamConfig) *WebSocketClientStream {
	s := &WebSocketClientStream{
		conn:   conn,
		ctx:    ctx,
		method: method,
		config: config,
		done:   make(chan struct{}),
		own:    make(map[string]*webSocketCall),
	}
	s.stop = context.AfterFunc(ctx, s.contextDone)
	return s
}

// Acquire takes a reference on the connection for a new stream. It pings
// the server first and returns false, without a reference, when the ping
// fails or the connection is closed. A refused connection that no stream
// holds is closed, so its socket is released; a failure to close it is
// reported to the handler. A connection also closes when its last reference
// is released.
func (c *WebSocketClientConn) Acquire() bool {
	if c.acquire() {
		return true
	}
	c.mu.Lock()
	unused := c.refs <= 0
	if unused && c.err == nil {
		c.err = ErrWebSocketClientConnClosed
	}
	c.mu.Unlock()
	if unused {
		if err := c.ws.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			c.report(StreamErrorConnection, fmt.Errorf("closing unusable connection: %w", err), nil)
		}
	}
	return false
}

// Release returns a reference taken with Acquire. Releasing the last one
// closes the connection and returns the result of closing the socket. The
// connection is marked closed in the same critical section that drops the
// last reference, so a concurrent Acquire either takes its reference first
// or is refused.
func (c *WebSocketClientConn) Release() error {
	c.mu.Lock()
	c.refs--
	last := c.refs <= 0
	if last && c.err == nil {
		c.err = ErrWebSocketClientConnClosed
	}
	c.mu.Unlock()
	if last {
		if c.afterLastRelease != nil {
			c.afterLastRelease()
		}
		return c.ws.Close()
	}
	return nil
}

// Close closes the connection: the requests waiting on it fail with
// ErrWebSocketClientConnClosed and no request can be sent any more. Later
// calls return the result of the first one.
func (c *WebSocketClientConn) Close() error {
	c.mu.Lock()
	if c.err == nil {
		c.err = ErrWebSocketClientConnClosed
	}
	c.mu.Unlock()
	return c.ws.Close()
}

// Err returns the error that ended the connection, or nil while it is
// usable.
func (c *WebSocketClientConn) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Done returns a channel that is closed once the connection stopped reading
// and failed the requests waiting on it.
func (c *WebSocketClientConn) Done() <-chan struct{} {
	return c.done
}

// Context returns the context the stream was created with.
func (s *WebSocketClientStream) Context() context.Context {
	return s.ctx
}

// Send writes a request with params and queues it for Recv. A failed write
// ends the connection for the stream: later calls return the error.
func (s *WebSocketClientStream) Send(ctx context.Context, params any) error {
	call, err := s.register(true)
	if err != nil {
		return err
	}
	return s.write(ctx, call, params)
}

// Recv waits for the response to the oldest request sent with Send and not
// received yet. It returns the response when it has no error and the error
// of the response otherwise, reporting it to the error handler with
// StreamErrorProtocol. When ctx is done first, the request stays the oldest
// one; when the request times out, it is forgotten and its late response
// is reported as orphaned.
func (s *WebSocketClientStream) Recv(ctx context.Context) (*RawResponse, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrWebSocketClientStreamClosed
	}
	if s.err != nil {
		err := s.err
		s.mu.Unlock()
		return nil, err
	}
	if len(s.queue) == 0 {
		s.mu.Unlock()
		return nil, errors.New("no pending requests - call Send() first")
	}
	call := s.queue[0]
	s.queue = s.queue[1:]
	s.mu.Unlock()
	return s.await(ctx, call, true)
}

// Call writes a request with params and waits for its response, which it
// returns as Recv does. When ctx is done first, the request is forgotten.
func (s *WebSocketClientStream) Call(ctx context.Context, params any) (*RawResponse, error) {
	call, err := s.register(false)
	if err != nil {
		return nil, err
	}
	if err := s.write(ctx, call, params); err != nil {
		return nil, err
	}
	return s.await(ctx, call, false)
}

// Notify writes a request with params and without an id, which the server
// does not answer.
func (s *WebSocketClientStream) Notify(ctx context.Context, params any) error {
	if err := s.usable(); err != nil {
		return err
	}
	request := &Request{JSONRPC: "2.0", Method: s.method, Params: params}
	if err := s.conn.ws.WriteJSON(ctx, request); err != nil {
		return s.writeFailed(err)
	}
	return nil
}

// ReportError passes a stream error to the error handler of the stream
// configuration, if any, with the stream context.
func (s *WebSocketClientStream) ReportError(errorType StreamErrorType, err error, response *RawResponse) {
	if s.config.ErrorHandler != nil {
		s.config.ErrorHandler(s.ctx, errorType, err, response)
	}
}

// Close ends the stream and releases its connection reference. It returns
// the result of closing the connection when the stream held its last
// reference, and nil otherwise. Later calls return the same result.
func (s *WebSocketClientStream) Close() error {
	s.mu.Lock()
	s.closed = true
	s.queue = nil
	s.mu.Unlock()
	s.stop()
	s.end(nil)
	return s.release()
}

// acquire takes a reference when the connection is usable and answers a
// ping.
func (c *WebSocketClientConn) acquire() bool {
	if c.Err() != nil {
		return false
	}
	if conn := c.ws.Conn(); conn == nil || conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)) != nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return false
	}
	c.refs++
	return true
}

// read routes the responses until the connection fails or is closed.
func (c *WebSocketClientConn) read() {
	defer close(c.done)
	for {
		var response RawResponse
		if err := c.ws.ReadJSON(context.Background(), &response); err != nil {
			c.fail(err)
			return
		}
		c.route(&response)
	}
}

// route delivers a response to the request with its id.
func (c *WebSocketClientConn) route(response *RawResponse) {
	if response.ID == nil {
		c.report(StreamErrorNotification, errors.New("received server notification"), response)
		return
	}
	id := IDToString(response.ID)
	c.mu.Lock()
	call, ok := c.calls[id]
	if ok {
		delete(c.calls, id)
	}
	c.mu.Unlock()
	if !ok {
		c.report(StreamErrorOrphaned, fmt.Errorf("received response for unknown ID: %s", id), response)
		return
	}
	call.complete(webSocketResult{response: response})
}

// fail ends the connection after a failed read and fails every waiting
// request. Marking the connection ended and taking the waiting requests
// happen under one lock, so no request registers after the fan-out.
func (c *WebSocketClientConn) fail(readErr error) {
	c.mu.Lock()
	unexpected := c.err == nil
	if unexpected {
		c.err = fmt.Errorf("failed to read response: %w", readErr)
	}
	err := c.err
	calls := c.calls
	c.calls = make(map[string]*webSocketCall)
	c.mu.Unlock()
	if unexpected {
		c.report(StreamErrorConnection, err, nil)
	}
	for _, call := range calls {
		call.complete(webSocketResult{err: err})
	}
}

// register gives call a new id and registers it, unless the connection has
// ended.
func (c *WebSocketClientConn) register(call *webSocketCall) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.nextID++
	call.id = strconv.FormatUint(c.nextID, 10)
	c.calls[call.id] = call
	return nil
}

// unregister forgets call, whose late response then becomes an orphan.
func (c *WebSocketClientConn) unregister(call *webSocketCall) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.calls[call.id] == call {
		delete(c.calls, call.id)
	}
}

// report passes a connection error to the handler.
func (c *WebSocketClientConn) report(errorType StreamErrorType, err error, response *RawResponse) {
	if c.handler != nil {
		c.handler(context.Background(), errorType, err, response)
	}
}

// complete hands the result to the waiter. Only the goroutine that removed
// the call from the connection calls it, so the buffered send never blocks.
func (call *webSocketCall) complete(result webSocketResult) {
	call.result <- result
}

// register registers a new request on the connection, queuing it for Recv
// when queued is set. The stream lock covers the closed check and the
// registration, so a closed stream registers nothing.
func (s *WebSocketClientStream) register(queued bool) (*webSocketCall, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ended {
		return nil, ErrWebSocketClientStreamClosed
	}
	if s.err != nil {
		return nil, s.err
	}
	call := &webSocketCall{result: make(chan webSocketResult, 1)}
	if err := s.conn.register(call); err != nil {
		s.failLocked(err)
		return nil, err
	}
	call.timer = time.NewTimer(s.config.RequestTimeout)
	s.own[call.id] = call
	if queued {
		s.queue = append(s.queue, call)
	}
	return call, nil
}

// write sends the registered request. A failed write forgets it and fails
// the stream.
func (s *WebSocketClientStream) write(ctx context.Context, call *webSocketCall, params any) error {
	request := &Request{JSONRPC: "2.0", Method: s.method, Params: params, ID: call.id}
	if err := s.conn.ws.WriteJSON(ctx, request); err != nil {
		call.timer.Stop()
		s.forget(call)
		return s.writeFailed(err)
	}
	return nil
}

// writeFailed fails the stream with a write error and reports it.
func (s *WebSocketClientStream) writeFailed(err error) error {
	s.mu.Lock()
	s.failLocked(err)
	s.mu.Unlock()
	s.ReportError(StreamErrorConnection, err, nil)
	return fmt.Errorf("failed to send request: %w", err)
}

// usable returns the error that stops the stream from sending, if any.
func (s *WebSocketClientStream) usable() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ended {
		return ErrWebSocketClientStreamClosed
	}
	return s.err
}

// await waits for the result of call. When ctx is done first, the call goes
// back to the head of the Recv queue if requeue is set and is forgotten
// otherwise.
func (s *WebSocketClientStream) await(ctx context.Context, call *webSocketCall, requeue bool) (*RawResponse, error) {
	select {
	case result := <-call.result:
		call.timer.Stop()
		s.dropOwn(call)
		return s.result(result)
	case <-call.timer.C:
		s.forget(call)
		err := fmt.Errorf("request timeout after %v", s.config.RequestTimeout)
		s.ReportError(StreamErrorTimeout, err, nil)
		return nil, err
	case <-ctx.Done():
		if requeue {
			s.requeue(call)
		} else {
			call.timer.Stop()
			s.forget(call)
		}
		return nil, ctx.Err()
	case <-s.done:
		call.timer.Stop()
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.err == nil {
			return nil, ErrWebSocketClientStreamClosed
		}
		return nil, s.err
	}
}

// result returns the response in result, or its error.
func (s *WebSocketClientStream) result(result webSocketResult) (*RawResponse, error) {
	if result.err != nil {
		return nil, result.err
	}
	if result.response.Error != nil {
		s.ReportError(StreamErrorProtocol, result.response.Error, result.response)
		return nil, result.response.Error
	}
	return result.response, nil
}

// requeue puts back a call whose Recv was canceled as the oldest one.
func (s *WebSocketClientStream) requeue(call *webSocketCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.queue = append([]*webSocketCall{call}, s.queue...)
}

// forget unregisters call from the connection and the stream.
func (s *WebSocketClientStream) forget(call *webSocketCall) {
	s.conn.unregister(call)
	s.dropOwn(call)
}

// dropOwn removes call from the requests of the stream.
func (s *WebSocketClientStream) dropOwn(call *webSocketCall) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.own, call.id)
}

// failLocked records the first error of the stream, forgets its requests
// and wakes its waiters. It keeps the connection reference: Close or the
// end of the stream context releases it.
func (s *WebSocketClientStream) failLocked(err error) {
	if s.err == nil {
		s.err = err
	}
	s.stopLocked()
}

// stopLocked forgets the requests of the stream and wakes its waiters.
func (s *WebSocketClientStream) stopLocked() {
	for id, call := range s.own {
		s.conn.unregister(call)
		call.timer.Stop()
		delete(s.own, id)
	}
	if !s.doneSent {
		s.doneSent = true
		close(s.done)
	}
}

// end ends the stream, recording err when it is the first error. The
// caller releases the connection reference.
func (s *WebSocketClientStream) end(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	s.ended = true
	if err != nil && s.err == nil {
		s.err = err
	}
	s.stopLocked()
}

// contextDone ends the stream when its context is done and releases the
// connection reference, reporting a failure to close the connection.
func (s *WebSocketClientStream) contextDone() {
	s.end(s.ctx.Err())
	if err := s.release(); err != nil {
		s.ReportError(StreamErrorConnection, fmt.Errorf("closing connection: %w", err), nil)
	}
}

// release returns the connection reference once and records the result.
func (s *WebSocketClientStream) release() error {
	s.releaseOnce.Do(func() {
		s.releaseErr = s.conn.Release()
	})
	return s.releaseErr
}
