package grpc

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// NewProtoValue converts a Go value into its google.protobuf.Value form.
// It returns a descriptive error when the value is not representable by
// google.protobuf.Value.
func NewProtoValue(value any) (*structpb.Value, error) {
	converted, err := structpb.NewValue(value)
	if err != nil {
		return nil, fmt.Errorf("convert gRPC Any value: %w", err)
	}
	return converted, nil
}
