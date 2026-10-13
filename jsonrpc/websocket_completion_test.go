package jsonrpc

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketCompletedResponsesSurviveDelayedRecv(t *testing.T) {
	srv := newScriptedServer(t)
	errs := &recordedErrors{}
	conn, serverConn := dialScripted(t, srv, errs.handle)
	require.True(t, conn.Acquire())
	stream := NewWebSocketClientStream(t.Context(), conn, "echo", NewStreamConfig(
		WithRequestTimeout(50*time.Millisecond), WithErrorHandler(errs.handle)))
	t.Cleanup(func() {
		require.NoError(t, stream.Close())
	})
	const count = 32
	calls := make([]*webSocketCall, 0, count)
	for range count {
		require.NoError(t, stream.Send(t.Context(), "request"))
		request := srv.next(t)
		stream.mu.Lock()
		calls = append(calls, stream.queue[len(stream.queue)-1])
		stream.mu.Unlock()
		writeRaw(t, serverConn, fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"held"}`, IDToString(request.ID)))
	}
	// The read goroutine has actually delivered every response before the
	// delayed receiver starts; a successful server write alone is not enough.
	require.Eventually(t, func() bool {
		for _, call := range calls {
			if len(call.result) == 0 {
				return false
			}
		}
		return true
	}, time.Second, time.Millisecond)
	time.Sleep(75 * time.Millisecond)
	for range count {
		response, err := stream.Recv(t.Context())
		if assert.NoError(t, err) {
			assert.JSONEq(t, `"held"`, string(response.Result))
		}
	}
	assert.Empty(t, errs.kindsSeen(), "completed responses must not produce timeout reports")
}

func TestWebSocketDeliveredCallStopsRequestTimer(t *testing.T) {
	srv := newScriptedServer(t)
	conn, serverConn := dialScripted(t, srv, nil)
	stream := openStream(t, t.Context(), conn, nil)
	require.NoError(t, stream.Send(t.Context(), "request"))
	request := srv.next(t)
	stream.mu.Lock()
	call := stream.queue[0]
	stream.mu.Unlock()
	writeRaw(t, serverConn, fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"result":"held"}`, IDToString(request.ID)))
	require.Eventually(t, func() bool {
		return len(call.result) == 1
	}, time.Second, time.Millisecond)
	assert.False(t, call.timer.Stop(), "delivery must release the pending timer before publishing its result")
	response, err := stream.Recv(t.Context())
	require.NoError(t, err)
	assert.JSONEq(t, `"held"`, string(response.Result))
}

func TestWebSocketCallCompletionWinner(t *testing.T) {
	for _, first := range []string{"response", "connection failure", "timeout"} {
		t.Run(first, func(t *testing.T) {
			errs := &recordedErrors{}
			conn := &WebSocketClientConn{calls: make(map[string]*webSocketCall), handler: errs.handle}
			call := &webSocketCall{result: make(chan webSocketResult, 1)}
			require.NoError(t, conn.register(call, time.Hour))
			t.Cleanup(func() {
				call.timer.Stop()
			})
			response := &RawResponse{ID: call.id}
			timeout := errors.New("observed request timeout")
			failure := errors.New("native connection failure")
			switch first {
			case "response":
				// Pause the competing proposer at the exact connection lock.
				// Removal and publication must stay inside this critical section.
				conn.mu.Lock()
				loser := make(chan bool, 1)
				go func() {
					loser <- conn.complete(call, webSocketResult{err: timeout})
				}()
				assert.True(t, conn.completeLocked(call, webSocketResult{response: response}))
				assert.Len(t, call.result, 1)
				conn.mu.Unlock()
				assert.False(t, <-loser)
			case "connection failure":
				conn.fail(failure)
				assert.False(t, conn.complete(call, webSocketResult{err: timeout}))
			case "timeout":
				require.True(t, conn.complete(call, webSocketResult{err: timeout}))
				conn.route(response)
				assert.Equal(t, []StreamErrorType{StreamErrorOrphaned}, errs.kindsSeen())
			}
			assert.False(t, call.timer.Stop())
			assert.Empty(t, conn.calls)
			result := <-call.result
			switch first {
			case "response":
				assert.Same(t, response, result.response)
			case "connection failure":
				assert.ErrorIs(t, result.err, failure)
			case "timeout":
				assert.ErrorIs(t, result.err, timeout)
			}
			assert.Empty(t, call.result, "a losing completion must not publish a second result")
		})
	}
}

