package grpc

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	loom "github.com/CaliLuke/loom/pkg"
)

// NewProtoValue converts a Go value into its google.protobuf.Value form.
// It returns a descriptive error when the value is not representable by
// google.protobuf.Value.
func NewProtoValue(value any) (*structpb.Value, error) {
	if raw, ok := value.(jsontext.Value); ok {
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, fmt.Errorf("decode gRPC Any JSON value: %w", err)
		}
		value = decoded
	}
	converted, err := structpb.NewValue(value)
	if err != nil {
		return nil, fmt.Errorf("convert gRPC Any value: %w", err)
	}
	return converted, nil
}

// NewJSONValue converts a google.protobuf.Value into unparsed JSON.
func NewJSONValue(value *structpb.Value) (loom.JSONValue, error) {
	if value == nil {
		return nil, nil
	}
	converted, err := json.Marshal(value.AsInterface())
	if err != nil {
		return nil, fmt.Errorf("encode gRPC Any JSON value: %w", err)
	}
	return converted, nil
}
