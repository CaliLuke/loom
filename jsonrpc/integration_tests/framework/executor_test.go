package framework

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractExpectedErrorObject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		err         error
		wantCode    float64
		wantMessage string
	}{
		{
			name:        "http transport error",
			err:         errors.New(`unexpected status 400: {"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid request"},"id":"req-1"}`),
			wantCode:    -32600,
			wantMessage: "Invalid request",
		},
		{
			name:        "cli wrapped error",
			err:         errors.New("CLI command failed: exit status 1\nStderr: request failed\n{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32602,\"message\":\"Invalid params\"},\"id\":\"req-2\"}\nStdout: "),
			wantCode:    -32602,
			wantMessage: "Invalid params",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errObj, ok := extractExpectedErrorObject(tc.err)

			require.True(t, ok)
			require.Equal(t, tc.wantCode, errObj["code"])
			require.Equal(t, tc.wantMessage, errObj["message"])
		})
	}
}

func TestExtractExpectedErrorObjectFailsWithoutJSONRPCPayload(t *testing.T) {
	t.Parallel()

	errObj, ok := extractExpectedErrorObject(errors.New("request failed: connection refused"))

	require.False(t, ok)
	require.Nil(t, errObj)
}
