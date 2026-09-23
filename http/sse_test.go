package http

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"runtime"
	"strconv"
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
	lines := strings.Split(buf.String(), "\n")
	require.Len(t, lines, 4)
	require.Equal(t, "event: message", lines[0])
	require.Empty(t, lines[2])
	require.Empty(t, lines[3])
	require.True(t, strings.HasPrefix(lines[1], "data: "))
	require.JSONEq(t, `{"jsonrpc":"2.0","result":{"value":"ok"}}`, strings.TrimPrefix(lines[1], "data: "))
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

func TestSSEStreamReaderSkipsBlocksWithoutEvents(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []SSEEvent
	}{
		{
			name:  "keepalive comment before event",
			input: ": keepalive\n\ndata: first\n\n",
			want:  []SSEEvent{{Data: "first"}},
		},
		{
			name:  "blank-line runs between events",
			input: "data: first\n\n\n\n\ndata: second\n\n\n",
			want:  []SSEEvent{{Data: "first"}, {Data: "second"}},
		},
		{
			name:  "unknown field block",
			input: "0\n\ndata: first\n\n",
			want:  []SSEEvent{{Data: "first"}},
		},
		{
			name:  "retry-only block",
			input: "retry: 1000\n\nevent: tick\ndata: first\n\n",
			want:  []SSEEvent{{Type: "tick", Data: "first"}},
		},
		{
			name:  "only comments",
			input: ":\n\n: ping\n\n",
			want:  []SSEEvent{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := NewSSEStreamReader(io.NopCloser(strings.NewReader(tc.input)))
			got := make([]SSEEvent, 0, len(tc.want))
			for {
				frame, err := reader.ReadEvent(context.Background())
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				event, err := ParseSSEEvent(frame)
				require.NoError(t, err, "frame %q", frame)
				got = append(got, event)
			}
			require.Equal(t, tc.want, got)
		})
	}
}

