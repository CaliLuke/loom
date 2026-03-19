package harness

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSSEEventsPreservesEventTypes(t *testing.T) {
	client := &Client{}
	raw := strings.NewReader(strings.Join([]string{
		"event: message",
		`data: {"jsonrpc":"2.0","method":"stream_string_sse","params":{"value":"step-1"}}`,
		"",
		"event: response",
		`data: {"jsonrpc":"2.0","id":"done","result":{"value":"complete"}}`,
		"",
		"event: message",
		`data: {"jsonrpc":"2.0","id":"oops","error":{"code":-32601,"message":"Method not found"}}`,
		"",
	}, "\n"))

	events, err := client.parseSSEEvents(raw)
	require.NoError(t, err)
	require.Len(t, events, 3)

	require.Equal(t, "message", events[0].Type)
	require.JSONEq(t, `{"jsonrpc":"2.0","method":"stream_string_sse","params":{"value":"step-1"}}`, string(events[0].Data))

	require.Equal(t, "response", events[1].Type)
	require.JSONEq(t, `{"jsonrpc":"2.0","id":"done","result":{"value":"complete"}}`, string(events[1].Data))

	require.Equal(t, "message", events[2].Type)
	require.JSONEq(t, `{"jsonrpc":"2.0","id":"oops","error":{"code":-32601,"message":"Method not found"}}`, string(events[2].Data))
}
