package jsonrpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

func TestSuppressedObservationReadiness(t *testing.T) {
	for _, hasID := range []bool{false, true} {
		t.Run(map[bool]string{false: "notification", true: "response control"}[hasID], func(t *testing.T) {
			var events []loomtransport.Event
			ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, e loomtransport.Event) {
				events = append(events, e)
			}))
			request := httptest.NewRequest(http.MethodPost, "/rpc", nil).WithContext(ctx)
			request.Pattern = "POST /rpc"
			observer, _ := loomtransport.BeginJSONRPCRequest(ctx, httptest.NewRecorder(), "clock", request)
			observer.SetJSONRPC("tick", "", 0, !hasID)
			ctx = loomtransport.WithRequestObserver(ctx, observer)
			sends := 0
			require.NoError(t, CompleteStream(ctx, hasID, "1", nil, func(*Response) error {
				sends++
				return nil
			}))
			observer.End()
			if hasID {
				require.Equal(t, 1, sends)
				require.Len(t, events, 2)
				return
			}
			require.Zero(t, sends)
			require.Len(t, events, 3)
			event := events[1]
			require.Equal(t, loomtransport.ReasonStreamFinalResponseSuppressed, event.Reason)
			require.Equal(t, "clock", event.Service)
			require.Equal(t, "tick", event.Method)
			require.Equal(t, "POST /rpc", event.Route)
			require.True(t, event.Notification)
		})
	}
}

// TestSSECompletionObservation covers terminal suppression and recoverable
// encoding failure at the shared protocol/writer boundary.
func TestSSECompletionObservation(t *testing.T) {
	for _, notification := range []bool{false, true} {
		t.Run(map[bool]string{false: "response", true: "notification"}[notification], func(t *testing.T) {
			var events []loomtransport.Event
			ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
				events = append(events, event)
			}))
			request := httptest.NewRequest(http.MethodPost, "/rpc", nil).WithContext(ctx)
			observer, capture := loomtransport.BeginJSONRPCRequest(ctx, httptest.NewRecorder(), "clock", request)
			observer.SetJSONRPC("tick", "r", 0, notification)
			ctx = loomtransport.WithRequestObserver(ctx, observer)
			writer := loomhttp.NewSSEStreamWriter(capture, ctx, loomtransport.TransportJSONRPC, loomhttp.StreamWritePolicy{})
			if !notification {
				require.Error(t, CompleteSSEStream(context.Background(), writer, true, func() *Response {
					return MakeSuccessResponse("r", make(chan int))
				}))
				require.False(t, writer.Started())
			}
			require.NoError(t, CompleteSSEStream(context.Background(), writer, !notification, func() *Response {
				if notification {
					t.Error("suppressed value evaluated")
				}
				return MakeErrorResponse("r", InternalError, "failed", nil)
			}))
			require.ErrorIs(t, CompleteSSEStream(context.Background(), writer, !notification, func() *Response {
				t.Error("duplicate final evaluated")
				return nil
			}), loomhttp.ErrSSEStreamClosed)
			require.True(t, errors.Is(writer.SendComment(context.Background(), "late"), loomhttp.ErrSSEStreamClosed))
			suppressed := 0
			for _, event := range events {
				if event.Reason == loomtransport.ReasonStreamFinalResponseSuppressed {
					suppressed++
					require.Equal(t, "clock", event.Service)
					require.Equal(t, "tick", event.Method)
				}
			}
			if notification {
				require.Equal(t, 1, suppressed)
			} else {
				require.Zero(t, suppressed)
			}
		})
	}
}

// TestSSECompletionObservationConcurrent checks the protocol marker follows only
// one admitted suppression, independent of cancellation and outside writer locks.
func TestSSECompletionObservationConcurrent(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "live", true: "canceled"}[canceled], func(t *testing.T) {
			var writer *loomhttp.SSEStreamWriter
			var mu sync.Mutex
			markers := 0
			callbackClosed := make(chan error, 2)
			ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
				if event.Reason == loomtransport.ReasonStreamFinalResponseSuppressed {
					mu.Lock()
					markers++
					mu.Unlock()
					// Reentry completes only if protocol observation runs outside writer ownership.
					callbackClosed <- writer.Close()
				}
			}))
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			rec := httptest.NewRecorder()
			writer = loomhttp.NewSSEStreamWriter(rec, ctx, loomtransport.TransportJSONRPC, loomhttp.StreamWritePolicy{}, loomhttp.WithSSEHandshake(func(context.Context) (loomhttp.SSEHandshake, error) {
				t.Error("suppressed completion resolved handshake")
				return loomhttp.SSEHandshake{Status: 201}, nil
			}))
			if canceled {
				cancel()
			}
			start := make(chan struct{})
			done := make(chan error, 2)
			for range 2 {
				go func() {
					<-start
					done <- CompleteSSEStream(ctx, writer, false, func() *Response {
						t.Error("notification response evaluated")
						return nil
					})
				}()
			}
			close(start)
			first, second := <-done, <-done
			if first != nil {
				first, second = second, first
			}
			require.NoError(t, first)
			require.ErrorIs(t, second, loomhttp.ErrSSEStreamClosed)
			require.NoError(t, <-callbackClosed)
			mu.Lock()
			require.Equal(t, 1, markers)
			mu.Unlock()
			require.False(t, writer.Started())
			require.False(t, rec.Flushed)
			require.Empty(t, rec.Header())
			require.Empty(t, rec.Body.String())
		})
	}
}
