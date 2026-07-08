package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestEncodeSSEData(t *testing.T) {
	cases := []struct {
		name    string
		payload any
		want    string
	}{
		{"nil", nil, "null"},
		{"string", "hello", "hello"},
		{"bytes", []byte("hello"), "hello"},
		{"bool", true, "true"},
		{"int", int8(3), "3"},
		{"float", 1.5, "1.5"},
		{"object", map[string]string{"hello": "world"}, `{"hello":"world"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EncodeSSEData(tc.payload)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSSEStreamReader(t *testing.T) {
	reader := NewSSEStreamReader(io.NopCloser(strings.NewReader(strings.Join([]string{
		"event: one",
		"data: first",
		"",
		"event: two",
		"data: second",
		"",
		"",
	}, "\n"))))

	first, err := reader.ReadEvent(context.Background())
	require.NoError(t, err)
	require.Equal(t, "event: one\ndata: first\n\n", string(first))

	second, err := reader.ReadEvent(context.Background())
	require.NoError(t, err)
	require.Equal(t, "event: two\ndata: second\n\n", string(second))

	_, err = reader.ReadEvent(context.Background())
	require.ErrorIs(t, err, io.EOF)
	require.NoError(t, reader.Close())
	require.NoError(t, reader.Close())
}

func TestSSEStreamReaderNoBufferAliasing(t *testing.T) {
	const (
		eventOne   = "event: one\ndata: first\n\n"
		eventTwo   = "event: two\ndata: second\n\n"
		partialTwo = "event: two\ndata: sec"
		restTwo    = "ond\n\n"
		partial    = "event: three\ndata: thi"
	)

	cases := []struct {
		name   string
		chunks []string
		want   []string
	}{
		{
			name:   "two complete events in one read chunk",
			chunks: []string{eventOne + eventTwo},
			want:   []string{eventOne, eventTwo},
		},
		{
			name:   "two complete events plus partial third in one chunk",
			chunks: []string{eventOne + eventTwo + partial},
			want:   []string{eventOne, eventTwo},
		},
		{
			name:   "event split across multiple small reads",
			chunks: []string{eventOne, partialTwo, restTwo},
			want:   []string{eventOne, eventTwo},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := NewSSEStreamReader(io.NopCloser(&chunkedReader{chunks: tc.chunks}))
			got := make([]string, 0, len(tc.want))
			for range tc.want {
				event, err := reader.ReadEvent(context.Background())
				require.NoError(t, err)
				got = append(got, string(event))
			}
			require.Equal(t, tc.want, got)
			require.NoError(t, reader.Close())
		})
	}
}

func TestSSEStreamReaderContextErrorWinsOverCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	body := newBlockingSSEBody(closeErr)
	reader := NewSSEStreamReader(body)
	ctx, cancel := context.WithCancel(context.Background())

	readc := make(chan error, 1)
	go func() {
		_, err := reader.ReadEvent(ctx)
		readc <- err
	}()

	select {
	case <-body.readStarted:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for reader to start reading")
	}
	cancel()

	select {
	case err := <-readc:
		require.ErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, closeErr)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for canceled read")
	}
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

// chunkedReader returns each configured chunk on a separate Read call so tests
// can control how event boundaries land across reads.
type chunkedReader struct {
	chunks []string
}

func (c *chunkedReader) Read(p []byte) (int, error) {
	if len(c.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := c.chunks[0]
	if len(p) < len(chunk) {
		n := copy(p, chunk)
		c.chunks[0] = chunk[n:]
		return n, nil
	}
	n := copy(p, chunk)
	c.chunks = c.chunks[1:]
	return n, nil
}

type blockingSSEBody struct {
	readStarted chan struct{}
	closed      chan struct{}
	closeOnce   sync.Once
	closeErr    error
}

func newBlockingSSEBody(closeErr error) *blockingSSEBody {
	return &blockingSSEBody{
		readStarted: make(chan struct{}),
		closed:      make(chan struct{}),
		closeErr:    closeErr,
	}
}

func (b *blockingSSEBody) Read([]byte) (int, error) {
	close(b.readStarted)
	<-b.closed
	return 0, io.EOF
}

func (b *blockingSSEBody) Close() error {
	b.closeOnce.Do(func() {
		close(b.closed)
	})
	return b.closeErr
}
