package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
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
		require.Equal(t, 1, w.writeHeaderCalls)
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

func TestSSEStreamWriterCommitsHeaderOnce(t *testing.T) {
	tests := []struct {
		name        string
		write       func(*SSEStreamWriter) error
		wantBody    string
		wantFlushes int
	}{
		{
			name: "open followed by events and comments",
			write: func(stream *SSEStreamWriter) error {
				if err := stream.Open(context.Background()); err != nil {
					return err
				}
				if err := stream.Open(context.Background()); err != nil {
					return err
				}
				if err := writeTestSSEEvent(stream); err != nil {
					return err
				}
				if err := writeTestSSEEvent(stream); err != nil {
					return err
				}
				if err := stream.SendComment(context.Background(), "heartbeat"); err != nil {
					return err
				}
				return stream.SendComment(context.Background(), "heartbeat")
			},
			wantBody:    "data: event\n\ndata: event\n\n: heartbeat\n\n: heartbeat\n\n",
			wantFlushes: 5,
		},
		{
			name: "first event without open",
			write: func(stream *SSEStreamWriter) error {
				if err := writeTestSSEEvent(stream); err != nil {
					return err
				}
				if err := writeTestSSEEvent(stream); err != nil {
					return err
				}
				return stream.SendComment(context.Background(), "heartbeat")
			},
			wantBody:    "data: event\n\ndata: event\n\n: heartbeat\n\n",
			wantFlushes: 3,
		},
		{
			name: "first comment without open",
			write: func(stream *SSEStreamWriter) error {
				if err := stream.SendComment(context.Background(), "heartbeat"); err != nil {
					return err
				}
				if err := stream.SendComment(context.Background(), "heartbeat"); err != nil {
					return err
				}
				return writeTestSSEEvent(stream)
			},
			wantBody:    ": heartbeat\n\n: heartbeat\n\ndata: event\n\n",
			wantFlushes: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := &deadlineResponseWriter{header: make(http.Header)}
			stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})

			require.NoError(t, test.write(stream))
			require.Equal(t, http.StatusOK, w.status)
			require.Equal(t, 1, w.writeHeaderCalls)
			require.Equal(t, test.wantFlushes, w.flushes)
			require.Equal(t, test.wantBody, w.body.String())
		})
	}
}

func TestSSEStreamWriterDoesNotRecommitAfterFailure(t *testing.T) {
	t.Run("write failure", func(t *testing.T) {
		writeErr := errors.New("write failed")
		w := &deadlineResponseWriter{header: make(http.Header)}
		stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})

		err := stream.WriteEvent(context.Background(), func(io.Writer) error {
			return writeErr
		})
		require.ErrorIs(t, err, writeErr)
		require.NoError(t, stream.SendComment(context.Background(), "heartbeat"))
		require.Equal(t, 1, w.writeHeaderCalls)
		require.Equal(t, 1, w.flushes)
	})

	t.Run("flush failure", func(t *testing.T) {
		flushErr := errors.New("flush failed")
		w := &failingFlushWriter{header: make(http.Header), err: flushErr}
		stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})

		require.ErrorIs(t, stream.Open(context.Background()), flushErr)
		require.ErrorIs(t, stream.Open(context.Background()), flushErr)
		require.ErrorIs(t, stream.SendComment(context.Background(), "heartbeat"), flushErr)
		require.Equal(t, 1, w.writeHeaderCalls)
		require.Equal(t, 2, w.flushes)
	})
}

func TestSSEStreamWriterConcurrentWritesCommitHeaderOnce(t *testing.T) {
	const writers = 16
	w := &deadlineResponseWriter{header: make(http.Header)}
	stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})
	errs := make(chan error, writers)
	var wait sync.WaitGroup

	for i := range writers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if i%2 == 0 {
				errs <- writeTestSSEEvent(stream)
				return
			}
			errs <- stream.SendComment(context.Background(), "heartbeat")
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, 1, w.writeHeaderCalls)
	require.Equal(t, writers, w.flushes)
	require.Equal(t, writers/2, strings.Count(w.body.String(), "data: event\n\n"))
	require.Equal(t, writers/2, strings.Count(w.body.String(), ": heartbeat\n\n"))
}

func TestSSEStreamWriterDoesNotLogSuperfluousWriteHeader(t *testing.T) {
	var serverLog bytes.Buffer
	handlerErr := make(chan error, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := NewSSEStreamWriter(w, r.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
		firstErr := writeTestSSEEvent(stream)
		secondErr := writeTestSSEEvent(stream)
		handlerErr <- errors.Join(firstErr, secondErr)
	}))
	server.Config.ErrorLog = log.New(&serverLog, "", 0)
	server.Start()
	defer server.Close()

	response, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	require.NoError(t, readErr)
	require.NoError(t, closeErr)
	require.NoError(t, <-handlerErr)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "text/event-stream", response.Header.Get("Content-Type"))
	require.Equal(t, "data: event\n\ndata: event\n\n", string(body))
	require.NotContains(t, serverLog.String(), "superfluous response.WriteHeader")
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
	lock             sync.Mutex
	header           http.Header
	body             bytes.Buffer
	status           int
	writeHeaderCalls int
	flushes          int
	deadlines        []time.Time
}

type failingFlushWriter struct {
	header           http.Header
	err              error
	writeHeaderCalls int
	flushes          int
}

func (w *failingFlushWriter) Header() http.Header {
	return w.header
}

func (w *failingFlushWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *failingFlushWriter) WriteHeader(int) {
	w.writeHeaderCalls++
}

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
	w.writeHeaderCalls++
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

func writeTestSSEEvent(stream *SSEStreamWriter) error {
	return stream.WriteEvent(context.Background(), func(w io.Writer) error {
		_, err := io.WriteString(w, "data: event\n\n")
		return err
	})
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
