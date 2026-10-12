package http

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const webSocketFinalJSON = -1

// Close terminally releases the connection immediately. The first terminal
// transition owns the cause; repeated calls return its recorded cleanup error.
// Close before attachment rejects and releases any later connection.
func (s *WebSocketStream) Close() error {
	if s == nil {
		return nil
	}
	s.stop(nil, 0, nil, false)
	return s.closeResult()
}

// CloseWithStatus terminally attempts a close control frame, then releases the
// socket even when the bounded one-second write fails. Codes must be standard
// sendable close codes or in 3000..4999; reason must be valid UTF-8 of at most
// 123 bytes. Invalid arguments still release the socket and return an error.
// Repeated valid calls preserve the first terminal cause and cleanup result.
func (s *WebSocketStream) CloseWithStatus(code int, reason string) error {
	valid := code >= 3000 && code <= 4999
	switch code {
	case 1000, 1001, 1002, 1003, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014:
		valid = true
	}
	if !valid || len(reason) > 123 || !utf8.ValidString(reason) {
		return errors.Join(fmt.Errorf("loom http websocket invalid close status or reason"), s.Close())
	}
	s.stop(nil, code, reason, false)
	return s.closeResult()
}

// CloseWithFinalJSON terminally writes value and closes the connection. Its
// one-second deadline covers write-lock acquisition and the write, independently
// of the normal write policy. Physical release is unconditional. HTTP client
// Close uses a nil value to retain its final JSON-null framing; CloseAndRecv
// must instead write null without terminally closing the connection.
func (s *WebSocketStream) CloseWithFinalJSON(value any) error {
	s.stop(nil, webSocketFinalJSON, value, false)
	return s.closeResult()
}

// RunHandler runs one handler and owns its deferred connection cleanup without
// forcing a lazy upgrade. Success closes with 1000; an error or panic closes
// with 1011 and a generic reason. The original panic is propagated even if the
// cleanup reporter panics. A nonnil reporter receives only cleanup failures,
// preserving the handler error; nil joins cleanup failures into the returned
// error. While unwinding a panic, the panic retains precedence over cleanup.
func (s *WebSocketStream) RunHandler(run func() error, reportCloseError func(error)) (err error) {
	defer func() {
		panicked := recover()
		if panicked != nil {
			defer func() {
				panic(panicked)
			}()
		}
		code := websocket.CloseNormalClosure
		if err != nil || panicked != nil {
			code = websocket.CloseInternalServerErr
		}
		s.stop(nil, code, nil, false)
		if closeErr := s.closeResult(); closeErr != nil {
			if reportCloseError != nil {
				reportCloseError(closeErr)
			} else {
				err = errors.Join(err, closeErr)
			}
		}
	}()
	return run()
}

func (s *WebSocketStream) stop(readCause error, code int, final any, keepQueue bool) {
	s.connLock.Lock()
	if s.closed {
		s.connLock.Unlock()
		return
	}
	s.closed = true
	s.readErr = readCause
	if !keepQueue {
		for _, data := range s.inbox {
			s.bufferedBytes -= int64(len(data))
			s.bufferedMessages--
		}
		s.inbox = nil
	}
	conn, cancel, stopLifetime := s.conn, s.cancel, s.stopLifetime
	s.signalLocked()
	s.connLock.Unlock()
	if stopLifetime != nil {
		stopLifetime()
	}
	if cancel != nil {
		cancel()
	}
	var closeErr error
	if conn != nil {
		closeErr = s.releaseConnection(conn, code, final)
	}
	s.connLock.Lock()
	s.closeErr = errors.Join(s.closeErr, closeErr)
	if readCause != nil && s.closeErr != nil {
		s.readErr = errors.Join(readCause, s.closeErr)
	}
	close(s.terminalDone)
	s.connLock.Unlock()
}

func (s *WebSocketStream) releaseConnection(conn *websocket.Conn, code int, final any) error {
	var releaseOnce sync.Once
	var physicalErr error
	release := func() {
		releaseOnce.Do(func() {
			physicalErr = conn.Close()
		})
	}
	var writeErr error
	switch {
	case code == webSocketFinalJSON:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		stop := context.AfterFunc(ctx, release)
		if writeErr = s.acquireWrite(ctx); writeErr == nil {
			writeErr = s.writeJSONWithDeadline(ctx, conn, final)
			s.releaseWrite()
		}
		stop()
	case code > 0:
		reason, supplied := final.(string)
		if !supplied && code == websocket.CloseInternalServerErr {
			reason = "handler failed"
		}
		writeErr = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	}
	release()
	return errors.Join(writeErr, physicalErr)
}

func (s *WebSocketStream) closeResult() error {
	<-s.terminalDone
	s.connLock.Lock()
	defer s.connLock.Unlock()
	return s.closeErr
}
