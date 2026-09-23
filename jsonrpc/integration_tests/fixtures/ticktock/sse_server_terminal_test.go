package ticktock

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	clock "example.com/ticktock/gen/clock"
	clockjssvr "example.com/ticktock/gen/jsonrpc/clock/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type (
	// scriptedClock runs a test-provided Tick body against the generated
	// server stream.
	scriptedClock struct {
		tick func(context.Context, clock.TickServerStream) error
	}

	// recordingSSEResponse records SSE output and can hold the first body
	// write open to force concurrent stream operations to queue behind it.
	recordingSSEResponse struct {
		header     http.Header
		mu         sync.Mutex
		body       bytes.Buffer
		holdFirst  chan struct{}
		firstWrite chan struct{}
		once       sync.Once
	}

	terminalCase struct {
		name     string
		hasID    bool
		terminal func(context.Context, clock.TickServerStream) error
		want     []string
	}

	followUpCase struct {
		name string
		call func(context.Context, clock.TickServerStream) error
	}
)

func TestJSONRPCGeneratedSSEServerRejectsWritesAfterTerminalResponse(t *testing.T) {
	terminals := []terminalCase{
		{name: "SendAndClose with ID", hasID: true, terminal: sendAndClose("done"), want: []string{"result"}},
		{name: "SendError with ID", hasID: true, terminal: sendError(), want: []string{"error"}},
		{name: "SendAndClose without ID", terminal: sendAndClose("done"), want: nil},
		{name: "SendError without ID", terminal: sendError(), want: nil},
	}
	followUps := []followUpCase{
		{name: "Send", call: send("late")},
		{name: "SendAndClose", call: sendAndClose("late")},
		{name: "SendError", call: sendError()},
		{name: "SendComment", call: func(ctx context.Context, stream clock.TickServerStream) error {
			return controlOf(stream).SendComment(ctx, "late")
		}},
		{name: "Open", call: func(ctx context.Context, stream clock.TickServerStream) error {
			return controlOf(stream).Open(ctx)
		}},
	}
	for _, terminal := range terminals {
		for _, followUp := range followUps {
			t.Run(terminal.name+" then "+followUp.name, func(t *testing.T) {
				var terminalErr, followUpErr error
				w := newRecordingSSEResponse(false)
				handlerErrs := serveScriptedTick(t, w, terminal.hasID, func(ctx context.Context, stream clock.TickServerStream) error {
					terminalErr = terminal.terminal(ctx, stream)
					followUpErr = followUp.call(ctx, stream)
					return nil
				})

				require.NoError(t, terminalErr)
				require.ErrorIs(t, followUpErr, loomhttp.ErrSSEStreamClosed)
				require.Equal(t, terminal.want, w.frameKinds(t))
				require.Empty(t, handlerErrs)
			})
		}
	}
}

// TestJSONRPCGeneratedSSEServerStressConcurrentSendAndClose is a stress
// regression guard for the Send/SendAndClose check-then-write race. The
// ordering of racing senders is scheduler dependent; the deterministic
// assertions are that SendAndClose cannot complete while an earlier write is
// parked, and that every Send after it returns loomhttp.ErrSSEStreamClosed.
func TestJSONRPCGeneratedSSEServerStressConcurrentSendAndClose(t *testing.T) {
	const (
		iterations = 25
		senders    = 16
	)
	for range iterations {
		var (
			closeErr     error
			parkedClosed bool
			sendErrs     []error
			lateErrs     []error
		)
		w := newRecordingSSEResponse(true)
		handlerErrs := serveScriptedTick(t, w, true, func(ctx context.Context, stream clock.TickServerStream) error {
			var wg sync.WaitGroup
			results := make(chan error, senders+1)
			wg.Go(func() {
				results <- stream.Send(ctx, tickResult("held"))
			})
			w.waitFirstWrite(t)

			start := make(chan struct{})
			for range senders {
				wg.Go(func() {
					<-start
					results <- stream.Send(ctx, tickResult("racing"))
				})
			}
			closeDone := make(chan error, 1)
			go func() {
				<-start
				closeDone <- stream.SendAndClose(ctx, tickResult("done"))
			}()
			close(start)
			select {
			case closeErr = <-closeDone:
				parkedClosed = true
			default:
			}
			close(w.holdFirst)
			if !parkedClosed {
				closeErr = <-closeDone
			}
			wg.Wait()
			close(results)
			for err := range results {
				sendErrs = append(sendErrs, err)
			}
			for range senders {
				lateErrs = append(lateErrs, stream.Send(ctx, tickResult("late")))
			}
			return nil
		})

		require.False(t, parkedClosed, "SendAndClose completed while an earlier write was parked")
		kinds := w.frameKinds(t)
		require.NotEmpty(t, kinds)
		require.Equal(t, "result", kinds[len(kinds)-1], "no event may follow the final response: %v", kinds)
		require.NoError(t, closeErr)
		require.Empty(t, handlerErrs)
		delivered := 0
		for _, err := range sendErrs {
			if err == nil {
				delivered++
				continue
			}
			require.ErrorIs(t, err, loomhttp.ErrSSEStreamClosed)
		}
		require.Len(t, kinds, delivered+1)
		for _, err := range lateErrs {
			require.ErrorIs(t, err, loomhttp.ErrSSEStreamClosed)
		}
	}
}

