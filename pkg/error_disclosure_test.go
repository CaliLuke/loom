package loom

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type disclosureNamedError struct{}

func (disclosureNamedError) Error() string {
	return "designed public detail"
}

func (disclosureNamedError) LoomErrorName() string {
	return "designed"
}

func TestErrorDisclosure(t *testing.T) {
	secret := errors.New("private database password")
	public := NewServiceError(secret, "denied", false, false, false)
	public.Message = "public denial"
	sameText := WithErrorRemedy(Fault("internal server error"), &ErrorRemedy{SafeMessage: "internal server error"})
	safe := WithErrorRemedy(NewServiceError(secret, "fault", false, false, true), &ErrorRemedy{SafeMessage: "safe detail"})
	outer := NewServiceError(errors.Join(public, secret), "aggregate", false, false, true)
	outer.Message = "public aggregate"
	for _, tc := range []struct {
		name string
		err  error
		want string
		safe bool
	}{
		{"nil", nil, "", false},
		{"plain", secret, "internal server error", false},
		{"wrapped plain", fmt.Errorf("private query: %w", secret), "internal server error", false},
		{"service", public, "public denial", false},
		{"wrapped service", fmt.Errorf("private query: %w", public), "public denial", false},
		{"generated", disclosureNamedError{}, "designed public detail", false},
		{"wrapped generated", fmt.Errorf("private query: %w", disclosureNamedError{}), "designed public detail", false},
		{"safe service", safe, "safe detail", true},
		{"safe custom", testRemediableError{}, "safe detail", true},
		{"safe text equals fallback", sameText, "internal server error", true},
		{"named text equals fallback", Fault("internal server error"), "internal server error", false},
		{"single join", errors.Join(nil, public), "public denial", false},
		{"independent join", errors.Join(public, secret), "internal server error", false},
		{"reordered join", errors.Join(secret, public), "internal server error", false},
		{"safe branch is not owner", errors.Join(safe, secret), "internal server error", false},
		{"outer join owner", outer, "public aggregate", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before string
			if tc.err != nil {
				before = tc.err.Error()
			}
			message, safeOverride := ResolveErrorMessage(tc.err)
			require.Equal(t, tc.want, message)
			require.Equal(t, tc.safe, safeOverride)
			require.Equal(t, tc.want, ErrorSafeMessage(tc.err))
			if tc.err != nil {
				require.Equal(t, before, tc.err.Error())
			}
		})
	}
	require.ErrorIs(t, public, secret)
	require.ErrorIs(t, safe, secret)
	require.Equal(t, "private database password", safe.Message)
}
