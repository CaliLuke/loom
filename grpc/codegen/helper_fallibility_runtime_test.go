package codegen

import "testing"

func TestHelperFallibilityRuntime(t *testing.T) {
	root := RunGRPCDSL(t, helperFallibilityDSL("nested Any", false))
	runWrapperContractModule(t, root, helperFallibilityRuntimeHarness)
}

const helperFallibilityRuntimeHarness = `package probe

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	loom "github.com/CaliLuke/loom/pkg"
	svc "example.com/wrapperprobe/gen/mixed"
	"example.com/wrapperprobe/gen/grpc/mixed/client"
	pb "example.com/wrapperprobe/gen/grpc/mixed/pb"
	"example.com/wrapperprobe/gen/grpc/mixed/server"
)

func TestNestedHelperErrors(t *testing.T) {
	value := &svc.Fallible{Nested: &svc.Nested{Value: loom.JSONValue("{")}}
	request, err := client.NewProtoSecondRequest(value)
	require.Error(t, err)
	require.Nil(t, request)
	response, err := server.NewProtoSecondResponse(value)
	require.Error(t, err)
	require.Nil(t, response)
	var md metadata.MD
	encodedRequest, err := client.EncodeSecondRequest(context.Background(), value, &md)
	require.Error(t, err)
	require.Nil(t, encodedRequest)
	encodedResponse, err := server.EncodeSecondResponse(context.Background(), value, &md, &md)
	require.Error(t, err)
	require.Nil(t, encodedResponse)
	invalid := &pb.Nested{Value: structpb.NewStringValue(string([]byte{0xff}))}
	payload, err := server.NewSecondPayload(&pb.SecondRequest{Nested: invalid})
	require.Error(t, err)
	require.Nil(t, payload)
	result, err := client.NewSecondResult(&pb.SecondResponse{Nested: invalid})
	require.Error(t, err)
	require.Nil(t, result)
}

func TestNestedHelperValues(t *testing.T) {
	name := "kept"
	for _, raw := range []loom.JSONValue{nil, loom.JSONValue("null"), loom.JSONValue("false"), loom.JSONValue("0"), loom.JSONValue("{\"k\":[]}")} {
		value := &svc.Fallible{Leaf: &svc.Leaf{Name: &name}, Nested: &svc.Nested{Value: raw}}
		request, err := client.NewProtoSecondRequest(value)
		require.NoError(t, err)
		wire, err := proto.Marshal(request)
		require.NoError(t, err)
		decoded := &pb.SecondRequest{}
		require.NoError(t, proto.Unmarshal(wire, decoded))
		got, err := server.NewSecondPayload(decoded)
		require.NoError(t, err)
		require.Equal(t, value, got)
		response, err := server.NewProtoSecondResponse(value)
		require.NoError(t, err)
		wire, err = proto.Marshal(response)
		require.NoError(t, err)
		decodedResponse := &pb.SecondResponse{}
		require.NoError(t, proto.Unmarshal(wire, decodedResponse))
		got, err = client.NewSecondResult(decodedResponse)
		require.NoError(t, err)
		require.Equal(t, value, got)
	}
	plain := &svc.Plain{Leaf: &svc.Leaf{Name: &name}}
	require.Equal(t, plain, server.NewFirstPayload(client.NewProtoFirstRequest(plain)))
}
`
