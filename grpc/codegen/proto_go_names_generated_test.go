package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

const protoGoNamesHarness = `package nametest

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"%[1]s/gen/grpc/names/client"
	pb "%[1]s/gen/grpc/names/pb"
	"%[1]s/gen/grpc/names/server"
)

func TestNamesRoundTrip(t *testing.T) {
	t.Run("echo", func(t *testing.T) {
		checkNames(t, &pb.EchoRequest{}, &pb.EchoResponse{}, server.NewEchoPayload, client.NewProtoEchoRequest, client.NewEchoResult, server.NewProtoEchoResponse, server.ValidateEchoRequest, client.ValidateEchoResponse, client.BuildEchoPayload)
	})
	t.Run("reverse", func(t *testing.T) {
		checkNames(t, &pb.ReverseRequest{}, &pb.ReverseResponse{}, server.NewReversePayload, client.NewProtoReverseRequest, client.NewReverseResult, server.NewProtoReverseResponse, server.ValidateReverseRequest, client.ValidateReverseResponse, client.BuildReversePayload)
	})
	t.Run("choose", func(t *testing.T) {
		checkNames(t, &pb.ChooseRequest{}, &pb.ChooseResponse{}, server.NewChoosePayload, client.NewProtoChooseRequest, client.NewChooseResult, server.NewProtoChooseResponse, server.ValidateChooseRequest, client.ValidateChooseResponse, client.BuildChoosePayload)
	})
}

func checkNames[Q, R proto.Message, S any](t *testing.T, request Q, response R, decodeQ func(Q) S, encodeQ func(S) Q, decodeR func(R) S, encodeR func(S) R, validateQ func(Q) error, validateR func(R) error, cli func(string) (S, error)) {
	t.Helper()
	for _, branch := range []int{0, 1} {
		fill(request, branch)
		fill(response, branch)
		require.NoError(t, validateQ(request))
		require.NoError(t, validateR(response))
		require.True(t, proto.Equal(request, encodeQ(decodeQ(request))))
		require.True(t, proto.Equal(response, encodeR(decodeR(response))))
		data, err := protojson.Marshal(request)
		require.NoError(t, err)
		payload, err := cli(string(data))
		require.NoError(t, err)
		require.True(t, proto.Equal(request, encodeQ(payload)))
		checkInvalid(t, request, validateQ)
		checkInvalid(t, response, validateR)
	}
}

func fill(message proto.Message, branch int) {
	proto.Reset(message)
	m := message.ProtoReflect()
	fields := m.Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if oneof := field.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() && oneof.Fields().Get(branch).Number() != field.Number() {
			continue
		}
		if field.IsMap() {
			m.Mutable(field).Map().Set(protoreflect.ValueOfString("key").MapKey(), protoreflect.ValueOfString("ok"))
		} else {
			m.Set(field, protoreflect.ValueOfString("ok"))
		}
	}
}

func checkInvalid[M proto.Message](t *testing.T, message M, validate func(M) error) {
	t.Helper()
	m := message.ProtoReflect()
	fields := m.Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if field.IsMap() || !m.Has(field) {
			continue
		}
		m.Set(field, protoreflect.ValueOfString("x"))
		require.Error(t, validate(message), string(field.Name()))
		m.Set(field, protoreflect.ValueOfString("ok"))
	}
}
`

func TestGeneratedProtoGoNames(t *testing.T) {
	const module = "example.com/protogonames"
	root := RunGRPCDSL(t, testdata.ProtoGoFieldNamesDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, module, root, resolveGRPCLoomSource(t))
	for _, file := range ClientCLIFiles(module+"/gen", CreateGRPCServices(root)) {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}
	testDir := filepath.Join(dir, "internal", "nametest")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "names_test.go"), []byte(fmt.Sprintf(protoGoNamesHarness, module)), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/nametest")
}
