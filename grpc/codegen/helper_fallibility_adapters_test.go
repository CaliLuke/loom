package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
)

func TestHelperFallibilityAdapters(t *testing.T) {
	root := RunGRPCDSL(t, helperFallibilityAdaptersDSL)
	runWrapperContractModule(t, root, helperFallibilityAdaptersHarness)
}

func helperFallibilityAdaptersDSL() {
	leaf := Type("Leaf", func() {
		Field(1, "name", String)
	})
	nested := Type("Nested", func() {
		Field(1, "value", Any)
	})
	message := Type("Message", func() {
		Field(1, "leaf", leaf)
		Field(2, "nested", nested)
	})
	failure := Type("Failure", func() {
		ErrorName(1, "name", String)
		Field(2, "leaf", leaf)
		Field(3, "nested", nested)
		Required("name")
	})
	Service("adapters", func() {
		Method("plain", func() {
			Payload(leaf)
			Result(leaf)
			GRPC(func() {})
		})
		Method("convert", func() {
			Payload(message)
			Result(message)
			Error("failure", failure)
			GRPC(func() {
				Response("failure", CodeInternal)
			})
		})
		Method("stream", func() {
			StreamingPayload(message)
			StreamingResult(message)
			Error("failure", failure)
			GRPC(func() {
				Response("failure", CodeInternal)
			})
		})
	})
}

const helperFallibilityAdaptersHarness = `package probe

import (
 "context"
 "testing"

 "github.com/stretchr/testify/require"
 loom "github.com/CaliLuke/loom/pkg"
 "google.golang.org/grpc/status"
 "google.golang.org/protobuf/types/known/structpb"
 svc "example.com/wrapperprobe/gen/adapters"
 pb "example.com/wrapperprobe/gen/grpc/adapters/pb"
 "example.com/wrapperprobe/gen/grpc/adapters/client"
 "example.com/wrapperprobe/gen/grpc/adapters/server"
)

func TestStreamSendRejectsBeforeWrite(t *testing.T) {
 value := &svc.Message{Nested: &svc.Nested{Value: loom.JSONValue("{")}}
 // Supply the operation context explicitly: these intentionally uninitialized
 // wrappers have no captured invocation context or native transport. A write
 // would panic; an admission error must not stand in for conversion rejection.
 require.ErrorContains(t, new(client.StreamClientStream).SendWithContext(context.Background(), value), "decode gRPC Any JSON value")
 require.ErrorContains(t, new(server.StreamServerStream).SendWithContext(context.Background(), value), "decode gRPC Any JSON value")
}

func TestDeclaredErrorConversion(t *testing.T) {
 invalid := &svc.Failure{Name: "failure", Nested: &svc.Nested{Value: loom.JSONValue("{")}}
 unary, err := server.NewConvertFailureError(invalid)
 require.Error(t, err)
 require.Nil(t, unary)
 stream, err := server.NewStreamFailureError(invalid)
 require.Error(t, err)
 require.Nil(t, stream)
 native := &pb.Nested{Value: structpb.NewStringValue(string([]byte{0xff}))}
 decoded, err := client.NewConvertFailureError(&pb.ConvertFailureError{Name: new("failure"), Nested: native})
 require.Error(t, err)
 require.Nil(t, decoded)
 decoded, err = client.NewStreamFailureError(&pb.StreamFailureError{Name: new("failure"), Nested: native})
 require.Error(t, err)
 require.Nil(t, decoded)
 for _, raw := range []loom.JSONValue{nil, loom.JSONValue("null"), loom.JSONValue("false"), loom.JSONValue("{\"v\":[]}")} {
  valid := &svc.Failure{Name: "failure", Nested: &svc.Nested{Value: raw}}
  unary, err := server.NewConvertFailureError(valid)
  require.NoError(t, err)
  decoded, err := client.NewConvertFailureError(unary)
  require.NoError(t, err)
  require.Equal(t, valid, decoded)
  stream, err := server.NewStreamFailureError(valid)
  require.NoError(t, err)
  decoded, err = client.NewStreamFailureError(stream)
  require.NoError(t, err)
  require.Equal(t, valid, decoded)
 }
 // The generated declared-error mapper must not emit a partially converted detail.
 endpoint := func(context.Context, any) (any, error) {
  return nil, invalid
 }
 transport := server.New(&svc.Endpoints{Convert: endpoint}, nil, nil)
 response, err := transport.Convert(context.Background(), &pb.ConvertRequest{})
 require.Error(t, err)
 require.Nil(t, response)
 for _, detail := range status.Convert(err).Details() {
  _, malformed := detail.(*pb.ConvertFailureError)
  require.False(t, malformed)
 }
}
`
