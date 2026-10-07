package loom

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLengthErrorDiagnostic(t *testing.T) {
	for _, tc := range []struct {
		name          string
		actual, bound int
		minimum       bool
		want          string
	}{
		{"minimum", 6, 8, true, "length of credential must be greater or equal than 8 but got length 6"},
		{"maximum", 1200000, 8, false, "length of credential must be lesser or equal than 8 but got length 1200000"},
		{"zero maximum", 1, 0, false, "length of credential must be lesser or equal than 0 but got length 1"},
		{"short collection", 1, 2, true, "length of credential must be greater or equal than 2 but got length 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := InvalidLengthError("credential", tc.actual, tc.bound, tc.minimum)
			require.EqualError(t, err, tc.want)
			var serviceErr *ServiceError
			require.ErrorAs(t, err, &serviceErr)
			require.Equal(t, InvalidLength, serviceErr.Name)
			require.NotNil(t, serviceErr.Field)
			require.Equal(t, "credential", *serviceErr.Field)
			require.False(t, serviceErr.Fault)
			require.False(t, serviceErr.Temporary)
			require.False(t, serviceErr.Timeout)
			require.Equal(t, "validation error", ErrorSafeMessage(err))
		})
	}
}
