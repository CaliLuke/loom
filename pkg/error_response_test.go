package loom

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	responseNamedWrapper struct {
		cause error
	}
	responseRemedyWrapper struct {
		cause  error
		remedy *ErrorRemedy
	}
	responseAsWrapper struct {
		cause error
		calls *int
	}
)

func TestResolveErrorResponseOwnership(t *testing.T) {
	branch := WithErrorRemedy(TemporaryError("branch", "public branch"), &ErrorRemedy{Code: "branch action", SafeMessage: "branch safe"})
	other := errors.New("independent")
	outer := responseNamedWrapper{cause: errors.Join(branch, other)}
	safe := &ErrorRemedy{Code: "outer action", SafeMessage: "outer safe"}
	for _, tc := range []struct {
		name    string
		err     error
		owner   error
		remedy  *ErrorRemedy
		message string
	}{
		{"nil", nil, nil, nil, ""},
		{"direct", branch, branch, branch.Remedy, "branch safe"},
		{"wrapped", fmt.Errorf("diagnostic: %w", branch), branch, branch.Remedy, "branch safe"},
		{"single join", errors.Join(nil, branch), branch, branch.Remedy, "branch safe"},
		{"independent join", errors.Join(branch, other), nil, nil, "internal server error"},
		{"reversed join", errors.Join(other, branch), nil, nil, "internal server error"},
		{"named outer", outer, outer, nil, "outer message"},
		{"safe wrapper", responseRemedyWrapper{branch, safe}, branch, safe, "outer safe"},
		{"safe joined wrapper", responseRemedyWrapper{errors.Join(branch, other), safe}, nil, safe, "outer safe"},
		{"nil remedy shadows metadata", responseRemedyWrapper{branch, nil}, branch, nil, "branch safe"},
		{"empty remedy shadows metadata", responseRemedyWrapper{branch, &ErrorRemedy{}}, branch, &ErrorRemedy{}, "branch safe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveErrorResponse(tc.err)
			require.Equal(t, tc.owner, got.Owner)
			require.Equal(t, tc.remedy, got.Remedy)
			require.Equal(t, tc.message, ErrorSafeMessage(tc.err))
			if tc.owner == nil {
				require.Empty(t, got.Name)
			}
		})
	}
	calls := 0
	got := ResolveErrorResponse(responseAsWrapper{cause: errors.Join(branch, other), calls: &calls})
	require.Nil(t, got.Owner)
	require.Nil(t, got.Remedy)
	require.Zero(t, calls, "custom As cannot impersonate wire ownership")
	// Standalone diagnostic extraction still walks the full tree.
	require.Same(t, branch.Remedy, ExtractErrorRemedy(errors.Join(other, branch)))
}

func (e responseNamedWrapper) Error() string {
	return "outer message"
}

func (e responseNamedWrapper) LoomErrorName() string {
	return "outer"
}

func (e responseNamedWrapper) Unwrap() error {
	return e.cause
}

func (e responseRemedyWrapper) Error() string {
	return "diagnostic wrapper"
}

func (e responseRemedyWrapper) LoomErrorRemedy() *ErrorRemedy {
	return e.remedy
}

func (e responseRemedyWrapper) Unwrap() error {
	return e.cause
}

func (e responseAsWrapper) Error() string {
	return "diagnostic wrapper"
}

func (e responseAsWrapper) As(target any) bool {
	*e.calls++
	if target, ok := target.(*LoomErrorNamer); ok {
		*target = responseNamedWrapper{}
		return true
	}
	return false
}

func (e responseAsWrapper) Unwrap() error {
	return e.cause
}
