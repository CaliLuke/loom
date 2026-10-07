package loom

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeErrorsOwnership(t *testing.T) {
	first := ownershipError("first")
	second := ownershipError("second")
	merged := mergedServiceError(t, first, second)
	require.NotSame(t, first, merged)
	require.Equal(t, "first", first.Message)
	require.Equal(t, "second", second.Message)
	require.Equal(t, "first; second", merged.Message)
	require.ErrorIs(t, merged, first)
	require.ErrorIs(t, merged, second)

	*merged.Field = "changed"
	merged.Remedy.SafeMessage = "changed"
	require.Equal(t, "first", *first.Field)
	require.Equal(t, "first", first.Remedy.SafeMessage)
	*second.Field = "changed"
	second.Remedy.SafeMessage = "changed"
	history := merged.History()
	require.Equal(t, "second", *history[1].Field)
	require.Equal(t, "second", history[1].Remedy.SafeMessage)

	next := mergedServiceError(t, merged, first)
	require.Equal(t, "first; second", merged.Message)
	require.Equal(t, "first; second; first", next.Message)
	require.Equal(t, []string{"first", "second", "first"}, historyMessages(next.History()))
}

func TestServiceErrorHistorySnapshots(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func() *ServiceError
	}{
		{"single", func() *ServiceError {
			return ownershipError("first")
		}},
		{"merged", func() *ServiceError {
			return mergedServiceError(t, ownershipError("first"), ownershipError("second"))
		}},
		{"attached", func() *ServiceError {
			return WithErrorHistory(ownershipError("top"), ownershipError("first"))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.make()
			before := err.Message
			history := err.History()
			history[0].Message = "changed"
			*history[0].Field = "changed"
			history[0].Remedy.SafeMessage = "changed"
			history[0] = ownershipError("replacement")
			fresh := err.History()
			require.Equal(t, "first", fresh[0].Message)
			require.Equal(t, "first", *fresh[0].Field)
			require.Equal(t, "first", fresh[0].Remedy.SafeMessage)
			require.Equal(t, before, err.Message)
		})
	}
}

func TestWithErrorHistorySnapshotsContributions(t *testing.T) {
	first := ownershipError("first")
	merged := mergedServiceError(t, first, ownershipError("second"))
	top := WithErrorHistory(ownershipError("top"), nil, merged, first)
	first.Message = "changed"
	*merged.Field = "changed"
	merged.Remedy.SafeMessage = "changed"
	history := top.History()
	require.Equal(t, []string{"first", "second", "first"}, historyMessages(history))
	require.Equal(t, "first", *history[0].Field)
	require.Equal(t, "first", history[0].Remedy.SafeMessage)
	require.Len(t, history[0].History(), 1)
}

func TestMergeErrorsNilIdentity(t *testing.T) {
	plain := errors.New("plain")
	service := ownershipError("service")
	require.Nil(t, MergeErrors(nil, nil))
	for _, err := range []error{plain, service} {
		require.Same(t, err, MergeErrors(nil, err))
		require.Same(t, err, MergeErrors(err, nil))
	}
}

func TestMergeErrorsPreservesWrappedCauses(t *testing.T) {
	first := ownershipError("first")
	wrapped := fmt.Errorf("context: %w", first)
	second := errors.New("second")
	merged := MergeErrors(wrapped, second)
	require.ErrorIs(t, merged, wrapped)
	require.ErrorIs(t, merged, first)
	require.ErrorIs(t, merged, second)
}

func ownershipError(message string) *ServiceError {
	field := message
	err := NewServiceError(errors.New(message), message, true, true, false)
	err.Field = &field
	err.Remedy = &ErrorRemedy{Code: message, SafeMessage: message, RetryHint: message}
	return err
}

func historyMessages(history []*ServiceError) []string {
	messages := make([]string, len(history))
	for i, err := range history {
		messages[i] = err.Message
	}
	return messages
}

func mergedServiceError(t *testing.T, first, second error) *ServiceError {
	t.Helper()
	merged := MergeErrors(first, second)
	require.IsType(t, &ServiceError{}, merged)
	var service *ServiceError
	require.ErrorAs(t, merged, &service)
	return service
}
