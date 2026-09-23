package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	// sseReadOutcome records what a generated SSE client observes for one
	// frame returned by SSEStreamReader.ReadEvent: either the parsed event or
	// the parse error text.
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
// and that complete streams produce exactly the events of an independent
// WHATWG event-stream reference parser.
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

		// Terminate the stream so every block is complete, then compare with
		// the reference parser.
		complete := append(append([]byte(nil), input...), "\n\n"...)
		got := readSSEOutcomes(t, &sseChunkReader{data: complete, sizes: sizes})
		events := referenceSSEFrameEvents(complete)
		want := make([]sseReadOutcome, 0, len(events))
		for _, event := range events {
			want = append(want, sseReadOutcome{Event: event})
		}
		require.Equal(t, want, got, "reader events differ from the reference parser")

		_, err := ParseSSEStream(bytes.NewReader(input))
		if err != nil {
			require.NotEmpty(t, err.Error())
		}
	})
}

// readSSEOutcomes drains an SSEStreamReader the way generated HTTP clients do:
// every frame returned by ReadEvent is handed to ParseSSEEvent.
func readSSEOutcomes(t *testing.T, body io.Reader) []sseReadOutcome {
	t.Helper()
	reader := NewSSEStreamReader(io.NopCloser(body))
	outcomes := make([]sseReadOutcome, 0)
	for range 10_000 {
		frame, err := reader.ReadEvent(context.Background())
		if errors.Is(err, io.EOF) {
			require.Empty(t, frame)
			require.NoError(t, reader.Close())
			return outcomes
		}
		if err != nil {
			outcomes = append(outcomes, sseReadOutcome{Err: "read: " + err.Error()})
			require.NoError(t, reader.Close())
			return outcomes
		}
		event, err := ParseSSEEvent(frame)
		if err != nil {
			outcomes = append(outcomes, sseReadOutcome{Err: err.Error()})
			continue
		}
		outcomes = append(outcomes, sseReadOutcome{Event: event})
	}
	t.Fatalf("SSEStreamReader did not reach EOF")
	return nil
}

// referenceSSEFrameEvents interprets an event stream following the WHATWG
// "Interpreting an event stream" algorithm, resolved to the per-frame contract
// Loom exposes through ReadEvent and ParseSSEEvent:
//
//   - Lines end at CRLF, LF, or CR. A trailing unterminated line and a trailing
//     incomplete block are discarded.
//   - SSEEvent.ID is the id field of the frame itself (ParseSSEEvent sees one
//     frame), not the stream-wide last event ID buffer.
//   - A block dispatches when it has a data, event, or valid id field. The
//     spec dispatches only when the data buffer is non-empty; Loom inherits the
//     broader rule from go-sse.
//   - A UTF-8 BOM is ignored at the start of each block, which is what
//     go-sse does when it parses each frame. The spec ignores it only at the
//     start of the stream.
func referenceSSEFrameEvents(input []byte) []SSEEvent {
	var (
		events     []SSEEvent
		data       strings.Builder
		id, typ    string
		dirty      bool
		blockStart = true
	)
	for line := range referenceSSELines(input) {
		if blockStart {
			line = strings.TrimPrefix(line, sseBOM)
		}
		if line == "" {
			if dirty {
				payload := data.String()
				events = append(events, SSEEvent{ID: id, Type: typ, Data: strings.TrimSuffix(payload, "\n")})
			}
			data.Reset()
			id, typ, dirty, blockStart = "", "", false, true
			continue
		}
		blockStart = false
		if line[0] == ':' {
			continue
		}
		name, value, found := strings.Cut(line, ":")
		if found {
			value = strings.TrimPrefix(value, " ")
		}
		switch name {
		case "data":
			data.WriteString(value)
			data.WriteByte('\n')
			dirty = true
		case "event":
			typ = value
			dirty = true
		case "id":
			if !strings.ContainsRune(value, 0) {
				id = value
				dirty = true
			}
		}
	}
	return events
}

// referenceSSELines yields every terminated line of input, splitting on CRLF,
// LF, and CR.
func referenceSSELines(input []byte) func(func(string) bool) {
	return func(yield func(string) bool) {
		start := 0
		for i := 0; i < len(input); i++ {
			switch input[i] {
			case '\n':
			case '\r':
				if i+1 < len(input) && input[i+1] == '\n' {
					if !yield(string(input[start:i])) {
						return
					}
					i++
					start = i + 1
					continue
				}
			default:
				continue
			}
			if !yield(string(input[start:i])) {
				return
			}
			start = i + 1
		}
	}
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
