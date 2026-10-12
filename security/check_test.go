package security

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCheckOriginalCancellation(t *testing.T) {
	type key struct{}
	original, cancel := context.WithCancel(context.Background())
	defer cancel()
	admitted, err := RunCheck(original, nil, func(context.Context, any) (context.Context, error) {
		return context.WithValue(context.Background(), key{}, "principal"), nil
	})
	require.NoError(t, err)
	require.Equal(t, "principal", admitted.Value(key{}))
	calls := 0
	check := func(ctx context.Context, _ any) (context.Context, error) {
		calls++
		return ctx, nil
	}
	_, err = RunCheck(admitted, nil, check)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	cancel()
	_, err = RunCheck(admitted, nil, check)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
}

func TestRunCheckErrorPrecedence(t *testing.T) {
	expected := errors.New("callback failure")
	for _, tc := range []struct {
		name        string
		callbackErr error
		want        error
	}{
		{"callback error precedes late cancellation", expected, expected},
		{"late cancellation rejects replaced context", nil, context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := RunCheck(ctx, nil, func(context.Context, any) (context.Context, error) {
				cancel()
				return context.Background(), tc.callbackErr
			})
			require.ErrorIs(t, err, tc.want)
		})
	}
}
