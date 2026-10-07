package grpc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestErrorHistoryOwnershipRoundTrip(t *testing.T) {
	first := loom.MissingFieldError("name", "body")
	second := loom.MissingFieldError("age", "body")
	merged := loom.MergeErrors(loom.MergeErrors(first, second), first)
	response := NewErrorResponse(merged)
	wire, err := proto.Marshal(response)
	require.NoError(t, err)
	var decoded loompb.ErrorResponse
	require.NoError(t, proto.Unmarshal(wire, &decoded))
	got := NewServiceError(&decoded)
	history := got.History()
	require.Len(t, history, 3)
	require.Equal(t, first.Error(), history[0].Message)
	require.Equal(t, second.Error(), history[1].Message)
	require.Equal(t, first.Error(), history[2].Message)
	require.Equal(t, "name", *history[0].Field)
	history[0].Message = "changed"
	*history[0].Field = "changed"
	decoded.History[1].Msg = "changed"
	fresh := got.History()
	require.Equal(t, first.Error(), fresh[0].Message)
	require.Equal(t, "name", *fresh[0].Field)
	require.Equal(t, second.Error(), fresh[1].Message)
}
