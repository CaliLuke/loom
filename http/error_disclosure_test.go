package http

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestProblemErrorDisclosure(t *testing.T) {
	secret := errors.New("private database password")
	public := loom.NewServiceError(secret, "public", false, false, true)
	public.Message = "public detail"
	safe := loom.WithErrorRemedy(loom.NewServiceError(secret, "public", false, false, true), &loom.ErrorRemedy{SafeMessage: "safe detail"})
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"plain", secret, "internal server error"},
		{"wrapped plain", fmt.Errorf("private wrapper: %w", secret), "internal server error"},
		{"public", public, "public detail"},
		{"wrapped public", fmt.Errorf("private wrapper: %w", public), "public detail"},
		{"safe", safe, "safe detail"},
		{"join", errors.Join(public, secret), "internal server error"},
		{"reverse join", errors.Join(secret, public), "internal server error"},
		{"safe branch", errors.Join(safe, secret), "internal server error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.err.Error()
			got := NewProblemResponse(context.Background(), tc.err, 0, "", "")
			require.Equal(t, tc.want, got.Detail)
			require.Equal(t, 500, got.Status)
			require.Equal(t, before, tc.err.Error())
		})
	}
	require.ErrorIs(t, public, secret)
}