func TestSSEStreamReaderLineTerminators(t *testing.T) {
	cases := []struct {
		name   string
		chunks []string
		want   []SSEEvent
	}{
		{
			name:   "CRLF",
			chunks: []string{"event: one\r\ndata: a\r\n\r\nevent: two\r\ndata: b\r\n\r\n"},
			want:   []SSEEvent{{Type: "one", Data: "a"}, {Type: "two", Data: "b"}},
		},
		{
			name:   "CR",
			chunks: []string{"data: a\r\rdata: b\r\r"},
			want:   []SSEEvent{{Data: "a"}, {Data: "b"}},
		},
		{
			name:   "mixed terminators",
			chunks: []string{"data: a\r\ndata: b\n\rdata: c\r\r\ndata: d\n\n"},
			want:   []SSEEvent{{Data: "a\nb"}, {Data: "c"}, {Data: "d"}},
		},
		{
			name:   "CRLF split across reads inside a block",
			chunks: []string{"data: a\r", "\ndata: b\r\n\r", "\n"},
			want:   []SSEEvent{{Data: "a\nb"}},
		},
		{
			name:   "CR blank line followed by LF in the next read",
			chunks: []string{"data: a\r\r", "\ndata: b\r\r"},
			want:   []SSEEvent{{Data: "a"}, {Data: "b"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := NewSSEStreamReader(io.NopCloser(&chunkedReader{chunks: tc.chunks}))
			got := make([]SSEEvent, 0, len(tc.want))
			for {
				frame, err := reader.ReadEvent(context.Background())
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				event, err := ParseSSEEvent(frame)
				require.NoError(t, err, "frame %q", frame)
				got = append(got, event)
			}
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("CRLF event is delivered before the stream ends", func(t *testing.T) {
		pr, pw := io.Pipe()
		t.Cleanup(func() {
			require.NoError(t, pw.Close())
		})
		go func() {
			_, err := pw.Write([]byte("data: live\r\n\r\n"))
			if err != nil {
				t.Errorf("write: %v", err)
			}
		}()
		reader := NewSSEStreamReader(pr)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		frame, err := reader.ReadEvent(ctx)
		require.NoError(t, err)
		event, err := ParseSSEEvent(frame)
		require.NoError(t, err)
		require.Equal(t, SSEEvent{Data: "live"}, event)
	})
}

func TestSSEStreamReaderBoundsEventSize(t *testing.T) {
	cases := []struct {
		name string
		unit string
	}{
		{name: "unterminated line", unit: "x"},
		{name: "block without blank line", unit: "data: x\n"},
		{name: "block without blank line CRLF", unit: "data: x\r\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := &repeatingReader{unit: []byte(tc.unit), limit: 8 << 20}
			reader := NewSSEStreamReader(io.NopCloser(body))
			_, err := reader.ReadEvent(context.Background())
			require.ErrorIs(t, err, bufio.ErrTooLong)
			require.Less(t, body.read, int64(bufio.MaxScanTokenSize+16<<10), "reader buffered past the event limit")

			_, err = reader.ReadEvent(context.Background())
			require.ErrorIs(t, err, bufio.ErrTooLong, "oversized event error must be sticky")
			require.Less(t, body.read, int64(bufio.MaxScanTokenSize+16<<10), "reader kept buffering after the limit")
		})
	}

	t.Run("event at the limit is accepted", func(t *testing.T) {
		prefix, suffix := "data: ", "\n\n"
		payload := strings.Repeat("x", bufio.MaxScanTokenSize-len(prefix)-len(suffix))
		input := prefix + payload + suffix + "data: next\n\n"
		reader := NewSSEStreamReader(io.NopCloser(strings.NewReader(input)))
		frame, err := reader.ReadEvent(context.Background())
		require.NoError(t, err)
		event, err := ParseSSEEvent(frame)
		require.NoError(t, err)
		require.Equal(t, payload, event.Data)
		frame, err = reader.ReadEvent(context.Background())
		require.NoError(t, err)
		require.Equal(t, "data: next\n\n", string(frame))
	})
}

func TestSSEFrameDispatchesMatchesParser(t *testing.T) {
	cases := []string{
		"",
		"\n",
		"\r\n\r\n",
		": keepalive\n\n",
		":\r\r",
		"data: x\n\n",
		"data\n\n",
		"data:\n\n",
		"data:\r\r",
		"event\n\n",
		"event: tick\r\n\r\n",
		"id\n\n",
		"id: 1\n\n",
		"id: a\x00b\n\n",
		"id\x00\n\n",
		"retry: 1000\n\n",
		"retry: abc\n\n",
		"retry\n\n",
		"unknown: field\n\n",
		"0\n\n",
		"Data: x\n\n",
		"data : x\n\n",
		" data: x\n\n",
		"dataa: x\n\n",
		"datax\n\n",
		"\xEF\xBB\xBFdata: x\n\n",
		"\xEF\xBB\xBF: comment\n\n",
		"\xEF\xBB\xBF\n\n",
		"\n\xEF\xBB\xBFdata: x\n\n",
		": c\n\xEF\xBB\xBFdata: x\n\n",
		"\xEF\xBB\xBF\xEF\xBB\xBFdata: x\n\n",
		"retry: 1\r: c\rid: 2\r\r",
		"retry: 1\r\nunknown\r\n\r\n",
		"data: no terminator",
		": comment no terminator",
		"retry: 1\n: trailing",
		"\n\n: ping\n\ndata: second\n\n",
		": a\n\n: b\n\n",
	}
	for _, input := range cases {
		t.Run(strconvQuote(input), func(t *testing.T) {
			events, err := ParseSSEStream(strings.NewReader(input))
			want := err != nil || len(events) > 0
			require.Equal(t, want, sseFrameDispatches([]byte(input)), "events %v, err %v", events, err)
		})
	}
}

func TestSSEStreamReaderKeepalivesDoNotAllocatePerBlock(t *testing.T) {
	const (
		keepalives = 200
		runs       = 20
	)
	group := strings.Repeat(": keepalive\n\n", keepalives) + "data: x\n\n"
	reader := NewSSEStreamReader(io.NopCloser(strings.NewReader(strings.Repeat(group, runs+1))))
	allocs := testing.AllocsPerRun(runs, func() {
		frame, err := reader.ReadEvent(context.Background())
		if err != nil || string(frame) != "data: x\n\n" {
			t.Errorf("ReadEvent = %q, %v", frame, err)
		}
	})
	// Each ReadEvent skips 200 keepalive blocks; only the returned frame and
	// the per-read goroutine bookkeeping may allocate.
	require.Less(t, allocs, float64(keepalives)/10, "allocations per ReadEvent")
}

func TestSSEStreamReaderKeepalivesDoNotAllocatePerByte(t *testing.T) {
	const (
		keepalives = 200
		warmup     = 100
		reads      = 2000
	)
	group := strings.Repeat(": keepalive\n\n", keepalives) + "data: x\n\n"
	body := &repeatingReader{unit: []byte(group), limit: int64(len(group)) * (warmup + reads + 10)}
	reader := NewSSEStreamReader(io.NopCloser(body))
	readEvent := func() {
		frame, err := reader.ReadEvent(context.Background())
		if err != nil || string(frame) != "data: x\n\n" {
			t.Fatalf("ReadEvent = %q, %v", frame, err)
		}
	}
	for range warmup {
		readEvent()
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range reads {
		readEvent()
	}
	runtime.ReadMemStats(&after)
	perRead := float64(after.TotalAlloc-before.TotalAlloc) / reads
	// Each ReadEvent consumes len(group) bytes of input. The scanner buffer
	// must be reused rather than reallocated as the window slides forward, so
	// allocation per event stays far below the bytes read.
	require.Less(t, perRead, float64(len(group))/8, "bytes allocated per ReadEvent (input %d bytes)", len(group))
}

// strconvQuote names subtests after inputs with control bytes.
func strconvQuote(s string) string {
	return strings.ReplaceAll(strconv.Quote(s), "/", "_")
}

// repeatingReader yields unit forever, failing once limit bytes were read so a
// reader that never stops buffering fails the test instead of exhausting
// memory.
type repeatingReader struct {
	unit  []byte
	read  int64
	limit int64
}

func (r *repeatingReader) Read(p []byte) (int, error) {
	if r.read >= r.limit {
		return 0, errors.New("repeatingReader: read limit exceeded")
	}
	n := 0
	for n < len(p) {
		n += copy(p[n:], r.unit[int(r.read+int64(n))%len(r.unit):])
	}
	r.read += int64(n)
	return n, nil
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
