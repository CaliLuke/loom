package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
)

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
