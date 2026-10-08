package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	// sseReadOutcome records what a generated SSE client observes for one
	// parsed event returned by SSEStreamReader.ReadEvent, or the read error.
	sseReadOutcome struct {
		Event SSEEvent
		Err   string
	}

	// sseChunkReader returns data in reads whose sizes are drawn cyclically
	// from sizes, so fuzz inputs control where read boundaries land.
	sseChunkReader struct {
		data  []byte
		sizes []byte
		next  int
	}
)

const sseBOM = "\xEF\xBB\xBF"

// FuzzSSEStreamReader checks that the events a generated client observes from
// SSEStreamReader do not depend on how the byte stream is split into reads,
// including incomplete tails. Protocol interpretation belongs to the library;
// these cases exercise the adapter with representative wire inputs.
func FuzzSSEStreamReader(f *testing.F) {
	seeds := []string{
		"event: one\ndata: first\n\nevent: two\ndata: second\n\n",
		"id: 1\nevent: tick\ndata: a\ndata: b\n\n",
		"data: crlf\r\n\r\ndata: next\r\n\r\n",
		"data: cr\r\rdata: next\r\r",
		"data: mixed\r\n\ndata: x\r\r\n",
		sseBOM + "data: bom\n\n",
		sseBOM + "\ndata: bom-blank\n\n",
		": keepalive\n\ndata: after-comment\n\n",
		":\n\n:\n\n",
		"data\n\n",
		"data:\n\n",
		"data:  two-spaces\n\n",
		"event\ndata: typed-without-colon\n\n",
		"retry: 1000\ndata: r\n\n",
		"retry: abc\ndata: bad-retry\n\n",
		"retry: 1000\n\n",
		"id: a\x00b\ndata: nul-id\n\n",
		"id\ndata: empty-id\n\n",
		"unknown: field\ndata: kept\n\n",
		"\n\n\ndata: leading-blank-lines\n\n",
		"data: no terminator",
		"data: one line only\n",
		`data: {"jsonrpc":"2.0","result":{}}` + "\n\n",
		"event: only-type\n\n",
		"id: only-id\n\n",
	}
	for i, seed := range seeds {
		f.Add([]byte(seed), []byte{byte(i), 1, 2, 3, 5, 8})
	}

	f.Fuzz(func(t *testing.T, input, sizes []byte) {
		whole := readSSEOutcomes(t, bytes.NewReader(input))
		chunked := readSSEOutcomes(t, &sseChunkReader{data: input, sizes: sizes})
		require.Equal(t, whole, chunked, "chunking changed the observed events")

		oneByte := readSSEOutcomes(t, &sseChunkReader{data: input, sizes: []byte{0}})
		require.Equal(t, whole, oneByte, "one-byte reads changed the observed events")

		_, err := ParseSSEStream(bytes.NewReader(input))
		if err != nil {
			require.NotEmpty(t, err.Error())
		}
	})
}

// readSSEOutcomes drains an SSEStreamReader the way generated HTTP clients do:
// ReadEvent returns parsed events directly from the library iterator.
func readSSEOutcomes(t *testing.T, body io.Reader) []sseReadOutcome {
	t.Helper()
	reader := NewSSEStreamReader(io.NopCloser(body))
	outcomes := make([]sseReadOutcome, 0)
	for range 10_000 {
		event, err := reader.ReadEvent(context.Background())
		if errors.Is(err, io.EOF) {
			require.Empty(t, event)
			require.NoError(t, reader.Close())
			return outcomes
		}
		if err != nil {
			outcomes = append(outcomes, sseReadOutcome{Err: "read: " + err.Error()})
			require.NoError(t, reader.Close())
			return outcomes
		}
		outcomes = append(outcomes, sseReadOutcome{Event: event})
	}
	t.Fatalf("SSEStreamReader did not reach EOF")
	return nil
}

func (r *sseChunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	size := len(p)
	if len(r.sizes) > 0 {
		size = int(r.sizes[r.next%len(r.sizes)])%17 + 1
		r.next++
	}
	size = min(size, len(p), len(r.data))
	n := copy(p, r.data[:size])
	r.data = r.data[n:]
	return n, nil
}
