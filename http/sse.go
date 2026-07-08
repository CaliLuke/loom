package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	sse "github.com/tmaxmax/go-sse"
)

type (
	// SSEEvent represents a parsed server-sent event frame.
	SSEEvent struct {
		ID   string
		Type string
		Data string
	}

	// SSEMessage describes an SSE frame to write to an output stream.
	SSEMessage struct {
		ID          string
		Type        string
		Data        string
		RetryMillis int64
	}

	// SSEStreamReader reads framed Server-Sent Events from a response body.
	SSEStreamReader struct {
		body     io.ReadCloser
		buffer   []byte
		readLock sync.Mutex
		lock     sync.Mutex
		closed   bool
	}

	sseReadResult struct {
		n   int
		err error
	}
)

// NewSSEStreamReader returns a reader for framed Server-Sent Events.
func NewSSEStreamReader(body io.ReadCloser) *SSEStreamReader {
	return &SSEStreamReader{
		body:   body,
		buffer: make([]byte, 0, 4096),
	}
}

// ReadEvent reads a single raw SSE event frame.
func (r *SSEStreamReader) ReadEvent(ctx context.Context) ([]byte, error) {
	const bufSize = 4096

	r.readLock.Lock()
	defer r.readLock.Unlock()

	event, ok := r.checkBuffer()
	if ok {
		return event, nil
	}

	eventData := event
	wasNewline := len(eventData) > 0 && eventData[len(eventData)-1] == '\n'
	buf := make([]byte, bufSize)
	for {
		body, done, err := r.currentBody(ctx, eventData)
		if err != nil {
			return eventData, err
		}
		if done {
			return eventData, nil
		}

		n, err := r.readChunk(ctx, body, buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		var complete bool
		eventData, wasNewline, complete = r.appendChunk(eventData, wasNewline, buf[:n])
		if complete {
			return eventData, nil
		}

		if errors.Is(err, io.EOF) {
			if len(eventData) > 0 {
				return eventData, nil
			}
			return nil, io.EOF
		}
	}
}

func (r *SSEStreamReader) currentBody(ctx context.Context, eventData []byte) (io.ReadCloser, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if r.closed {
		if len(eventData) > 0 {
			return nil, true, nil
		}
		return nil, false, io.EOF
	}
	return r.body, false, nil
}

func (r *SSEStreamReader) readChunk(ctx context.Context, body io.Reader, buf []byte) (int, error) {
	readc := make(chan sseReadResult, 1)
	go func() {
		n, err := body.Read(buf)
		readc <- sseReadResult{n: n, err: err}
	}()

	select {
	case result := <-readc:
		return result.n, result.err
	case <-ctx.Done():
		select {
		case result := <-readc:
			return result.n, result.err
		default:
			if err := r.Close(); err != nil {
				// Preserve the client contract: cancellation is the
				// observable cause even when closing the body fails.
				return 0, ctx.Err()
			}
			return 0, ctx.Err()
		}
	}
}

func (r *SSEStreamReader) appendChunk(eventData []byte, wasNewline bool, chunk []byte) ([]byte, bool, bool) {
	for i, b := range chunk {
		eventData = append(eventData, b)
		if b == '\n' && wasNewline {
			if i+1 < len(chunk) {
				r.lock.Lock()
				// Copy the leftover into a fresh slice so r.buffer never shares
				// a backing array with the returned event accumulator.
				r.buffer = append([]byte(nil), chunk[i+1:]...)
				r.lock.Unlock()
			}
			return eventData, wasNewline, true
		}
		wasNewline = b == '\n'
	}
	return eventData, wasNewline, false
}

// Close closes the SSE stream body.
func (r *SSEStreamReader) Close() error {
	r.lock.Lock()
	if r.closed {
		r.lock.Unlock()
		return nil
	}
	r.closed = true
	body := r.body
	r.lock.Unlock()
	return body.Close()
}

func (r *SSEStreamReader) checkBuffer() ([]byte, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if len(r.buffer) == 0 {
		return nil, false
	}
	for i := 0; i < len(r.buffer)-1; i++ {
		if r.buffer[i] == '\n' && r.buffer[i+1] == '\n' {
			eventEnd := i + 2
			// Copy the event out before compacting: the compaction below
			// rewrites the same backing array and would otherwise corrupt the
			// slice we just returned.
			eventData := append([]byte(nil), r.buffer[:eventEnd]...)
			if eventEnd < len(r.buffer) {
				r.buffer = append(r.buffer[:0], r.buffer[eventEnd:]...)
			} else {
				r.buffer = r.buffer[:0]
			}
			return eventData, true
		}
	}
	// Copy the partial event out so a later r.buffer mutation cannot alias the
	// returned accumulator.
	eventData := append([]byte(nil), r.buffer...)
	r.buffer = r.buffer[:0]
	return eventData, false
}

// ParseSSEEvent parses a single SSE event frame.
func ParseSSEEvent(data []byte) (SSEEvent, error) {
	events, err := ParseSSEStream(bytes.NewReader(data))
	if err != nil {
		return SSEEvent{}, err
	}
	if len(events) == 0 {
		return SSEEvent{}, fmt.Errorf("ParseSSEEvent: no event found")
	}
	if len(events) != 1 {
		return SSEEvent{}, fmt.Errorf("ParseSSEEvent: expected 1 event, got %d", len(events))
	}
	return events[0], nil
}

// ParseSSEStream parses all SSE events from the reader.
func ParseSSEStream(r io.Reader) ([]SSEEvent, error) {
	events := make([]SSEEvent, 0)
	for event, err := range sse.Read(r, nil) {
		if err != nil {
			return nil, err
		}
		events = append(events, SSEEvent{
			ID:   event.LastEventID,
			Type: event.Type,
			Data: event.Data,
		})
	}
	return events, nil
}

// WriteJSONSSEEvent marshals payload as JSON and writes it as an SSE event.
func WriteJSONSSEEvent(w io.Writer, msg SSEMessage, payload any) error {
	byts, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg.Data = string(byts)
	return WriteSSEEvent(w, msg)
}

// EncodeSSEData encodes a payload as an SSE data field.
func EncodeSSEData(payload any) (string, error) {
	switch v := payload.(type) {
	case nil:
		return "null", nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int8:
		return fmt.Sprintf("%d", v), nil
	case int16:
		return fmt.Sprintf("%d", v), nil
	case int32:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case uint:
		return fmt.Sprintf("%d", v), nil
	case uint8:
		return fmt.Sprintf("%d", v), nil
	case uint16:
		return fmt.Sprintf("%d", v), nil
	case uint32:
		return fmt.Sprintf("%d", v), nil
	case uint64:
		return fmt.Sprintf("%d", v), nil
	case float32:
		return fmt.Sprintf("%g", v), nil
	case float64:
		return fmt.Sprintf("%g", v), nil
	default:
		byts, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return string(byts), nil
	}
}

// WriteSSEEvent writes a single SSE event frame.
func WriteSSEEvent(w io.Writer, msg SSEMessage) error {
	event := sse.Message{}
	if msg.ID != "" {
		id, err := sse.NewID(msg.ID)
		if err != nil {
			return err
		}
		event.ID = id
	}
	if msg.Type != "" {
		typ, err := sse.NewType(msg.Type)
		if err != nil {
			return err
		}
		event.Type = typ
	}
	if msg.RetryMillis > 0 {
		event.Retry = time.Duration(msg.RetryMillis) * time.Millisecond
	}
	event.AppendData(msg.Data)
	_, err := event.WriteTo(w)
	return err
}
