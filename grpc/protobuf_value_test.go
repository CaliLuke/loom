package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
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

func TestJSONValueProtoRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "direct Any", raw: "true"},
		{name: "array Any", raw: `["value",2,null]`},
		{name: "map Any", raw: `{"value":2,"enabled":true}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			protobuf, err := NewProtoValue(loom.JSONValue(test.raw))
			require.NoError(t, err)
			converted, err := NewJSONValue(protobuf)
			require.NoError(t, err)
			require.JSONEq(t, test.raw, string(converted))
		})
	}
}
