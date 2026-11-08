package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

func TestSSEHandshakeOpenTriggers(t *testing.T) {
	for name, trigger := range handshakeTriggers() {
		t.Run(name, func(t *testing.T) {
			w := &deadlineResponseWriter{header: make(http.Header)}
			calls := 0
			s := newHandshakeWriter(w, WithSSEHandshake(func(context.Context) (SSEHandshake, error) {
				calls++
				return SSEHandshake{Status: 201, Header: http.Header{"X-Selected": {"yes"}}}, nil
			}))
			require.NoError(t, trigger(s))
			require.Equal(t, 201, w.status)
			require.Equal(t, "yes", w.header.Get("X-Selected"))
			require.Equal(t, "text/event-stream", w.header.Get("Content-Type"))
			require.True(t, s.Started())
			require.Equal(t, 1, calls)
			require.NoError(t, s.Close())
			require.NoError(t, s.FinalizeSuccess(context.Background()))
			require.Equal(t, 1, w.writeHeaderCalls)
			require.Equal(t, 1, calls)
			if name == "nil" || name == "close" || name == "finalize" {
				require.Empty(t, w.body.String())
			}
		})
	}
}

func TestSSEHandshakeResolvesWithRequestContext(t *testing.T) {
	type requestKey struct{}
	for name, trigger := range handshakeTriggers() {
		t.Run(name, func(t *testing.T) {
			w := &deadlineResponseWriter{header: make(http.Header)}
			request := context.WithValue(context.Background(), requestKey{}, "request")
			calls := 0
			s := NewSSEStreamWriter(w, request, loomtransport.TransportHTTP, StreamWritePolicy{}, WithSSEHandshake(func(ctx context.Context) (SSEHandshake, error) {
				calls++
				if ctx.Value(requestKey{}) != "request" {
					t.Error("implicit metadata used the operation context")
				}
				return SSEHandshake{Status: 201}, nil
			}))
			require.NoError(t, trigger(s))
			require.Equal(t, 1, calls)
			require.Equal(t, 201, w.status)
		})
	}
}

func TestSSEHandshakeFailureAdmission(t *testing.T) {
	for name, trigger := range handshakeTriggers() {
		t.Run(name, func(t *testing.T) {
			w := &deadlineResponseWriter{header: http.Header{"X-Existing": {"retained"}}}
			bad := errors.New("missing status or required header")
			s := newHandshakeWriter(w, WithSSEHandshake(func(context.Context) (SSEHandshake, error) {
				return SSEHandshake{}, bad
			}))
			require.ErrorIs(t, trigger(s), bad)
			require.False(t, s.Started())
			require.Zero(t, w.writeHeaderCalls)
			require.Equal(t, http.Header{"X-Existing": {"retained"}}, w.header)
			require.Nil(t, s.failErr)
			err := s.SetHandshake(func() (SSEHandshake, error) {
				return SSEHandshake{Status: 201}, nil
			})
			switch name {
			case "open", "comment", "event":
				require.NoError(t, err)
				require.NoError(t, s.Open(context.Background()))
				require.Equal(t, 201, w.status)
			default:
				require.ErrorIs(t, err, ErrSSEResponseCommitted)
				require.ErrorIs(t, s.Open(context.Background()), ErrSSEStreamClosed)
				require.NoError(t, s.Close())
				require.Zero(t, w.writeHeaderCalls)
			}
		})
	}
}

func TestSSEHandshakeSnapshot(t *testing.T) {
	w := &deadlineResponseWriter{header: make(http.Header)}
	s := newHandshakeWriter(w)
	header := http.Header{"X-Selected": {"original"}}
	require.NoError(t, s.SetHandshake(func() (SSEHandshake, error) {
		return SSEHandshake{Status: 201, Header: header}, nil
	}))
	header["X-Selected"][0] = "mutated"
	bad := errors.New("conversion")
	require.ErrorIs(t, s.SetHandshake(func() (SSEHandshake, error) {
		return SSEHandshake{Status: 202}, bad
	}), bad)
	require.NoError(t, s.Open(context.Background()))
	require.Equal(t, 201, w.status)
	require.Equal(t, "original", w.header.Get("X-Selected"))
	require.ErrorIs(t, s.SetHandshake(func() (SSEHandshake, error) {
		t.Error("opened setter invoked")
		return SSEHandshake{}, nil
	}), ErrSSEResponseCommitted)
}

