package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

type (
	// cancelHookWriter wraps a real server ResponseWriter so write deadlines
	// affect the underlying connection, and lets tests cancel a call context at
	// a precise point of an in-flight operation.
	cancelHookWriter struct {
		inner       http.ResponseWriter
		lock        sync.Mutex
		expired     chan struct{}
		expiredOnce sync.Once
		afterFlush  func()
	}

	// failingWriteWriter rejects every body write without accepting bytes.
	failingWriteWriter struct {
		header http.Header
		err    error
	}

	// cancelRaceCase describes one per-call cancellation race.
	cancelRaceCase struct {
		name     string
		first    func(*SSEStreamWriter, *cancelHookWriter, context.Context, context.CancelFunc) error
		wantErr  error
		wantBody string
	}
)

func TestSSEStreamWriterCancellationDoesNotPoisonDeadline(t *testing.T) {
	tests := []cancelRaceCase{
		{
			name: "cancel during event write",
			first: func(stream *SSEStreamWriter, hook *cancelHookWriter, ctx context.Context, cancel context.CancelFunc) error {
				return stream.WriteEvent(ctx, func(w io.Writer) error {
					cancel()
					hook.waitExpired()
					_, err := io.WriteString(w, "data: first\n\n")
					return err
				})
			},
			wantErr:  context.Canceled,
			wantBody: "data: first\n\n: after\n\n",
		},
		{
			name: "cancel after flush completes before stop",
			first: func(stream *SSEStreamWriter, hook *cancelHookWriter, ctx context.Context, cancel context.CancelFunc) error {
				hook.setAfterFlush(func() {
					cancel()
					hook.waitExpired()
				})
				return stream.SendComment(ctx, "first")
			},
			wantBody: ": first\n\n: after\n\n",
		},
	}
	for _, policy := range []StreamWritePolicy{{}, mustStreamWritePolicy(t, time.Minute)} {
		for _, test := range tests {
			t.Run(test.name+"/timeout="+policy.Timeout().String(), func(t *testing.T) {
				runCancelRace(t, policy, test)
			})
		}
	}
}

func TestSSEStreamWriterFailureIsTerminal(t *testing.T) {
	writeErr := errors.New("write failed")
	flushErr := errors.New("flush failed")
	tests := []struct {
		name    string
		writer  func() (http.ResponseWriter, func() string)
		first   func(*SSEStreamWriter) error
		wantErr error
	}{
		{
			name: "partial event write",
			writer: func() (http.ResponseWriter, func() string) {
				w := &deadlineResponseWriter{header: make(http.Header)}
				return w, w.body.String
			},
			first: func(stream *SSEStreamWriter) error {
				return stream.WriteEvent(context.Background(), func(w io.Writer) error {
					if _, err := io.WriteString(w, "data: half"); err != nil {
						return err
					}
					return writeErr
				})
			},
			wantErr: writeErr,
		},
		{
			name: "underlying write error without bytes",
			writer: func() (http.ResponseWriter, func() string) {
				w := &failingWriteWriter{header: make(http.Header), err: writeErr}
				return w, func() string {
					return ""
				}
			},
			first:   writeTestSSEEvent,
			wantErr: writeErr,
		},
		{
			name: "event flush",
			writer: func() (http.ResponseWriter, func() string) {
				w := &failingFlushWriter{header: make(http.Header), err: flushErr}
				return w, func() string {
					return ""
				}
			},
			first:   writeTestSSEEvent,
			wantErr: flushErr,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, body := test.writer()
			stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})
			require.ErrorIs(t, test.first(stream), test.wantErr)
			before := body()

			wrote := false
			err := stream.WriteEvent(context.Background(), func(io.Writer) error {
				wrote = true
				return nil
			})
			require.ErrorIs(t, err, ErrSSEStreamClosed)
			require.ErrorIs(t, err, test.wantErr)
			require.False(t, wrote)
			require.ErrorIs(t, stream.SendComment(context.Background(), "late"), ErrSSEStreamClosed)
			require.ErrorIs(t, stream.Open(context.Background()), ErrSSEStreamClosed)
			require.Equal(t, before, body())
		})
	}
}

func TestSSEStreamWriterCallbackErrorWithoutBytesKeepsStream(t *testing.T) {
	encodeErr := errors.New("encode failed")
	w := &deadlineResponseWriter{header: make(http.Header)}
	stream := NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{})

	err := stream.WriteEvent(context.Background(), func(io.Writer) error {
		return encodeErr
	})
	require.ErrorIs(t, err, encodeErr)
	require.NotErrorIs(t, err, ErrSSEStreamClosed)
	require.Empty(t, w.body.String())
	require.NoError(t, writeTestSSEEvent(stream))
	require.NoError(t, stream.SendComment(context.Background(), "after"))
	require.NoError(t, stream.Open(context.Background()))
	require.Equal(t, "data: event\n\n: after\n\n", w.body.String())
}

func runCancelRace(t *testing.T, policy StreamWritePolicy, test cancelRaceCase) {
	t.Helper()
	type result struct {
		first, second error
	}
	results := make(chan result, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hook := &cancelHookWriter{inner: w, expired: make(chan struct{})}
		stream := NewSSEStreamWriter(hook, r.Context(), loomtransport.TransportHTTP, policy)
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		var res result
		res.first = test.first(stream, hook, ctx, cancel)
		hook.setAfterFlush(nil)
		res.second = stream.SendComment(r.Context(), "after")
		results <- res
	}))
	defer server.Close()

	response, err := server.Client().Get(server.URL)
	require.NoError(t, err)
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	res := <-results
	if test.wantErr == nil {
		require.NoError(t, res.first)
	} else {
		require.ErrorIs(t, res.first, test.wantErr)
	}
	require.NoError(t, res.second)
	require.NoError(t, readErr)
	require.NoError(t, closeErr)
	require.Equal(t, test.wantBody, string(body))
}

func mustStreamWritePolicy(t *testing.T, timeout time.Duration) StreamWritePolicy {
	t.Helper()
	policy, err := NewStreamWritePolicy(timeout)
	require.NoError(t, err)
	return policy
}

func (w *failingWriteWriter) Header() http.Header {
	return w.header
}

func (w *failingWriteWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func (w *failingWriteWriter) WriteHeader(int) {}

func (w *failingWriteWriter) Flush() {}

func (w *cancelHookWriter) Header() http.Header {
	return w.inner.Header()
}

func (w *cancelHookWriter) Write(p []byte) (int, error) {
	return w.inner.Write(p)
}

func (w *cancelHookWriter) WriteHeader(status int) {
	w.inner.WriteHeader(status)
}

func (w *cancelHookWriter) FlushError() error {
	if err := http.NewResponseController(w.inner).Flush(); err != nil {
		return err
	}
	w.lock.Lock()
	hook := w.afterFlush
	w.lock.Unlock()
	if hook != nil {
		hook()
	}
	return nil
}

func (w *cancelHookWriter) SetWriteDeadline(deadline time.Time) error {
	if err := http.NewResponseController(w.inner).SetWriteDeadline(deadline); err != nil {
		return err
	}
	if !deadline.IsZero() && !deadline.After(time.Now()) {
		w.expiredOnce.Do(func() {
			close(w.expired)
		})
	}
	return nil
}

func (w *cancelHookWriter) setAfterFlush(hook func()) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.afterFlush = hook
}

func (w *cancelHookWriter) waitExpired() {
	<-w.expired
}
