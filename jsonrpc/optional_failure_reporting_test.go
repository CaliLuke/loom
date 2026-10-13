package jsonrpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionalFailureReportingPreservesArguments(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	writer := httptest.NewRecorder()
	cause := errors.New("original cause")
	require.NotPanics(t, func() {
		ReportFailure(ctx, writer, nil, cause)
	})
	calls := 0
	ReportFailure(ctx, writer, func(gotContext context.Context, gotWriter http.ResponseWriter, gotError error) {
		calls++
		require.Same(t, ctx, gotContext)
		require.Same(t, writer, gotWriter)
		require.Same(t, cause, gotError)
	}, cause)
	require.Equal(t, 1, calls)
	require.Empty(t, writer.Body.String())
}
