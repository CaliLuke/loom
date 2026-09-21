package security

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestAuthorizationMiddlewareContext(t *testing.T) {
	type traceKey struct{}
	cases := []struct {
		name      string
		cancel    bool
		derive    func(context.Context) context.Context
		wantError bool
	}{
		{"child", false, func(ctx context.Context) context.Context {
			return context.WithValue(ctx, traceKey{}, "trace")
		}, false},
		{"discarded", false, func(context.Context) context.Context {
			return context.Background()
		}, true},
		{"canceled and discarded", true, func(context.Context) context.Context {
			return context.Background()
		}, true},
		{"canceled and detached", true, context.WithoutCancel, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			factories := 0
			check := func(context.Context, any) (context.Context, error) {
				// Authentication is permitted to replace the context.
				return context.Background(), nil
			}
			middleware := func(next loom.Endpoint) loom.Endpoint {
				factories++
				return func(ctx context.Context, req any) (any, error) {
					if tc.cancel {
						cancel()
					}
					return next(tc.derive(ctx), req)
				}
			}
			ep := Protect(check, func(context.Context, any) (any, error) {
				calls++
				return nil, nil
			}, middleware, middleware)
			_, err := ep(ctx, nil)
			require.Equal(t, tc.wantError, err != nil)
			require.Equal(t, 2, factories)
			if tc.wantError {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
		})
	}
}

func TestAuthorizationMiddlewareConcurrentInvocations(t *testing.T) {
	type request struct {
		cancel context.CancelFunc
		deny   bool
	}
	var calls atomic.Int32
	factories := 0
	wrap := func(next loom.Endpoint) loom.Endpoint {
		factories++
		return func(ctx context.Context, req any) (any, error) {
			p := req.(*request)
			if p.deny {
				p.cancel()
			}
			return next(context.WithoutCancel(ctx), req)
		}
	}
	ep := Protect(func(context.Context, any) (context.Context, error) {
		return context.Background(), nil
	}, func(context.Context, any) (any, error) {
		calls.Add(1)
		return nil, nil
	}, wrap, wrap)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := &request{cancel: cancel, deny: i%2 == 0}
			_, err := ep(ctx, p)
			if p.deny != (err != nil) {
				t.Errorf("request %d: deny=%v, error=%v", i, p.deny, err)
			}
		})
	}
	wg.Wait()
	require.Equal(t, int32(10), calls.Load())
	require.Equal(t, 2, factories)
}
