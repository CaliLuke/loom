package http

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"

	"github.com/gorilla/websocket"
)

// ErrWebSocketBufferFull is returned after the inbound byte or message-count
// budget is exceeded. The connection closes with code 1013 and cannot recover.
var ErrWebSocketBufferFull = errors.New("loom http websocket inbound buffer full")

// ReadJSON receives and decodes one message while retaining its admission
// reservation through decoding. Empty/whitespace input returns
// io.ErrUnexpectedEOF. Decode failures are recoverable; connection failures are
// terminal. An already-done ctx is nondestructive; active cancellation closes.
func (s *WebSocketStream) ReadJSON(ctx context.Context, value any) error {
	return s.DecodeMessage(ctx, func(data []byte) error {
		if jsonWhitespaceOnly(data) {
			return io.ErrUnexpectedEOF
		}
		return json.Unmarshal(data, value)
	})
}

// DecodeMessage receives one message and holds its byte/count reservation until
// decode returns. The callback must not retain data or call another receive.
// Its error is returned unchanged and does not itself terminate the connection.
// Use this seam for protocol decoders that must distinguish decode errors from
// connection failures. Like ReadJSON, at most one caller may receive at a time.
func (s *WebSocketStream) DecodeMessage(ctx context.Context, decode func([]byte) error) error {
	return s.withContext(ctx, func() error {
		data, err := s.nextMessage(ctx)
		if err != nil {
			return err
		}
		defer s.releaseMessage(data)
		return decode(data)
	})
}

// ReadMessage receives one text or binary message. Ownership of the returned
// bytes passes to the caller; their admission reservation is released on return.
// A natural peer read failure releases the socket immediately, preserves already
// admitted FIFO messages for live-context readers, and then returns its stable
// cause. Explicit close and overload reject queued work. Expanded-message
// overflow wraps websocket.ErrReadLimit; aggregate overflow wraps
// ErrWebSocketBufferFull. Only one caller may receive at a time.
func (s *WebSocketStream) ReadMessage(ctx context.Context) ([]byte, error) {
	var data []byte
	err := s.withContext(ctx, func() error {
		var err error
		data, err = s.nextMessage(ctx)
		if err == nil {
			s.releaseMessage(data)
		}
		return err
	})
	return data, err
}

func (s *WebSocketStream) readPump(conn *websocket.Conn) {
	defer close(s.pumpDone)
	for {
		_, reader, err := conn.NextReader()
		var data []byte
		if err == nil {
			data, err = io.ReadAll(io.LimitReader(reader, s.config.MaxMessageBytes()+1))
		}
		code := 0
		if err == nil && int64(len(data)) > s.config.MaxMessageBytes() {
			err = websocket.ErrReadLimit
			code = websocket.CloseMessageTooBig
		}
		if err != nil {
			s.stop(err, code, nil, !errors.Is(err, websocket.ErrReadLimit))
			return
		}
		s.connLock.Lock()
		if s.closed {
			s.connLock.Unlock()
			return
		}
		budget := s.config.MaxBufferedBytes() - s.config.MaxMessageBytes()
		if int64(len(data)) > budget-s.bufferedBytes || s.bufferedMessages >= s.config.MaxBufferedMessages() {
			s.connLock.Unlock()
			s.stop(ErrWebSocketBufferFull, websocket.CloseTryAgainLater, nil, false)
			return
		}
		s.inbox = append(s.inbox, data)
		s.bufferedBytes += int64(len(data))
		s.bufferedMessages++
		s.signalLocked()
		s.connLock.Unlock()
	}
}

func (s *WebSocketStream) nextMessage(ctx context.Context) ([]byte, error) {
	for {
		s.connLock.Lock()
		if len(s.inbox) > 0 {
			data := s.inbox[0]
			s.inbox[0] = nil
			s.inbox = s.inbox[1:]
			s.connLock.Unlock()
			return data, nil
		}
		if s.closed {
			s.connLock.Unlock()
			<-s.terminalDone
			s.connLock.Lock()
			err := s.readErr
			s.connLock.Unlock()
			if err != nil {
				return nil, err
			}
			return nil, ErrWebSocketStreamClosed
		}
		if s.conn == nil {
			s.connLock.Unlock()
			return nil, ErrWebSocketStreamClosed
		}
		notify := s.notify
		s.connLock.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-notify:
		}
	}
}

func (s *WebSocketStream) releaseMessage(data []byte) {
	s.connLock.Lock()
	defer s.connLock.Unlock()
	s.bufferedBytes -= int64(len(data))
	s.bufferedMessages--
}

func (s *WebSocketStream) signalLocked() {
	close(s.notify)
	s.notify = make(chan struct{})
}
