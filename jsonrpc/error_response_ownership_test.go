package jsonrpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

type responseOwnerError struct {
	cause error
}

func TestErrorDataCompleteErrorOwnership(t *testing.T) {
	branch := loom.WithErrorRemedy(loom.TemporaryError("busy", "branch detail"), &loom.ErrorRemedy{Code: "branch action", RetryHint: "branch retry"})
	branch.ID = "branch-id"
	other := errors.New("independent cleanup")
	for _, tc := range []struct {
		name  string
		err   error
		owned bool
	}{
		{"direct", branch, true},
		{"wrapped", fmt.Errorf("diagnostic: %w", branch), true},
		{"single join", errors.Join(nil, branch), true},
		{"join", errors.Join(branch, other), false},
		{"reverse", errors.Join(other, branch), false},
		{"wrapped join", fmt.Errorf("diagnostic: %w", errors.Join(branch, other)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NewErrorData(tc.err)
			if tc.owned {
				require.Equal(t, &ErrorData{Name: "busy", ID: branch.ID, Temporary: true, Remedy: branch.Remedy}, got)
				return
			}
			require.Nil(t, got, "independent branch metadata does not describe the complete error")
		})
	}
	for _, tc := range []struct {
		name  string
		cause error
	}{
		{"outer single cause", branch},
		{"outer join", errors.Join(branch, other)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NewErrorData(responseOwnerError{cause: tc.cause})
			require.Equal(t, &ErrorData{Name: "outer"}, got, "named outer owner must not borrow inner ServiceError metadata")
		})
	}
}

func TestEnvelopeDecodeErrorUsesCompleteOwner(t *testing.T) {
	branch := &loom.ServiceError{Name: loom.RequestBodyTooLarge, Message: "oversize"}
	for _, tc := range []struct {
		name  string
		err   error
		owned bool
	}{
		{"direct", branch, true},
		{"wrapped", fmt.Errorf("wrapped: %w", branch), true},
		{"single join", errors.Join(branch), true},
		{"independent join", errors.Join(branch, errors.New("other")), false},
		{"outer owner", responseOwnerError{cause: branch}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, message, data := envelopeDecodeError(tc.err)
			if tc.owned {
				require.Equal(t, CodeForServiceError(branch), code)
				require.Equal(t, loom.ErrorSafeMessage(tc.err), message)
				require.NotNil(t, data)
			} else {
				require.Equal(t, ParseError, code)
				require.Equal(t, "Parse error", message)
				require.Nil(t, data)
			}
		})
	}
}

func (e responseOwnerError) Error() string {
	return "outer public detail"
}

func (e responseOwnerError) LoomErrorName() string {
	return "outer"
}

func (e responseOwnerError) Unwrap() error {
	return e.cause
}
