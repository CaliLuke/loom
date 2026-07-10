package http

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebSocketStreamWithContextPreservesSuccessfulOperation(t *testing.T) {
	stream := NewWebSocketStream(nil)
	ctx, cancel := context.WithCancel(context.Background())

	err := stream.withContext(ctx, func() error {
		cancel()
		return nil
	})

	require.NoError(t, err)
}

func TestWebSocketStreamWithContextUsesCancellationForFailedOperation(t *testing.T) {
	stream := NewWebSocketStream(nil)
	ctx, cancel := context.WithCancel(context.Background())
	operationErr := errors.New("operation failed")

	err := stream.withContext(ctx, func() error {
		cancel()
		return operationErr
	})

	require.ErrorIs(t, err, context.Canceled)
}
