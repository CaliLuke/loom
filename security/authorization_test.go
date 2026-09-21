package security

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestProtectAuthorization(t *testing.T) {
	denied := errors.New("denied")
	cases := []struct {
		name       string
		middleware func(loom.Endpoint) loom.Endpoint
		request    string
		wantErr    error
		wantCalls  int
	}{
		{"allow", nil, "allowed", nil, 1},
		{"deny", nil, "denied", denied, 0},
		{"cache cannot bypass denial", func(_ loom.Endpoint) loom.Endpoint {
			return func(context.Context, any) (any, error) {
				return "cached", nil
			}
		}, "denied", denied, 0},
		{"mutation is checked again", func(next loom.Endpoint) loom.Endpoint {
			return func(ctx context.Context, _ any) (any, error) {
				return next(ctx, "denied")
			}
		}, "allowed", denied, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			check := func(ctx context.Context, req any) (context.Context, error) {
				if req != "allowed" {
					return nil, denied
				}
				return ctx, nil
			}
			ep := Protect(check, func(context.Context, any) (any, error) {
				calls++
				return "ok", nil
			})
			if tc.middleware != nil {
				ep = Protect(check, ep, tc.middleware)
			}
			_, err := ep(context.Background(), tc.request)
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.wantCalls, calls)
		})
	}
}

func TestAuthorizationDependencies(t *testing.T) {
	require.Panics(t, func() {
		RequireAuthorizer(nil)
	})
	var typedNil *struct {
	}
	require.Panics(t, func() {
		RequireAuthorizer(typedNil)
	})
	require.NotPanics(t, func() {
		RequireAuthorizer(&struct {
		}{})
	})
	require.Panics(t, func() {
		Protect(nil, nil)
	})
}

func TestAuthorizationCancellationAndInvalidContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		name  string
		ctx   context.Context
		check EndpointCheck
	}{
		{"already canceled", ctx, func(context.Context, any) (context.Context, error) {
			t.Error("check ran after cancellation")
			return context.Background(), nil
		}},
		{"canceled during check", context.Background(), func(context.Context, any) (context.Context, error) {
			return ctx, nil
		}},
		{"nil success context", context.Background(), func(context.Context, any) (context.Context, error) {
			return nil, nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ep := Protect(tc.check, func(context.Context, any) (any, error) {
				t.Error("service ran after failed check")
				return nil, nil
			})
			_, err := ep(tc.ctx, nil)
			require.Error(t, err)
		})
	}
}

func TestAuthorizationCannotDiscardCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ep := Protect(func(context.Context, any) (context.Context, error) {
		cancel()
		return context.Background(), nil
	}, func(context.Context, any) (any, error) {
		t.Error("service ran after the request was canceled")
		return nil, nil
	})
	_, err := ep(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)
}
