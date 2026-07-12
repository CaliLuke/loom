package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestInvokerInvoke(t *testing.T) {
	var (
		errEncode = errors.New("encode failed")
		errDecode = errors.New("decode failed")
		errRemote = status.Error(codes.Unavailable, "remote failed")
	)

	cases := []struct {
		name    string
		encoder RequestEncoder
		decoder ResponseDecoder
		fn      RemoteFunc
		req     any
		wantRes any
		wantErr error
	}{
		{
			name: "happy path",
			encoder: func(_ context.Context, v any, _ *metadata.MD) (any, error) {
				assert.Equal(t, "loom-request", v)
				return "pb-request", nil
			},
			decoder: func(_ context.Context, pb any, _, _ metadata.MD) (any, error) {
				assert.Equal(t, "pb-response", pb)
				return "loom-response", nil
			},
			fn: func(_ context.Context, reqpb any, _ ...grpc.CallOption) (any, error) {
				assert.Equal(t, "pb-request", reqpb)
				return "pb-response", nil
			},
			req:     "loom-request",
			wantRes: "loom-response",
		},
		{
			// With a nil decoder the remote response is dropped and the
			// invoker returns nil. This documents current behavior.
			name: "nil encoder and decoder",
			fn: func(_ context.Context, reqpb any, _ ...grpc.CallOption) (any, error) {
				assert.Nil(t, reqpb)
				return "pb-response", nil
			},
			req:     "loom-request",
			wantRes: nil,
		},
		{
			name: "encoder error",
			encoder: func(_ context.Context, _ any, _ *metadata.MD) (any, error) {
				return nil, errEncode
			},
			fn: func(_ context.Context, _ any, _ ...grpc.CallOption) (any, error) {
				t.Errorf("remote function must not be called when encoding fails")
				return nil, nil
			},
			req:     "loom-request",
			wantErr: errEncode,
		},
		{
			name: "remote error propagates unchanged",
			encoder: func(_ context.Context, _ any, _ *metadata.MD) (any, error) {
				return "pb-request", nil
			},
			decoder: func(_ context.Context, _ any, _, _ metadata.MD) (any, error) {
				t.Errorf("decoder must not be called when the remote call fails")
				return nil, nil
			},
			fn: func(_ context.Context, _ any, _ ...grpc.CallOption) (any, error) {
				return nil, errRemote
			},
			req:     "loom-request",
			wantErr: errRemote,
		},
		{
			name: "decoder error",
			encoder: func(_ context.Context, _ any, _ *metadata.MD) (any, error) {
				return "pb-request", nil
			},
			decoder: func(_ context.Context, _ any, _, _ metadata.MD) (any, error) {
				return nil, errDecode
			},
			fn: func(_ context.Context, _ any, _ ...grpc.CallOption) (any, error) {
				return "pb-response", nil
			},
			req:     "loom-request",
			wantErr: errDecode,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			invoker := NewInvoker(c.fn, c.encoder, c.decoder)
			res, err := invoker.Invoke(context.Background(), c.req)
			if c.wantErr != nil {
				require.ErrorIs(t, err, c.wantErr)
				assert.Nil(t, res)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.wantRes, res)
		})
	}
}

func TestInvokerInvokeMetadataFlow(t *testing.T) {
	// The encoder must see metadata already present in the outgoing context,
	// the remote function must see metadata added by the encoder, and the
	// decoder must receive the header and trailer metadata populated by the
	// remote call.
	encoder := func(_ context.Context, v any, md *metadata.MD) (any, error) {
		require.NotNil(t, md)
		assert.Equal(t, []string{"preset-value"}, md.Get("preset-key"), "encoder must see preexisting outgoing metadata")
		md.Set("encoded-key", "encoded-value")
		return "pb-request", nil
	}
	fn := func(ctx context.Context, _ any, opts ...grpc.CallOption) (any, error) {
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok, "remote function must receive outgoing metadata")
		assert.Equal(t, []string{"preset-value"}, md.Get("preset-key"))
		assert.Equal(t, []string{"encoded-value"}, md.Get("encoded-key"))
		for _, opt := range opts {
			switch o := opt.(type) {
			case grpc.HeaderCallOption:
				*o.HeaderAddr = metadata.Pairs("resp-header", "header-value")
			case grpc.TrailerCallOption:
				*o.TrailerAddr = metadata.Pairs("resp-trailer", "trailer-value")
			}
		}
		return "pb-response", nil
	}
	decoder := func(_ context.Context, pb any, hdr, trlr metadata.MD) (any, error) {
		assert.Equal(t, "pb-response", pb)
		assert.Equal(t, []string{"header-value"}, hdr.Get("resp-header"), "decoder must receive response header metadata")
		assert.Equal(t, []string{"trailer-value"}, trlr.Get("resp-trailer"), "decoder must receive response trailer metadata")
		return "loom-response", nil
	}

	invoker := NewInvoker(fn, encoder, decoder)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("preset-key", "preset-value"))
	res, err := invoker.Invoke(ctx, "loom-request")
	require.NoError(t, err)
	assert.Equal(t, "loom-response", res)
}
