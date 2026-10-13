package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestStreamOperationCanceledBeforeSend(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), false)
	defer owner.abort(context.Canceled)
	canceled, stop := context.WithCancel(ctx)
	stop()
	called := false
	err := SendStreamMessage(canceled, ctx, wrapperspb.String("value"), func(*wrapperspb.StringValue) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, called)
	require.NoError(t, SendStreamMessage(ctx, ctx, wrapperspb.String("value"), func(*wrapperspb.StringValue) error {
		return nil
	}))
}

func TestStreamOperationDetachedCanceledSend(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), false)
	defer owner.abort(context.Canceled)
	operation, cancel := context.WithCancel(ctx)
	entered := make(chan struct{})
	release := make(chan struct{})
	observed := make(chan string, 1)
	result := make(chan error, 1)
	message := wrapperspb.Bytes([]byte("original"))
	go func() {
		result <- SendStreamMessage(operation, ctx, message, func(value *wrapperspb.BytesValue) error {
			close(entered)
			<-release
			observed <- string(value.Value)
			return nil
		})
	}()
	<-entered
	cancel()
	require.ErrorIs(t, <-result, context.Canceled)
	copy(message.Value, "modified")
	close(release)
	require.Equal(t, "original", <-observed)
	require.ErrorIs(t, SendStreamMessage(context.Background(), ctx, message, func(*wrapperspb.BytesValue) error {
		t.Error("native I/O started after terminal cancellation")
		return nil
	}), context.Canceled)
}

func TestStreamOperationCompletedContextCannotCancelNext(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), false)
	defer owner.abort(context.Canceled)
	first, cancel := context.WithCancel(ctx)
	_, err := ReceiveStreamMessage(first, ctx, false, func() (*wrapperspb.StringValue, error) {
		return wrapperspb.String("first"), nil
	})
	require.NoError(t, err)
	cancel()
	value, err := ReceiveStreamMessage(ctx, ctx, false, func() (*wrapperspb.StringValue, error) {
		return wrapperspb.String("second"), nil
	})
	require.NoError(t, err)
	require.Equal(t, "second", value.Value)
}

func TestStreamOperationPanicRelays(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), false)
	defer owner.abort(context.Canceled)
	want := errors.New("native failure")
	require.PanicsWithValue(t, want, func() {
		_, err := ReceiveStreamMessage(ctx, ctx, false, func() (*wrapperspb.StringValue, error) {
			panic(want)
		})
		require.NoError(t, err)
	})
}
