package grpc

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestStreamHalfCloseAfterTerminalReceive(t *testing.T) {
	for _, tc := range []struct {
		name  string
		final bool
		err   error
	}{
		{name: "eof", err: io.EOF},
		{name: "remote-status", err: status.Error(codes.InvalidArgument, "rejected")},
		{name: "final-success", final: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, owner := newStreamContext(context.Background(), true)
			defer owner.abort(context.Canceled)
			_, err := ReceiveStreamMessage(ctx, ctx, tc.final, func() (*wrapperspb.StringValue, error) {
				return wrapperspb.String("value"), tc.err
			})
			require.ErrorIs(t, err, tc.err)
			require.ErrorIs(t, ctx.Err(), context.Canceled)
			for range 2 {
				err := CloseStreamSend(ctx, ctx, func() error {
					t.Error("terminal half-close performed native I/O")
					return nil
				})
				require.NoError(t, err, "receive owns the terminal status; half-close is already satisfied")
			}
		})
	}
}

func TestStreamHalfCloseRacesTerminalRelease(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), true)
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	defer close(release)
	go func() {
		done <- CloseStreamSend(ctx, ctx, func() error {
			close(started)
			<-release
			return nil
		})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("native half-close did not start")
	}
	owner.abort(io.EOF)
	select {
	case err := <-done:
		require.NoError(t, err, "terminal release satisfies an unfinished half-close")
	case <-time.After(time.Second):
		t.Fatal("half-close did not return after terminal release")
	}
	require.ErrorIs(t, owner.cause(), io.EOF, "cleanup must not replace terminal status")
}

func TestStreamHalfClosePreservesLiveAdmission(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), true)
	defer owner.abort(context.Canceled)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	err := CloseStreamSend(canceled, ctx, func() error {
		t.Error("pre-canceled live half-close performed native I/O")
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, owner.cause())
	want := errors.New("custom native half-close error")
	err = CloseStreamSend(ctx, ctx, func() error {
		return want
	})
	require.ErrorIs(t, err, want)
	require.NoError(t, ctx.Err(), "half-close does not end reception")
	owner.abort(io.EOF)
	require.ErrorIs(t, err, want, "later terminal release cannot replace a published native result")
	require.NoError(t, CloseStreamSend(ctx, ctx, func() error {
		t.Error("cleanup after terminal release performed I/O")
		return nil
	}))
}

func TestStreamHalfCloseSharesSendLease(t *testing.T) {
	ctx, owner := newStreamContext(context.Background(), true)
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- CloseStreamSend(ctx, ctx, func() error {
			close(started)
			<-release
			return nil
		})
	}()
	t.Cleanup(func() {
		owner.abort(context.Canceled)
		close(release)
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("native half-close did not start")
	}
	err := SendStreamMessage(ctx, ctx, wrapperspb.String("overlap"), func(*wrapperspb.StringValue) error {
		t.Error("competing send reached native I/O")
		return nil
	})
	require.ErrorContains(t, err, "concurrent gRPC stream operations in the same direction")
	require.NoError(t, ctx.Err(), "rejected overlap leaves the owner alive")
	owner.abort(io.EOF)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("terminal cleanup did not return")
	}
	owner.mu.Lock()
	occupied := owner.active[0] != nil
	owner.mu.Unlock()
	require.True(t, occupied, "late native I/O retains its slot until it unwinds")
	require.NoError(t, CloseStreamSend(ctx, ctx, func() error {
		t.Error("repeated terminal half-close started a second worker")
		return nil
	}))
}

func TestStreamHalfCloseAfterOriginalContextCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	ctx, owner := newStreamContext(parent, true)
	defer owner.abort(context.Canceled)
	cancel()
	for range 2 {
		require.NoError(t, CloseStreamSend(ctx, ctx, func() error {
			t.Error("canceled RPC has no send direction left to close")
			return nil
		}))
	}
}
