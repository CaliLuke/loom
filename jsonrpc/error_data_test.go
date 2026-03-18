package jsonrpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	goa "goa.design/goa/v3/pkg"
)

type (
	testNamedRemedyError struct{}
)

func (testNamedRemedyError) Error() string {
	return "named failure"
}

func (testNamedRemedyError) GoaErrorName() string {
	return "named_failure"
}

func (testNamedRemedyError) GoaErrorRemedy() *goa.ErrorRemedy {
	return &goa.ErrorRemedy{
		Code:        "named.fix",
		SafeMessage: "Named safe message.",
		RetryHint:   "Retry later.",
	}
}

func TestNewErrorData(t *testing.T) {
	t.Run("service error includes transport-neutral metadata", func(t *testing.T) {
		err := goa.WithErrorRemedy(
			goa.NewServiceError(errors.New("internal detail"), "bad_request", true, true, true),
			&goa.ErrorRemedy{
				Code:        "bad_request.fix",
				SafeMessage: "Retry with a valid request.",
				RetryHint:   "Correct the payload and retry.",
			},
		)

		data, ok := NewErrorData(err).(*ErrorData)
		require.True(t, ok)
		assert.Equal(t, "bad_request", data.Name)
		assert.Equal(t, err.ID, data.ID)
		assert.True(t, data.Temporary)
		assert.True(t, data.Timeout)
		assert.True(t, data.Fault)
		require.NotNil(t, data.Remedy)
		assert.Equal(t, "bad_request.fix", data.Remedy.Code)
	})

	t.Run("named remedy error keeps name and remedy without service fields", func(t *testing.T) {
		data, ok := NewErrorData(testNamedRemedyError{}).(*ErrorData)
		require.True(t, ok)
		assert.Equal(t, "named_failure", data.Name)
		assert.Empty(t, data.ID)
		assert.False(t, data.Temporary)
		assert.False(t, data.Timeout)
		assert.False(t, data.Fault)
		require.NotNil(t, data.Remedy)
		assert.Equal(t, "named.fix", data.Remedy.Code)
	})

	t.Run("plain errors return nil", func(t *testing.T) {
		assert.Nil(t, NewErrorData(errors.New("plain failure")))
		assert.Nil(t, NewErrorData(nil))
	})
}
