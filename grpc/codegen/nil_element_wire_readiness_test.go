package codegen

import (
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestNilElementWireReadiness(t *testing.T) {
	root := RunGRPCDSL(t, nilElementWireDSL)
	runWrapperContractModule(t, root, nilElementWireHarness)
}

func nilElementWireDSL() {
	leaf := Type("Leaf", func() {
		Field(1, "name", String)
	})
	var node expr.UserType
	node = Type("Node", func() {
		Field(1, "children", ArrayOf(node))
	})
	empty := Type("Empty", func() {
	})
	numbers := Type("Numbers", ArrayOf(Int))
	data := Type("Data", func() {
		Field(1, "leaves", ArrayOf(leaf))
		Field(2, "by_key", MapOf(String, leaf))
		Field(3, "trees", ArrayOf(node))
		Field(4, "nested", ArrayOf(ArrayOf(Int)))
		Field(5, "optional", leaf)
		Field(6, "empty_objects", ArrayOf(empty))
		Field(7, "named_nested", ArrayOf(numbers))
	})
	Service("nilwire", func() {
		Method("echo", func() {
			Payload(data)
			Result(data)
			GRPC(func() {
			})
		})
	})
}

const nilElementWireHarness = `package probe

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	svc "example.com/wrapperprobe/gen/nilwire"
	"example.com/wrapperprobe/gen/grpc/nilwire/client"
	pb "example.com/wrapperprobe/gen/grpc/nilwire/pb"
	"example.com/wrapperprobe/gen/grpc/nilwire/server"
)

func TestNativeNilMessageWireNormalization(t *testing.T) {
	for _, tc := range []struct {
		name string
		value *pb.EchoRequest
	}{
		{"array", &pb.EchoRequest{Leaves: []*pb.Leaf{nil}}},
		{"map", &pb.EchoRequest{ByKey: map[string]*pb.Leaf{"k": nil}}},
		{"recursive", &pb.EchoRequest{Trees: []*pb.Node{nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := proto.Marshal(tc.value)
			require.NoError(t, err)
			decoded := &pb.EchoRequest{}
			require.NoError(t, proto.Unmarshal(data, decoded))
			require.NoError(t, server.ValidateEchoRequest(decoded), "wire normalization erases nil into a valid empty message")
		})
	}
}

func TestRecursiveAndEmptyElementAdmission(t *testing.T) {
 for _, value := range []*svc.Data{
  {Trees: []*svc.Node{nil}},
  {Trees: []*svc.Node{{Children: []*svc.Node{nil}}}},
  {EmptyObjects: []*svc.Empty{nil}},
 } {
  request, err := client.NewProtoEchoRequest(value)
  require.Error(t, err)
  require.Nil(t, request)
  response, err := server.NewProtoEchoResponse(value)
  require.Error(t, err)
  require.Nil(t, response)
 }
}

func TestNativeNilReceiveConversion(t *testing.T) {
	for _, tc := range []struct {
		name string
		request *pb.EchoRequest
		response *pb.EchoResponse
	}{
		{"array", &pb.EchoRequest{Leaves: []*pb.Leaf{nil}}, &pb.EchoResponse{Leaves: []*pb.Leaf{nil}}},
		{"map", &pb.EchoRequest{ByKey: map[string]*pb.Leaf{"k": nil}}, &pb.EchoResponse{ByKey: map[string]*pb.Leaf{"k": nil}}},
        {"recursive", &pb.EchoRequest{Trees: []*pb.Node{nil}}, &pb.EchoResponse{Trees: []*pb.Node{nil}}},
        {"empty", &pb.EchoRequest{EmptyObjects: []*pb.Empty{nil}}, &pb.EchoResponse{EmptyObjects: []*pb.Empty{nil}}},
        {"wrapper", &pb.EchoRequest{Nested: []*pb.ArrayOfSint64{nil}}, &pb.EchoResponse{Nested: []*pb.ArrayOfSint64{nil}}},
        {"named wrapper", &pb.EchoRequest{NamedNested: []*pb.Numbers{nil}}, &pb.EchoResponse{NamedNested: []*pb.Numbers{nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("request", func(t *testing.T) {
				value, err := server.NewEchoPayload(tc.request)
                require.Error(t, err)
                require.Nil(t, value)
			})
			t.Run("response", func(t *testing.T) {
				value, err := client.NewEchoResult(tc.response)
                require.Error(t, err)
                require.Nil(t, value)
			})
		})
	}
}

func TestCollectionAndFieldControls(t *testing.T) {
	for _, tc := range []struct {
		name string
		value *svc.Data
	}{
		{"absent fields", &svc.Data{}},
		{"empty collection", &svc.Data{Leaves: []*svc.Leaf{}}},
		{"empty object element", &svc.Data{Leaves: []*svc.Leaf{{}}}},
		{"nil inner collection", &svc.Data{Nested: [][]int{nil}}},
		{"empty inner collection", &svc.Data{Nested: [][]int{{}}}},
        {"native zero inner collection", &svc.Data{Nested: [][]int{{0}}}},
        {"nil named inner collection", &svc.Data{NamedNested: []svc.Numbers{nil}}},
        {"empty named inner collection", &svc.Data{NamedNested: []svc.Numbers{{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := client.NewProtoEchoRequest(tc.value)
            require.NoError(t, err)
			require.NoError(t, server.ValidateEchoRequest(request))
			data, err := proto.Marshal(request)
			require.NoError(t, err)
			decoded := &pb.EchoRequest{}
			require.NoError(t, proto.Unmarshal(data, decoded))
			require.NoError(t, server.ValidateEchoRequest(decoded))
			if len(tc.value.Nested) > 0 {
                value, err := server.NewEchoPayload(decoded)
                require.NoError(t, err)
                require.NotNil(t, value.Nested[0])
                require.Equal(t, len(tc.value.Nested[0]), len(value.Nested[0]))
			}
		})
	}
}
`
