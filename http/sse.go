package http

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"iter"
	"sync"
	"time"

	sse "github.com/CaliLuke/go-sse"
)

type (
	// SSEEvent represents a parsed server-sent event.
	SSEEvent struct {
		// ID is the last event ID received on the stream, including an empty reset.
		ID string
		// Type is the event type, or empty when no type was specified.
		Type string
		// Data is the event payload, with multiple data lines joined by newlines.
		Data string
	}

	// SSEMessage describes an SSE frame to write to an output stream.
	SSEMessage struct {
		// ID is the event identifier to write.
		ID string
		// Type is the event type to write.
		Type string
		// Data is the event payload; newlines produce separate data fields.
		Data string
		// RetryMillis is the reconnection delay in milliseconds.
		RetryMillis int64
	}

	// SSEStreamReader adapts the SSE library iterator to context-aware reads.
	// The library owns framing, parsing, event limits, and event memory. Reads
	// are serialized. Closing the response body must unblock an active Read,
	// as required by net/http.Response.Body.
	SSEStreamReader struct {
		body      io.ReadCloser
		next      func() (sse.Event, error, bool)
		stop      func()
		readLock  sync.Mutex
		readErr   error
		closeOnce sync.Once
		closed    chan struct{}
		closeErr  error
	}
)

// NewSSEStreamReader returns a stream reader with the library's default 64 KiB
// event limit. The caller must call Close to release the body and iterator.
func NewSSEStreamReader(body io.ReadCloser) *SSEStreamReader {
	next, stop := iter.Pull2(sse.Read(body, nil))
	return &SSEStreamReader{body: body, next: next, stop: stop, closed: make(chan struct{})}
}

// ReadEvent returns the next parsed SSE event. EOF discards an incomplete final
// event. Canceling ctx closes the stream and returns ctx.Err unless a complete
// event was already obtained. Subsequent reads of a closed stream return EOF.
func (r *SSEStreamReader) ReadEvent(ctx context.Context) (SSEEvent, error) {
	r.readLock.Lock()
	defer r.readLock.Unlock()
	if err := ctx.Err(); err != nil {
		return SSEEvent{}, err
	}
	select {
	case <-r.closed:
		return SSEEvent{}, io.EOF
	default:
	}
	if r.readErr != nil {
		return SSEEvent{}, r.readErr
	}
	stopCancellation := context.AfterFunc(ctx, r.close)
	event, err, ok := r.next()
	stopCancellation()
	if err == nil && ok {
		return SSEEvent{ID: event.LastEventID, Type: event.Type, Data: event.Data}, nil
	}
	r.stop()
	if ctx.Err() != nil {
		err = ctx.Err()
	} else if err == nil {
		err = io.EOF
	}
	r.readErr = err
	return SSEEvent{}, err
}

// Close closes the response body once and releases the library iterator. It
// waits for any active read to finish and returns the body's close error.
func (r *SSEStreamReader) Close() error {
	r.close()
	return r.closeErr
}

// close also serves as the cancellation callback. Record the body error for
// Close while ReadEvent preserves context cancellation as its primary cause.
func (r *SSEStreamReader) close() {
	r.closeOnce.Do(func() {
		close(r.closed)
		r.closeErr = r.body.Close()
	})
	r.readLock.Lock()
	defer r.readLock.Unlock()
	r.stop()
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

// EncodeSSEJSONData encodes payload as the JSON data of a whole-event SSE
// frame. Unlike EncodeSSEData, strings become JSON strings and byte slices
// become base64 JSON strings, so every value, including empty strings,
// carriage returns, trailing newlines, and binary bytes, survives SSE framing
// and decodes to the value that was sent.
func EncodeSSEJSONData(payload any) (string, error) {
	byts, err := json.Marshal(payload, json.Deterministic(true))
	if err != nil {
		return "", err
	}
	return string(byts), nil
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
