package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProtoValueRejectsUnrepresentableValue(t *testing.T) {
	value, err := NewProtoValue(make(chan struct{}))

	require.Nil(t, value)
	require.ErrorContains(t, err, "convert gRPC Any value")
	require.ErrorContains(t, err, "invalid type")
}

func TestNewProtoValueAcceptsRepresentableValue(t *testing.T) {
	value, err := NewProtoValue(map[string]any{"name": "loom", "count": float64(2)})

	require.NoError(t, err)
	require.Equal(t, map[string]any{"name": "loom", "count": float64(2)}, value.AsInterface())
}