func TestJSONRPCGeneratedSSEServerHonorsCallContext(t *testing.T) {
	cases := []followUpCase{
		{name: "Send", call: send("canceled")},
		{name: "SendAndClose", call: sendAndClose("canceled")},
		{name: "SendError", call: sendError()},
		{name: "SendComment", call: func(ctx context.Context, stream clock.TickServerStream) error {
			return controlOf(stream).SendComment(ctx, "canceled")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var callErr error
			w := newRecordingSSEResponse(false)
			handlerErrs := serveScriptedTick(t, w, true, func(ctx context.Context, stream clock.TickServerStream) error {
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				callErr = tc.call(canceled, stream)
				return nil
			})

			require.ErrorIs(t, callErr, context.Canceled)
			require.Empty(t, w.frameKinds(t))
			require.Empty(t, handlerErrs)
		})
	}
}

func TestJSONRPCGeneratedSSEServerEndpointErrorAfterFinalResponse(t *testing.T) {
	late := errors.New("late failure")
	w := newRecordingSSEResponse(false)
	handlerErrs := serveScriptedTick(t, w, true, func(ctx context.Context, stream clock.TickServerStream) error {
		require.NoError(t, stream.SendAndClose(ctx, tickResult("done")))
		return late
	})

	require.Equal(t, []string{"result"}, w.frameKinds(t))
	require.Len(t, handlerErrs, 1)
	require.ErrorIs(t, handlerErrs[0], loomhttp.ErrSSEStreamClosed)
	require.ErrorIs(t, handlerErrs[0], late)
}

func TestJSONRPCGeneratedSSEServerEncodeFailureKeepsStreamOpen(t *testing.T) {
	cases := []followUpCase{
		{name: "SendAndClose encode failure then endpoint error delivers an error frame", call: sendAndClose("\xff")},
		{name: "Send encode failure then endpoint error delivers an error frame", call: send("\xff")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var callErr error
			w := newRecordingSSEResponse(false)
			handlerErrs := serveScriptedTick(t, w, true, func(ctx context.Context, stream clock.TickServerStream) error {
				callErr = tc.call(ctx, stream)
				return errors.New("endpoint failed")
			})

			require.Error(t, callErr)
			require.NotErrorIs(t, callErr, loomhttp.ErrSSEStreamClosed)
			require.Equal(t, []string{"error"}, w.frameKinds(t))
			require.Empty(t, handlerErrs)
		})
	}
}

func (c scriptedClock) Tick(ctx context.Context, _ *clock.TickPayload, stream clock.TickServerStream) error {
	return c.tick(ctx, stream)
}

func (scriptedClock) Tock(context.Context, *clock.TockPayload, clock.TockServerStream) error {
	return nil
}

func (w *recordingSSEResponse) Header() http.Header {
	return w.header
}

