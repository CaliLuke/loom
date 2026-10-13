package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	grpcnative "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestStreamInvokerOwnsOpeningContext(t *testing.T) {
	for _, failure := range []string{"open", "decode", "panic", "nil", "none"} {
		t.Run(failure, func(t *testing.T) {
			var opening context.Context
			want := errors.New("opening failure")
			invoker := NewStreamInvoker(func(ctx context.Context, _ any, _ ...grpcnative.CallOption) (any, error) {
				opening = ctx
				if failure == "open" {
					return nil, want
				}
				return "native stream", nil
			}, nil, func(ctx context.Context, _ any, _, _ metadata.MD) (any, error) {
				if failure == "decode" {
					return nil, want
				}
				if failure == "panic" {
					panic(want)
				}
				if failure == "nil" {
					return nil, nil
				}
				return ctx, nil
			})
			if failure == "panic" {
				require.PanicsWithValue(t, want, func() {
					_, err := invoker.Invoke(context.Background(), nil)
					require.NoError(t, err)
				})
				require.ErrorIs(t, opening.Err(), context.Canceled)
				return
			}
			value, err := invoker.Invoke(context.Background(), nil)
			if failure == "nil" {
				require.NoError(t, err)
				require.Nil(t, value)
				require.ErrorIs(t, opening.Err(), context.Canceled)
				return
			}
			if failure != "none" {
				require.ErrorIs(t, err, want)
				require.ErrorIs(t, opening.Err(), context.Canceled)
				return
			}
			require.NoError(t, err)
			require.NoError(t, opening.Err(), "response construction must transfer rather than close stream lifetime")
			ctx := value.(context.Context)
			owner := ctx.Value(streamContextKey{}).(*streamContext)
			owner.abort(context.Canceled)
			require.ErrorIs(t, opening.Err(), context.Canceled)
		})
	}
}
