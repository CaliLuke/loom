package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/CaliLuke/loom/security"
)

func TestAdmitStreamContextAndMetadata(t *testing.T) {
	type principal struct{}
	ctx, cancel := context.WithCancel(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "secret")))
	defer cancel()
	admitted, err := AdmitStream(ctx, func(_ context.Context, frame any, md metadata.MD) (any, error) {
		require.Nil(t, frame)
		return md.Get("authorization")[0], nil
	}, func(_ context.Context, payload any) (context.Context, error) {
		require.Equal(t, "secret", payload)
		return metadata.NewIncomingContext(context.WithValue(context.Background(), principal{}, "alice"), metadata.Pairs("authorization", "replacement")), nil
	})
	require.NoError(t, err)
	require.Equal(t, "alice", admitted.Value(principal{}))
	md, _ := metadata.FromIncomingContext(admitted)
	require.Equal(t, []string{"secret"}, md.Get("authorization"))
	cancel()
	_, err = security.RunCheck(admitted, nil, func(ctx context.Context, _ any) (context.Context, error) {
		t.Error("complete check ran after original cancellation")
		return ctx, nil
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestStreamAdmissionPreventsDecode(t *testing.T) {
	denied := loom.PermanentError("denied", "not admitted")
	err := ServeStream(context.Background(), StreamServerSpec{
		Admit: func(context.Context) (context.Context, error) {
			return nil, denied
		},
		Decode: func(context.Context) (any, error) {
			t.Error("receive/decode before admission")
			return nil, nil
		},
		Handle: func(context.Context, any) error {
			t.Error("handler after denied admission")
			return nil
		},
	})
	require.Error(t, err)
}

func TestAdmitStreamRejectsInvalidCallbackContext(t *testing.T) {
	for _, tc := range []struct {
		name  string
		check security.EndpointCheck
	}{
		{"nil", func(context.Context, any) (context.Context, error) {
			return nil, nil
		}},
		{"error", func(context.Context, any) (context.Context, error) {
			return nil, errors.New("denied")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := AdmitStream(context.Background(), func(context.Context, any, metadata.MD) (any, error) {
				return "secret", nil
			}, tc.check)
			require.Error(t, err)
		})
	}
}

func TestStreamAdmissionRetainsTransportObservation(t *testing.T) {
	var observer *loomtransport.RequestObserver
	err := ServeStream(context.Background(), StreamServerSpec{
		Service: "uploads", Method: "write",
		Admit: func(ctx context.Context) (context.Context, error) {
			observer = loomtransport.RequestObserverFromContext(ctx)
			return context.Background(), nil
		},
		Decode: func(ctx context.Context) (any, error) {
			require.Same(t, observer, loomtransport.RequestObserverFromContext(ctx))
			require.Equal(t, "uploads", ctx.Value(loom.ServiceKey))
			require.Equal(t, "write", ctx.Value(loom.MethodKey))
			return "complete", nil
		},
		Handle: func(ctx context.Context, input any) error {
			require.Same(t, observer, loomtransport.RequestObserverFromContext(ctx))
			require.Equal(t, "complete", input)
			return nil
		},
	})
	require.NoError(t, err)
}