func (w *recordingSSEResponse) WriteHeader(int) {
}

func (w *recordingSSEResponse) Write(p []byte) (int, error) {
	if w.holdFirst != nil {
		first := false
		w.once.Do(func() {
			first = true
			close(w.firstWrite)
		})
		if first {
			<-w.holdFirst
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(p)
}

func (w *recordingSSEResponse) Flush() {
}

func newRecordingSSEResponse(holdFirst bool) *recordingSSEResponse {
	w := &recordingSSEResponse{header: make(http.Header)}
	if holdFirst {
		w.holdFirst = make(chan struct{})
		w.firstWrite = make(chan struct{})
	}
	return w
}

// serveScriptedTick serves one POST-initiated Tick SSE request through the
// generated JSON-RPC server and returns the errors reported to the server
// error handler.
func serveScriptedTick(t *testing.T, w http.ResponseWriter, hasID bool, tick func(context.Context, clock.TickServerStream) error) []error {
	t.Helper()

	var (
		mu          sync.Mutex
		handlerErrs []error
	)
	errhandler := func(_ context.Context, _ http.ResponseWriter, err error) {
		mu.Lock()
		defer mu.Unlock()
		handlerErrs = append(handlerErrs, err)
	}
	srv := clockjssvr.New(
		clock.NewEndpoints(scriptedClock{tick: tick}),
		loomhttp.NewMuxer(),
		loomhttp.RequestDecoder,
		loomhttp.ResponseEncoder,
		errhandler,
	)
	body := `{"jsonrpc":"2.0","method":"Tick","params":{}}`
	if hasID {
		body = `{"jsonrpc":"2.0","method":"Tick","params":{},"id":"req-1"}`
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/rpc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	srv.ServeHTTP(w, req)

	mu.Lock()
	defer mu.Unlock()
	return handlerErrs
}

func controlOf(stream clock.TickServerStream) loomhttp.SSEControl {
	control, ok := stream.(loomhttp.SSEControl)
	if !ok {
		panic("generated JSON-RPC SSE stream does not implement loomhttp.SSEControl")
	}
	return control
}

func send(value string) func(context.Context, clock.TickServerStream) error {
	return func(ctx context.Context, stream clock.TickServerStream) error {
		return stream.Send(ctx, tickResult(value))
	}
}

func sendAndClose(value string) func(context.Context, clock.TickServerStream) error {
	return func(ctx context.Context, stream clock.TickServerStream) error {
		return stream.SendAndClose(ctx, tickResult(value))
	}
}

func sendError() func(context.Context, clock.TickServerStream) error {
	return func(ctx context.Context, stream clock.TickServerStream) error {
		return stream.SendError(ctx, "req-1", errors.New("stream failed"))
	}
}

func tickResult(value string) *clock.TickResult {
	return &clock.TickResult{Value: stringPtr(value)}
}

func (w *recordingSSEResponse) waitFirstWrite(t *testing.T) {
	t.Helper()

	select {
	case <-w.firstWrite:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first SSE write")
	}
}

// frameKinds classifies every written SSE frame as comment, notification,
// result, or error.
func (w *recordingSSEResponse) frameKinds(t *testing.T) []string {
	t.Helper()

	w.mu.Lock()
	raw := w.body.String()
	w.mu.Unlock()

	var kinds []string
	for frame := range strings.SplitSeq(raw, "\n\n") {
		if strings.TrimSpace(frame) == "" {
			continue
		}
		if strings.HasPrefix(frame, ":") {
			kinds = append(kinds, "comment")
			continue
		}
		var data string
		for line := range strings.SplitSeq(frame, "\n") {
			if value, ok := strings.CutPrefix(line, "data: "); ok {
				data = value
			}
		}
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(data), &envelope), "frame %q", frame)
		switch {
		case envelope["method"] != nil:
			kinds = append(kinds, "notification")
		case envelope["error"] != nil:
			kinds = append(kinds, "error")
		case envelope["result"] != nil:
			kinds = append(kinds, "result")
		default:
			t.Fatalf("unclassified SSE frame %q", frame)
		}
	}
	return kinds
}
