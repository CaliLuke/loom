package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

const protoDefaultBytesHarness = `package probe

import (
 "testing"

 "github.com/stretchr/testify/require"
 "google.golang.org/protobuf/proto"

 "example.com/wrapperprobe/gen/bytevalues"
 "example.com/wrapperprobe/gen/grpc/bytevalues/client"
 pb "example.com/wrapperprobe/gen/grpc/bytevalues/pb"
 "example.com/wrapperprobe/gen/grpc/bytevalues/server"
)

func TestByteDefaults(t *testing.T) {
 for _, tc := range []struct {
  name string
  value []byte
  want []byte
 }{
  {"absent", nil, []byte("fallback")},
  {"explicit empty", []byte{}, []byte{}},
  {"ordinary", []byte("value"), []byte("value")},
 } {
  t.Run(tc.name, func(t *testing.T) {
   t.Run("request", func(t *testing.T) {
    message := client.NewProtoEchoRequest(&bytevalues.State{Data: tc.value})
    wire, err := proto.Marshal(message)
    require.NoError(t, err)
    decoded := &pb.EchoRequest{}
    require.NoError(t, proto.Unmarshal(wire, decoded))
    require.Equal(t, tc.want, server.NewEchoPayload(decoded).Data)
   })
   t.Run("response", func(t *testing.T) {
    message := server.NewProtoEchoResponse(&bytevalues.State{Data: tc.value})
    wire, err := proto.Marshal(message)
    require.NoError(t, err)
    decoded := &pb.EchoResponse{}
    require.NoError(t, proto.Unmarshal(wire, decoded))
    require.Equal(t, tc.want, client.NewEchoResult(decoded).Data)
   })
  })
 }
}
`

func TestProtoDefaultApplication(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typ      expr.DataType
		fallback any
	}{
		{"boolean", expr.Boolean, true},
		{"integer", expr.Int, 7},
		{"string", expr.String, "fallback"},
		{"bytes", expr.Bytes, []byte("fallback")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: tc.typ, DefaultValue: tc.fallback}}}}
			sd := &ServiceData{Name: "test", Scope: codegen.NewNameScope()}
			message := makeProtoBufMessage(expr.DupAtt(attribute), "Message", sd)
			service := serviceTypeContext("service", sd.Scope)
			protobuf := protoBufTypeContext("pb", sd.Scope, false)
			encoded, _, err := protoBufTransform(attribute, message, "source", "target", service, protobuf, true, true)
			require.NoError(t, err)
			require.NotContains(t, encoded, "== zero")
			decoded, _, err := protoBufTransform(message, attribute, "source", "target", protobuf, service, false, true)
			require.NoError(t, err)
			require.Contains(t, decoded, "source.Value == nil", "actual absent protobuf scalar should receive its selected default")
			require.NotContains(t, decoded, "== zero")
		})
	}
}

func TestProtoDefaultBytesGenerated(t *testing.T) {
	root := RunGRPCDSL(t, testdata.DefaultBytesDSL)
	runWrapperContractModule(t, root, protoDefaultBytesHarness)
}
