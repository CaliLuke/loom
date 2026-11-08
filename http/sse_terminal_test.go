package http

import (
	"context"
	"errors"
	"net/http"
	"testing"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

func TestSSETerminalSuccessAdmission(t *testing.T) {
	for _, prior := range []string{"live", "success", "suppressed", "claim", "cleanup", "iofail"} {
		for _, operation := range []string{"close", "finalize"} {
			t.Run(prior+"/"+operation, func(t *testing.T) {
				failure := errors.New("flush")
				w := &deadlineResponseWriter{header: make(http.Header)}
				var target http.ResponseWriter = w
				if prior == "iofail" {
					target = &failingFlushWriter{header: make(http.Header), err: failure}
				}
				s := newHandshakeWriter(target)
				switch prior {
				case "success":
					require.NoError(t, s.Complete(context.Background(), nil))
				case "suppressed":
					require.NoError(t, s.Suppress())
				case "claim":
					require.False(t, s.ClaimFailure())
				case "cleanup":
					s.End()
				case "iofail":
					require.ErrorIs(t, s.Open(context.Background()), failure)
				}
				if operation == "close" {
					require.NoError(t, s.Close())
				} else {
					require.NoError(t, s.FinalizeSuccess(context.Background()))
				}
				require.Equal(t, prior == "live" || prior == "success" || prior == "iofail", s.Started())
				if prior == "live" || prior == "success" {
					require.Equal(t, http.StatusOK, w.status)
				}
				require.ErrorIs(t, s.Complete(context.Background(), func() (SSEMessage, error) {
					t.Error("closed preparation invoked")
					return SSEMessage{}, nil
				}), ErrSSEStreamClosed)
				if prior == "iofail" {
					require.ErrorIs(t, s.Open(context.Background()), failure)
				}
			})
		}
	}
}

func TestSSETerminalCancellation(t *testing.T) {
	for _, opened := range []bool{false, true} {
		state := "unopened"
		if opened {
			state = "opened"
		}
		t.Run(state, func(t *testing.T) {
			for _, operation := range []string{"close", "finalize", "nil", "message"} {
				t.Run(operation, func(t *testing.T) {
					request, cancel := context.WithCancel(context.Background())
					defer cancel()
					w := &deadlineResponseWriter{header: make(http.Header)}
					s := NewSSEStreamWriter(w, request, loomtransport.TransportHTTP, StreamWritePolicy{})
					if opened {
						require.NoError(t, s.Open(context.Background()))
					}
					var err error
					switch operation {
					case "close":
						cancel()
						err = s.Close()
					case "finalize":
						cancel()
						err = s.FinalizeSuccess(context.Background())
					case "nil":
						cancel()
						err = s.Complete(context.Background(), nil)
					case "message":
						err = s.Complete(context.Background(), func() (SSEMessage, error) {
							cancel()
							return SSEMessage{Data: "unused"}, nil
						})
					}
					require.ErrorIs(t, err, context.Canceled)
					require.True(t, s.closed)
					require.Nil(t, s.failErr)
					require.Equal(t, opened, s.Started())
					if opened {
						require.Equal(t, 1, w.writeHeaderCalls)
					} else {
						require.Zero(t, w.writeHeaderCalls)
					}
					require.Empty(t, w.body.String())
					require.NoError(t, s.Close())
				})
			}
		})
	}
}

func TestSSESuppressionAdmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := &deadlineResponseWriter{header: make(http.Header)}
	s := NewSSEStreamWriter(w, ctx, loomtransport.TransportHTTP, StreamWritePolicy{}, WithSSEHandshake(func(context.Context) (SSEHandshake, error) {
		t.Error("suppression resolved handshake")
		return SSEHandshake{}, nil
	}))
	require.NoError(t, s.Suppress())
	require.ErrorIs(t, s.Suppress(), ErrSSEStreamClosed)
	require.NoError(t, s.Close())
	require.NoError(t, s.FinalizeSuccess(context.Background()))
	require.ErrorIs(t, s.Complete(context.Background(), nil), ErrSSEStreamClosed)
	s.End()
	require.Zero(t, w.writeHeaderCalls)
	require.False(t, s.Started())
	for name, other := range map[string]func(*SSEStreamWriter) error{
		"close": func(s *SSEStreamWriter) error {
			return s.Close()
		},
		"finalize": func(s *SSEStreamWriter) error {
			return s.FinalizeSuccess(context.Background())
		},
		"complete": func(s *SSEStreamWriter) error {
			return s.Complete(context.Background(), nil)
		},
		"claim": func(s *SSEStreamWriter) error {
			s.ClaimFailure()
			return nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			stream := newHandshakeWriter(&deadlineResponseWriter{header: make(http.Header)})
			start := make(chan struct{})
			a, b := make(chan error, 1), make(chan error, 1)
			go func() {
				<-start
				a <- stream.Suppress()
			}()
			go func() {
				<-start
				b <- other(stream)
			}()
			close(start)
			suppressed, opErr := <-a, <-b
			if suppressed == nil {
				require.False(t, stream.Started())
				if name == "complete" {
					require.ErrorIs(t, opErr, ErrSSEStreamClosed)
				} else {
					require.NoError(t, opErr)
				}
			} else {
				require.ErrorIs(t, suppressed, ErrSSEStreamClosed)
				require.NoError(t, opErr)
				require.Equal(t, name != "claim", stream.Started())
			}
		})
	}
}

func TestSSETerminalObserverOutsideOwner(t *testing.T) {
	for _, operation := range []string{"close", "finalize", "nil", "suppress"} {
		t.Run(operation, func(t *testing.T) {
			var s *SSEStreamWriter
			kinds := []loomtransport.EventKind{}
			closing := operation != "suppress"
			ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, e loomtransport.Event) {
				kinds = append(kinds, e.Kind)
				require.True(t, s.metadata.lock.TryLock(), "observer invoked under writer lock")
				s.metadata.lock.Unlock()
				require.True(t, s.Started())
				if closing {
					require.NoError(t, s.Close())
				}
			}))
			s = NewSSEStreamWriter(&deadlineResponseWriter{header: make(http.Header)}, ctx, loomtransport.TransportHTTP, StreamWritePolicy{})
			// Suppression of an already opened stream must drain its close after unlock.
			if operation == "suppress" {
				require.NoError(t, s.Open(ctx))
				closing = true
			}
			switch operation {
			case "close":
				require.NoError(t, s.Close())
			case "finalize":
				require.NoError(t, s.FinalizeSuccess(ctx))
			case "nil":
				require.NoError(t, s.Complete(ctx, nil))
			case "suppress":
				require.NoError(t, s.Suppress())
			}
			require.Equal(t, []loomtransport.EventKind{loomtransport.EventKindStreamOpen, loomtransport.EventKindStreamClose}, kinds)
		})
	}
}
