package admission_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/admission"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestRateAdmissionRejectsBackendFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision admission.RateDecision
		err      error
	}{
		{"backend error", admission.RateDecision{}, errors.New("sensitive key and backend details")},
		{"allow with error", admission.RateDecision{Allowed: true}, errors.New("sensitive failure")},
		{"negative retry", admission.RateDecision{RetryAfter: -time.Second}, nil},
		{"allow with retry", admission.RateDecision{Allowed: true, RetryAfter: time.Second}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
				return tc.decision, tc.err
			}, time.Second)
			require.NoError(t, err)
			decision, err := control.Check(context.Background(), "sensitive-key")
			require.False(t, decision.Allowed)
			requireRateUnavailable(t, err)
			require.NotContains(t, err.Error(), "sensitive")
		})
	}
}

func TestRateAdmissionDeadlineAndContext(t *testing.T) {
	for _, parentDeadline := range []time.Duration{0, 50 * time.Millisecond} {
		synctest.Test(t, func(t *testing.T) {
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "trace")
			if parentDeadline > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, parentDeadline)
				defer cancel()
			}
			start := time.Now()
			control, err := admission.NewRateAdmission(func(check context.Context, key string) (admission.RateDecision, error) {
				require.Equal(t, "group", key)
				require.Equal(t, "trace", check.Value(contextKey{}))
				<-check.Done()
				// Even a late successful answer cannot authorize execution.
				return admission.RateDecision{Allowed: true}, nil
			}, 100*time.Millisecond)
			require.NoError(t, err)
			decision, err := control.Check(ctx, "group")
			require.False(t, decision.Allowed)
			if parentDeadline > 0 {
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Equal(t, parentDeadline, time.Since(start))
			} else {
				requireRateUnavailable(t, err)
				require.Equal(t, 100*time.Millisecond, time.Since(start))
			}
		})
	}
}

func TestRateAdmissionMiddlewarePreservesContextAndGroups(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		first, err := admission.NewRate(admission.RatePolicy{PerSecond: 1, Burst: 1})
		require.NoError(t, err)
		second, err := admission.NewRate(admission.RatePolicy{PerSecond: 1, Burst: 1})
		require.NoError(t, err)
		groups := map[string]*admission.Rate{"first": first, "second": second}
		control, err := admission.NewRateAdmission(func(ctx context.Context, key string) (admission.RateDecision, error) {
			return groups[key].Allow(ctx)
		}, time.Second)
		require.NoError(t, err)
		denial := errors.New("declared quota error")
		wrap := control.Middleware(func(_ context.Context, request any) (string, error) {
			return request.(string), nil
		}, func(decision admission.RateDecision) error {
			require.Equal(t, time.Second, decision.RetryAfter)
			return denial
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		ep := wrap(func(got context.Context, request any) (any, error) {
			require.Same(t, ctx, got, "backend deadline must not replace the service context")
			require.NoError(t, got.Err())
			calls++
			return request, nil
		})
		result, err := ep(ctx, "first")
		require.NoError(t, err)
		require.Equal(t, "first", result)
		_, err = ep(ctx, "first")
		require.ErrorIs(t, err, denial)
		_, err = ep(ctx, "second")
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		cancel()
		_, err = ep(ctx, "first")
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 2, calls)
	})
}

func TestRateAdmissionMiddlewareFailsClosed(t *testing.T) {
	for _, failure := range []string{"grouping", "backend", "nil denial", "canceled"} {
		t.Run(failure, func(t *testing.T) {
			checks := 0
			control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
				checks++
				if failure == "backend" {
					return admission.RateDecision{}, errors.New("sensitive backend")
				}
				return admission.RateDecision{}, nil
			}, time.Second)
			require.NoError(t, err)
			wrap := control.Middleware(func(context.Context, any) (string, error) {
				if failure == "grouping" {
					return "", errors.New("sensitive identity")
				}
				return "group", nil
			}, func(admission.RateDecision) error {
				return nil
			})
			calls := 0
			ep := wrap(func(context.Context, any) (any, error) {
				calls++
				return nil, nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if failure == "canceled" {
				cancel()
			}
			_, err = ep(ctx, nil)
			if failure == "canceled" {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				requireRateUnavailable(t, err)
			}
			require.Zero(t, calls)
			if failure == "grouping" || failure == "canceled" {
				require.Zero(t, checks)
			}
		})
	}
}

func TestRateAdmissionCanceledCheck(t *testing.T) {
	checks := 0
	control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
		checks++
		return admission.RateDecision{Allowed: true}, nil
	}, time.Second)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	decision, err := control.Check(ctx, "group")
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, decision.Allowed)
	require.Zero(t, checks)
}

func TestRateAdmissionConfiguration(t *testing.T) {
	check := func(context.Context, string) (admission.RateDecision, error) {
		return admission.RateDecision{}, nil
	}
	for _, timeout := range []time.Duration{0, -time.Second} {
		_, err := admission.NewRateAdmission(check, timeout)
		require.Error(t, err)
	}
	_, err := admission.NewRateAdmission(nil, time.Second)
	require.Error(t, err)
	var zero admission.RateAdmission
	_, err = zero.Check(context.Background(), "")
	requireRateUnavailable(t, err)
	control, err := admission.NewRateAdmission(check, time.Second)
	require.NoError(t, err)
	require.Panics(t, func() {
		control.Middleware(nil, nil)
	})
	key := func(context.Context, any) (string, error) {
		return "", nil
	}
	denied := func(admission.RateDecision) error {
		return errors.New("quota exhausted")
	}
	require.Panics(t, func() {
		zero.Middleware(key, denied)
	})
	require.Panics(t, func() {
		control.Middleware(key, denied)(nil)
	})
}

func requireRateUnavailable(t *testing.T, err error) {
	t.Helper()
	var serviceError *loom.ServiceError
	require.ErrorAs(t, err, &serviceError)
	require.Equal(t, "rate_limit_unavailable", serviceError.Name)
}
