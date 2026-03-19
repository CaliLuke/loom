package http

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteSSEEvent(t *testing.T) {
	t.Run("multiline data is split into repeated data fields", func(t *testing.T) {
		var buf bytes.Buffer
		err := WriteSSEEvent(&buf, SSEMessage{
			ID:          "evt-1",
			Type:        "message",
			Data:        "line-1\nline-2",
			RetryMillis: 2500,
		})
		require.NoError(t, err)
		require.Equal(t, strings.Join([]string{
			"id: evt-1",
			"event: message",
			"retry: 2500",
			"data: line-1",
			"data: line-2",
			"",
			"",
		}, "\n"), buf.String())
	})

	t.Run("rejects multiline event metadata", func(t *testing.T) {
		err := WriteSSEEvent(&bytes.Buffer{}, SSEMessage{
			Type: "bad\nevent",
			Data: "payload",
		})
		require.Error(t, err)
	})
}

func TestWriteJSONSSEEvent(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSONSSEEvent(&buf, SSEMessage{Type: "message"}, map[string]any{
		"jsonrpc": "2.0",
		"result":  map[string]any{"value": "ok"},
	})
	require.NoError(t, err)
	require.Equal(t, strings.Join([]string{
		"event: message",
		`data: {"jsonrpc":"2.0","result":{"value":"ok"}}`,
		"",
		"",
	}, "\n"), buf.String())
}

func TestParseSSEEvent(t *testing.T) {
	t.Run("parses a single event frame", func(t *testing.T) {
		event, err := ParseSSEEvent([]byte(strings.Join([]string{
			"id: evt-1",
			"event: message",
			"data: line-1",
			"data: line-2",
			"",
		}, "\n")))
		require.NoError(t, err)
		require.Equal(t, SSEEvent{
			ID:   "evt-1",
			Type: "message",
			Data: "line-1\nline-2",
		}, event)
	})

	t.Run("rejects multi-event chunks", func(t *testing.T) {
		_, err := ParseSSEEvent([]byte(strings.Join([]string{
			"event: one",
			"data: first",
			"",
			"event: two",
			"data: second",
			"",
		}, "\n")))
		require.Error(t, err)
	})
}

func TestParseSSEStream(t *testing.T) {
	events, err := ParseSSEStream(strings.NewReader(strings.Join([]string{
		"event: message",
		`data: {"step":1}`,
		"",
		"event: response",
		`data: {"step":2}`,
		"",
	}, "\n")))
	require.NoError(t, err)
	require.Equal(t, []SSEEvent{
		{Type: "message", Data: `{"step":1}`},
		{Type: "response", Data: `{"step":2}`},
	}, events)
}