func TestWebSocketCallTimerInitializedBeforePublication(t *testing.T) {
	conn := &WebSocketClientConn{calls: make(map[string]*webSocketCall)}
	call := &webSocketCall{result: make(chan webSocketResult, 1)}
	require.NoError(t, conn.register(call, time.Hour))
	conn.mu.Lock()
	published := conn.calls[call.id]
	assert.Same(t, call, published)
	assert.NotNil(t, published.timer, "routing must never observe a registration without its timer")
	conn.completeLocked(published, webSocketResult{err: errors.New("immediate read failure")})
	conn.mu.Unlock()
	assert.False(t, call.timer.Stop())
	conn.fail(errors.New("closed connection"))
	rejected := &webSocketCall{result: make(chan webSocketResult, 1)}
	require.Error(t, conn.register(rejected, time.Hour))
	assert.Nil(t, rejected.timer, "rejected registration must not acquire a timer")
}

func TestWebSocketCallCancellationPreservesCompletedResult(t *testing.T) {
	for _, outcome := range []string{"success", "protocol error", "connection error"} {
		t.Run(outcome, func(t *testing.T) {
			srv := newScriptedServer(t)
			errs := &recordedErrors{}
			conn, _ := dialScripted(t, srv, errs.handle)
			stream := openStream(t, t.Context(), conn, errs.handle)
			t.Cleanup(func() {
				require.NoError(t, stream.Close())
			})
			call, err := stream.register(false)
			require.NoError(t, err)
			response := &RawResponse{ID: call.id}
			failure := errors.New("read failed before cancellation")
			switch outcome {
			case "success":
				conn.route(response)
			case "protocol error":
				response.Error = &RawErrorResponse{Code: -32000, Message: "rejected"}
				conn.route(response)
			case "connection error":
				conn.fail(failure)
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			got, err := stream.await(ctx, call, false)
			switch outcome {
			case "success":
				require.NoError(t, err)
				assert.Same(t, response, got)
			case "protocol error":
				assert.ErrorIs(t, err, response.Error)
				assert.Equal(t, []StreamErrorType{StreamErrorProtocol}, errs.kindsSeen())
			case "connection error":
				assert.ErrorIs(t, err, failure)
			}
			assert.Empty(t, stream.own)
			assert.False(t, call.timer.Stop())
		})
	}
}

func TestWebSocketCallAndStreamCancellationCompletePendingCall(t *testing.T) {
	for _, end := range []string{"call cancellation", "stream context", "stream close"} {
		t.Run(end, func(t *testing.T) {
			srv := newScriptedServer(t)
			conn, _ := dialScripted(t, srv, nil)
			streamCtx, endStream := context.WithCancel(t.Context())
			defer endStream()
			stream := openStream(t, streamCtx, conn, nil)
			t.Cleanup(func() {
				require.NoError(t, stream.Close())
			})
			call, err := stream.register(false)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var want error = context.Canceled
			switch end {
			case "call cancellation":
				cancel()
			case "stream context":
				endStream()
				require.Eventually(t, func() bool {
					stream.mu.Lock()
					defer stream.mu.Unlock()
					return stream.ended && len(stream.own) == 0
				}, time.Second, time.Millisecond)
			case "stream close":
				require.NoError(t, stream.Close())
				want = ErrWebSocketClientStreamClosed
			}
			_, err = stream.await(ctx, call, false)
			assert.ErrorIs(t, err, want)
			assert.False(t, call.timer.Stop())
			conn.mu.Lock()
			assert.Empty(t, conn.calls)
			conn.mu.Unlock()
			stream.mu.Lock()
			assert.Empty(t, stream.own)
			stream.mu.Unlock()
		})
	}
}
