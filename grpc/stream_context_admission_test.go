package grpc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type (
	cancelAfterReadContext struct {
		context.Context
		cancel context.CancelFunc
		once   sync.Once
	}
	countedCallbackContext struct {
		context.Context
		registered atomic.Int32
	}
)

func TestStreamReceiveCancellationBeforeAdmissionKeepsClient(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), true)
	defer owner.abort(context.Canceled)
	operation, cancel := context.WithCancel(context.Background())
	defer cancel()
	staged := &cancelAfterReadContext{Context: operation, cancel: cancel}
	_, err := ReceiveStreamMessage(staged, ctx, false, func() (*wrapperspb.StringValue, error) {
		t.Error("native receive ran after cancellation before admission")
		return nil, nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, ctx.Err(), "an operation rejected before admission cannot terminate the client")
	value, err := ReceiveStreamMessage(ctx, ctx, false, func() (*wrapperspb.StringValue, error) {
		return wrapperspb.String("next"), nil
	})
	require.NoError(t, err)
	require.Equal(t, "next", value.Value)
}

func TestStreamSendCancellationDetachesParentCallback(t *testing.T) {
	base, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()
	parent := &countedCallbackContext{Context: base}
	ctx, owner := newStreamContext(parent, true)
	defer owner.abort(context.Canceled)
	require.Positive(t, parent.registered.Load())
	operation, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- SendStreamMessage(operation, ctx, wrapperspb.String("value"), func(*wrapperspb.StringValue) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	cancel()
	err := <-result
	close(release)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, parent.registered.Load(), "terminal send must release registered parent callbacks")
}

func TestStreamTerminalRegistrationArbitration(t *testing.T) {
	for _, order := range []string{"register-first", "terminate-first"} {
		t.Run(order, func(t *testing.T) {
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			parent := &countedCallbackContext{Context: base}
			_, owner := newStreamContext(context.Background(), true)
			if order == "register-first" {
				owner.watch(parent)
				require.EqualValues(t, 1, parent.registered.Load())
				owner.abort(context.Canceled)
			} else {
				owner.abort(context.Canceled)
				owner.watch(parent)
			}
			require.Zero(t, parent.registered.Load())
		})
	}
}

func (c *cancelAfterReadContext) Err() error {
	err := c.Context.Err()
	c.once.Do(c.cancel)
	return err
}

func (c *countedCallbackContext) Value(any) any {
	// Force context's documented AfterFunc delegation rather than its private
	// cancelCtx fast path, so the test can observe callback registration.
	return nil
}

func (c *countedCallbackContext) AfterFunc(f func()) func() bool {
	c.registered.Add(1)
	var once sync.Once
	release := func() {
		c.registered.Add(-1)
	}
	stop := context.AfterFunc(c.Context, func() {
		once.Do(release)
		f()
	})
	return func() bool {
		stopped := stop()
		if stopped {
			once.Do(release)
		}
		return stopped
	}
}