func TestSSEHandshakeAdmissionRace(t *testing.T) {
	t.Run("open wins", func(t *testing.T) {
		w := &deadlineResponseWriter{header: make(http.Header)}
		entered, release := make(chan struct{}), make(chan struct{})
		s := newHandshakeWriter(w, WithSSEHandshake(func(context.Context) (SSEHandshake, error) {
			close(entered)
			<-release
			return SSEHandshake{Status: 201, Header: http.Header{"X-Selected": {"original"}}}, nil
		}))
		opened, setDone := make(chan error, 1), make(chan error, 1)
		go func() {
			opened <- s.Open(context.Background())
		}()
		<-entered
		setterStarted := make(chan struct{})
		go func() {
			close(setterStarted)
			setDone <- s.SetHandshake(func() (SSEHandshake, error) {
				t.Error("refused setter invoked its conversion callback")
				return SSEHandshake{Status: 202}, nil
			})
		}()
		<-setterStarted
		select {
		case err := <-setDone:
			close(release)
			require.NoError(t, <-opened)
			t.Errorf("setter returned before first-open admission completed: %v", err)
			return
		default:
		}
		close(release)
		require.NoError(t, <-opened)
		require.ErrorIs(t, <-setDone, ErrSSEResponseCommitted)
		require.Equal(t, 201, w.status)
		require.Equal(t, "original", w.header.Get("X-Selected"))
		require.Equal(t, 1, w.writeHeaderCalls)
	})
	for name, op := range map[string]func(*SSEStreamWriter) error{
		"open": func(s *SSEStreamWriter) error {
			return s.Open(context.Background())
		},
		"close": func(s *SSEStreamWriter) error {
			return s.Close()
		},
		"claim": func(s *SSEStreamWriter) error {
			s.ClaimFailure()
			return nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := &deadlineResponseWriter{header: make(http.Header)}
			s := newHandshakeWriter(w)
			entered, release := make(chan struct{}), make(chan struct{})
			setDone, opDone := make(chan error, 1), make(chan error, 1)
			go func() {
				setDone <- s.SetHandshake(func() (SSEHandshake, error) {
					close(entered)
					<-release
					return SSEHandshake{Status: 201, Header: http.Header{"X-Selected": {"atomic"}}}, nil
				})
			}()
			<-entered
			go func() {
				opDone <- op(s)
			}()
			close(release)
			require.NoError(t, <-setDone)
			require.NoError(t, <-opDone)
			if name != "claim" {
				require.Equal(t, 201, w.status)
				require.Equal(t, "atomic", w.header.Get("X-Selected"))
			} else {
				require.Zero(t, w.writeHeaderCalls)
			}
			require.ErrorIs(t, s.SetHandshake(func() (SSEHandshake, error) {
				t.Error("rejected setter invoked")
				return SSEHandshake{}, nil
			}), ErrSSEResponseCommitted)
		})
	}
}

func TestSSEHandshakeClosesInformationalAdmission(t *testing.T) {
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events", nil)
	lifecycle := NewHandlerLifecycle(w, request, "svc", "watch")
	exchange := lifecycle.Exchange()
	bad := errors.New("required metadata")
	calls := 0
	s := newHandshakeWriter(lifecycle.Writer(), WithSSEExchange(exchange), WithSSEHandshake(func(context.Context) (SSEHandshake, error) {
		calls++
		require.ErrorIs(t, exchange.Send(103, nil), ErrInformationalClosed)
		if calls == 1 {
			return SSEHandshake{}, bad
		}
		return SSEHandshake{Status: 201}, nil
	}))
	require.ErrorIs(t, s.Open(context.Background()), bad)
	require.False(t, s.Started())
	require.Empty(t, w.Header())
	require.ErrorIs(t, exchange.Send(103, nil), ErrInformationalClosed)
	require.NoError(t, s.Open(context.Background()))
	require.Equal(t, 201, w.Code)
}

func handshakeTriggers() map[string]func(*SSEStreamWriter) error {
	return map[string]func(*SSEStreamWriter) error{
		"open": func(s *SSEStreamWriter) error {
			return s.Open(context.Background())
		},
		"comment": func(s *SSEStreamWriter) error {
			return s.SendComment(context.Background(), "ready")
		},
		"event": func(s *SSEStreamWriter) error {
			return s.WriteEvent(context.Background(), func(w io.Writer) error {
				_, err := io.WriteString(w, "data: value\n\n")
				return err
			})
		},
		"message": func(s *SSEStreamWriter) error {
			return s.Complete(context.Background(), func() (SSEMessage, error) {
				return SSEMessage{Data: "value"}, nil
			})
		},
		"nil": func(s *SSEStreamWriter) error {
			return s.Complete(context.Background(), nil)
		},
		"close": func(s *SSEStreamWriter) error {
			return s.Close()
		},
		"finalize": func(s *SSEStreamWriter) error {
			return s.FinalizeSuccess(context.Background())
		},
	}
}

func newHandshakeWriter(w http.ResponseWriter, opts ...SSEStreamOption) *SSEStreamWriter {
	return NewSSEStreamWriter(w, context.Background(), loomtransport.TransportHTTP, StreamWritePolicy{}, opts...)
}
