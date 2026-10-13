package grpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

type streamLogWriter struct {
	entries chan string
}

func (w *streamLogWriter) Write(data []byte) (int, error) {
	w.entries <- string(data)
	return len(data), nil
}

func TestStreamOwnerBoundsBothDirections(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), false)
	defer owner.abort(context.Canceled)
	operation, cancel := context.WithCancel(ctx)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		results <- SendStreamMessage(operation, ctx, wrapperspb.String("send"), func(*wrapperspb.StringValue) error {
			entered <- struct{}{}
			<-release
			return nil
		})
	}()
	go func() {
		_, err := ReceiveStreamMessage(context.Background(), ctx, false, func() (*wrapperspb.StringValue, error) {
			entered <- struct{}{}
			<-release
			return wrapperspb.String("private result"), nil
		})
		results <- err
	}()
	<-entered
	<-entered
	err := SendStreamMessage(ctx, ctx, wrapperspb.String("second send"), func(*wrapperspb.StringValue) error {
		t.Error("second native send started while the direction is occupied")
		return nil
	})
	require.ErrorContains(t, err, "same direction")
	cancel()
	for range 2 {
		require.ErrorIs(t, <-results, context.Canceled)
	}
	owner.abort(context.DeadlineExceeded)
	require.ErrorIs(t, owner.cause(), context.Canceled)
	close(release)
	require.Eventually(t, func() bool {
		owner.mu.Lock()
		defer owner.mu.Unlock()
		return owner.active[0] == nil && owner.active[1] == nil
	}, time.Second, time.Millisecond)
}

func TestStreamCanceledReceiveHidesWorkerResult(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), false)
	defer owner.abort(context.Canceled)
	operation, cancel := context.WithCancel(ctx)
	entered := make(chan struct{})
	release := make(chan struct{})
	workerReturned := make(chan struct{})
	result := make(chan *wrapperspb.BytesValue, 1)
	errors := make(chan error, 1)
	go func() {
		value, err := ReceiveStreamMessage(operation, ctx, false, func() (*wrapperspb.BytesValue, error) {
			value := wrapperspb.Bytes(make([]byte, 16))
			close(entered)
			<-release
			copy(value.Value, "native mutation")
			close(workerReturned)
			return value, nil
		})
		result <- value
		errors <- err
	}()
	<-entered
	cancel()
	require.ErrorIs(t, <-errors, context.Canceled)
	require.Nil(t, <-result)
	close(release)
	<-workerReturned
}

func TestStreamLateNativePanicDiagnostic(t *testing.T) {
	writer := &streamLogWriter{entries: make(chan string, 1)}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(writer, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})
	ctx, owner := newStreamContext(context.Background(), false)
	defer owner.abort(context.Canceled)
	operation, cancel := context.WithCancel(ctx)
	entered := make(chan struct{})
	release := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := ReceiveStreamMessage(operation, ctx, false, func() (*wrapperspb.StringValue, error) {
			close(entered)
			<-release
			panic(errors.New("secret-native-panic-value"))
		})
		result <- err
	}()
	<-entered
	cancel()
	require.ErrorIs(t, <-result, context.Canceled)
	close(release)
	select {
	case entry := <-writer.entries:
		require.Contains(t, entry, "native stream operation panicked after cancellation")
		require.Contains(t, entry, "panic_type=")
		require.NotContains(t, entry, "secret-native-panic-value")
	case <-time.After(time.Second):
		t.Fatal("late native panic diagnostic missing")
	}
}

func TestStreamClientHalfCloseAndFinalRelease(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), true)
	defer owner.abort(context.Canceled)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err := ReceiveStreamMessage(canceled, ctx, false, func() (*wrapperspb.StringValue, error) {
		t.Error("pre-canceled receive performed native I/O")
		return nil, nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, owner.cause(), "pre-canceled receive must leave a healthy stream usable")
	require.NoError(t, CloseStreamSend(ctx, ctx, func() error {
		return nil
	}))
	require.NoError(t, ctx.Err(), "CloseSend must preserve the opening context")
	value, err := ReceiveStreamMessage(ctx, ctx, true, func() (*wrapperspb.StringValue, error) {
		return wrapperspb.String("final"), nil
	})
	require.NoError(t, err)
	require.Equal(t, "final", value.Value)
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.ErrorIs(t, owner.cause(), io.EOF)
}

func TestStreamOriginalCancellationSurvivesAdmission(t *testing.T) {
	original, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := ServeStream(original, StreamServerSpec{
		Admit: func(context.Context) (context.Context, error) {
			return context.Background(), nil
		},
		Decode: func(ctx context.Context) (any, error) {
			return ctx, nil
		},
		Handle: func(ctx context.Context, _ any) error {
			cancel()
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
				t.Error("authentication context replacement discarded original cancellation")
			}
			return nil
		},
	})
	require.Equal(t, codes.Canceled, status.Code(err))
}

func TestStreamCanceledOperationCannotFinishCleanly(t *testing.T) {
	recorder := &serverEventRecorder{}
	ctx := loomtransport.WithObserver(context.Background(), recorder)
	release := make(chan struct{})
	var owner *streamContext
	err := ServeStream(ctx, StreamServerSpec{
		Decode: func(ctx context.Context) (any, error) {
			owner = ctx.Value(streamContextKey{}).(*streamContext)
			return nil, nil
		},
		Handle: func(ctx context.Context, _ any) error {
			operation, cancel := context.WithCancel(ctx)
			defer cancel()
			_, err := ReceiveStreamMessage(operation, ctx, false, func() (*wrapperspb.StringValue, error) {
				cancel()
				<-release
				return nil, io.EOF
			})
			require.ErrorIs(t, ObserveStreamDecodeError(ctx, err), context.Canceled)
			return nil
		},
	})
	require.Equal(t, codes.Canceled, status.Code(err))
	events := recorder.snapshot()
	require.Len(t, events, 3)
	require.Equal(t, loomtransport.EventKindRequestFailure, events[2].Kind)
	require.Equal(t, loomtransport.ReasonRequestDecodeFailed, events[2].Reason)
	close(release)
	require.Eventually(t, func() bool {
		owner.mu.Lock()
		defer owner.mu.Unlock()
		return owner.active[1] == nil
	}, time.Second, time.Millisecond)
	require.Equal(t, events, recorder.snapshot(), "native unwind must not mutate completed observation")
}

func TestStreamFinalReceiveReservesBothDirections(t *testing.T) {
	for _, occupied := range []string{"send", "receive"} {
		t.Run(occupied, func(t *testing.T) {
			ctx, owner := newStreamContext(context.Background(), true)
			defer owner.abort(context.Canceled)
			entered, release := make(chan struct{}), make(chan struct{})
			result := make(chan error, 1)
			go func() {
				if occupied == "send" {
					result <- SendStreamMessage(ctx, ctx, wrapperspb.String("value"), func(*wrapperspb.StringValue) error {
						close(entered)
						<-release
						return nil
					})
					return
				}
				_, err := ReceiveStreamMessage(ctx, ctx, false, func() (*wrapperspb.StringValue, error) {
					close(entered)
					<-release
					return wrapperspb.String("value"), nil
				})
				result <- err
			}()
			<-entered
			_, err := ReceiveStreamMessage(ctx, ctx, true, func() (*wrapperspb.StringValue, error) {
				t.Error("final receive started while one native direction was occupied")
				return wrapperspb.String("value"), nil
			})
			close(release)
			require.NoError(t, <-result)
			require.ErrorContains(t, err, "same direction")
			require.NoError(t, ctx.Err(), "rejected operation must not terminate the client")
		})
	}
}
