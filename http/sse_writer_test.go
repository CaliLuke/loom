package http

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

func TestSSEStreamWriterControls(t *testing.T) {
	t.Run("open is idempotent and comments share the stream", func(t *testing.T) {
		w := &deadlineResponseWriter{header: make(http.Header)}
		stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})

		require.NoError(t, stream.Open(context.Background()))
		require.NoError(t, stream.Open(context.Background()))
		require.NoError(t, stream.SendComment(context.Background(), "heartbeat"))
		require.Equal(t, http.StatusOK, w.status)
		require.Equal(t, "text/event-stream", w.header.Get("Content-Type"))
		require.Equal(t, 2, w.flushes)
		require.Equal(t, ": heartbeat\n\n", w.body.String())
		require.True(t, stream.Started())
	})

	t.Run("comment injection is rejected", func(t *testing.T) {
		w := &deadlineResponseWriter{header: make(http.Header)}
		stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})
		err := stream.SendComment(context.Background(), "ok\ndata: injected")
		require.ErrorIs(t, err, ErrInvalidSSEComment)
		require.False(t, stream.Started())
	})

	t.Run("open preserves its first flush error", func(t *testing.T) {
		flushErr := errors.New("flush failed")
		w := &failingFlushWriter{header: make(http.Header), err: flushErr}
		stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})
		require.ErrorIs(t, stream.Open(context.Background()), flushErr)
		require.ErrorIs(t, stream.Open(context.Background()), flushErr)
		require.Equal(t, 1, w.flushes)
	})

	t.Run("cancellation and closure reject writes", func(t *testing.T) {
		w := &deadlineResponseWriter{header: make(http.Header)}
		requestCtx, cancel := context.WithCancel(context.Background())
		stream := NewSSEStreamWriter(w, requestCtx, loomtransport.TransportHTTP, StreamWritePolicy{})
		cancel()
		require.ErrorIs(t, stream.Open(context.Background()), context.Canceled)

		stream = NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})
		require.NoError(t, stream.Close())
		require.ErrorIs(t, stream.SendComment(context.Background(), "late"), ErrSSEStreamClosed)
	})
}

func TestStreamWritePolicy(t *testing.T) {
	_, err := NewStreamWritePolicy(-time.Nanosecond)
	require.ErrorIs(t, err, ErrInvalidStreamWriteTimeout)

	policy, err := NewStreamWritePolicy(time.Second)
	require.NoError(t, err)
	require.Equal(t, time.Second, policy.Timeout())
}

func TestSSEStreamWriterAppliesFreshDeadlines(t *testing.T) {
	policy, err := NewStreamWritePolicy(time.Second)
	require.NoError(t, err)
	w := &deadlineResponseWriter{header: make(http.Header)}
	stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, policy)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	require.NoError(t, stream.SendComment(ctx, "heartbeat"))
	deadlines := w.writeDeadlines()
	require.Len(t, deadlines, 4)
	require.False(t, deadlines[0].IsZero())
	require.True(t, deadlines[1].IsZero())
	require.False(t, deadlines[2].IsZero())
	require.True(t, deadlines[3].IsZero())
	require.WithinDuration(t, time.Now().Add(100*time.Millisecond), deadlines[0], 50*time.Millisecond)
}

func TestSSEStreamWriterRejectsUnsupportedDeadline(t *testing.T) {
	policy, err := NewStreamWritePolicy(time.Second)
	require.NoError(t, err)
	stream := NewSSEStreamWriter(
		&unsupportedDeadlineWriter{header: make(http.Header)},
		context.Background(),
		loomtransport.TransportHTTP,
		policy,
	)
	require.ErrorIs(t, stream.SendComment(context.Background(), "heartbeat"), ErrStreamWriteDeadlineUnsupported)
}

func TestSSEStreamWriterBoundsBlockedWrite(t *testing.T) {
	policy, err := NewStreamWritePolicy(25 * time.Millisecond)
	require.NoError(t, err)
	w := newBlockingDeadlineWriter()
	stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, policy)
	start := time.Now()

	err = stream.SendComment(context.Background(), "heartbeat")
	require.Error(t, err)
	require.Less(t, time.Since(start), 500*time.Millisecond)
	var timeout interface{ Timeout() bool }
	require.ErrorAs(t, err, &timeout)
	require.True(t, timeout.Timeout())
}

type deadlineResponseWriter struct {
	lock      sync.Mutex
	header    http.Header
	body      bytes.Buffer
	status    int
	flushes   int
	deadlines []time.Time
}

type failingFlushWriter struct {
	header  http.Header
	err     error
	flushes int
}

func (w *failingFlushWriter) Header() http.Header {
	return w.header
}

func (w *failingFlushWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *failingFlushWriter) WriteHeader(int) {}

func (w *failingFlushWriter) FlushError() error {
	w.flushes++
	return w.err
}

func (w *deadlineResponseWriter) Header() http.Header {
	return w.header
}

func (w *deadlineResponseWriter) Write(p []byte) (int, error) {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.body.Write(p)
}

func (w *deadlineResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *deadlineResponseWriter) Flush() {
	w.flushes++
}

func (w *deadlineResponseWriter) SetWriteDeadline(deadline time.Time) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func (w *deadlineResponseWriter) writeDeadlines() []time.Time {
	w.lock.Lock()
	defer w.lock.Unlock()
	return append([]time.Time(nil), w.deadlines...)
}

type unsupportedDeadlineWriter struct {
	header http.Header
}

func (w *unsupportedDeadlineWriter) Header() http.Header {
	return w.header
}

func (w *unsupportedDeadlineWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *unsupportedDeadlineWriter) WriteHeader(int) {}

func (w *unsupportedDeadlineWriter) Flush() {}

func (w *unsupportedDeadlineWriter) SetWriteDeadline(time.Time) error {
	return errors.ErrUnsupported
}

type blockingDeadlineWriter struct {
	header   http.Header
	deadline chan time.Time
}

func newBlockingDeadlineWriter() *blockingDeadlineWriter {
	return &blockingDeadlineWriter{
		header:   make(http.Header),
		deadline: make(chan time.Time, 1),
	}
}

func (w *blockingDeadlineWriter) Header() http.Header {
	return w.header
}

func (w *blockingDeadlineWriter) Write([]byte) (int, error) {
	deadline := <-w.deadline
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	<-timer.C
	return 0, deadlineTimeoutError{}
}

func (w *blockingDeadlineWriter) WriteHeader(int) {}

func (w *blockingDeadlineWriter) Flush() {}

func (w *blockingDeadlineWriter) SetWriteDeadline(deadline time.Time) error {
	if !deadline.IsZero() {
		w.deadline <- deadline
	}
	return nil
}

type deadlineTimeoutError struct{}

func (deadlineTimeoutError) Error() string {
	return "write timed out"
}

func (deadlineTimeoutError) Timeout() bool {
	return true
}
