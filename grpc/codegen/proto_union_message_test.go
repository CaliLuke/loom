package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

var unionMessageRoundTripHarness = fmt.Sprintf(`package roundtrip

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pickunion "%[1]s/gen/pickunion"
	"%[1]s/gen/grpc/pickunion/client"
	pb "%[1]s/gen/grpc/pickunion/pb"
	"%[1]s/gen/grpc/pickunion/server"
)

type service struct{}

func (service) Echo(_ context.Context, p *pickunion.LeafOrOther) (*pickunion.LeafOrOther, error) {
	return p, nil
}

func (service) Named(_ context.Context, p *pickunion.Choice) (*pickunion.Choice, error) {
	return p, nil
}

func (service) Watch(_ context.Context, p *pickunion.LeafOrOther, stream pickunion.WatchServerStream) error {
	for range 2 {
		if err := stream.Send(p); err != nil {
			return err
		}
	}
	return stream.Close()
}

func (service) Upload(_ context.Context, stream pickunion.UploadServerStream) error {
	var last *pickunion.LeafOrOther
	for {
		p, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(last)
		}
		if err != nil {
			return err
		}
		last = p
	}
}

func (service) Relay(_ context.Context, stream pickunion.RelayServerStream) error {
	for {
		p, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.Close()
		}
		if err != nil {
			return err
		}
		if err := stream.Send(p); err != nil {
			return err
		}
	}
}

func dial(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterPickunionServer(srv, server.New(pickunion.NewEndpoints(service{}), nil, nil))
	go func() {
		if err := srv.Serve(listener); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return conn
}

func branches() map[string]*pickunion.LeafOrOther {
	name := "leaf"
	count := 42
	leaf := &pickunion.LeafOrOther{}
	leaf.SetLeaf(&pickunion.Leaf{Name: &name})
	other := &pickunion.LeafOrOther{}
	other.SetOther(&pickunion.Other{Count: &count})
	return map[string]*pickunion.LeafOrOther{"leaf": leaf, "other": other}
}

func TestRoundTrip(t *testing.T) {
	c := client.NewClient(dial(t))
	ctx := context.Background()
	for name, want := range branches() {
		t.Run("unary/"+name, func(t *testing.T) {
			got, err := c.Echo()(ctx, want)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
		t.Run("named/"+name, func(t *testing.T) {
			choice := &pickunion.Choice{}
			if leaf, ok := want.AsLeaf(); ok {
				choice.SetLeaf(leaf)
			}
			if other, ok := want.AsOther(); ok {
				choice.SetOther(other)
			}
			got, err := c.Named()(ctx, choice)
			require.NoError(t, err)
			require.Equal(t, choice, got)
		})
		t.Run("server stream/"+name, func(t *testing.T) {
			res, err := c.Watch()(ctx, want)
			require.NoError(t, err)
			stream := res.(pickunion.WatchClientStream)
			for range 2 {
				got, err := stream.Recv()
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
			_, err = stream.Recv()
			require.ErrorIs(t, err, io.EOF)
		})
		t.Run("client stream/"+name, func(t *testing.T) {
			res, err := c.Upload()(ctx, nil)
			require.NoError(t, err)
			stream := res.(pickunion.UploadClientStream)
			for _, p := range []*pickunion.LeafOrOther{branches()["other"], want} {
				require.NoError(t, stream.Send(p))
			}
			got, err := stream.CloseAndRecv()
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
		t.Run("bidirectional stream/"+name, func(t *testing.T) {
			res, err := c.Relay()(ctx, nil)
			require.NoError(t, err)
			stream := res.(pickunion.RelayClientStream)
			require.NoError(t, stream.Send(want))
			got, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, want, got)
			require.NoError(t, stream.Close())
			_, err = stream.Recv()
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

func TestUnsetBranchRejected(t *testing.T) {
	c := pb.NewPickunionClient(dial(t))
	ctx := context.Background()
	cases := map[string]func() error{
		"unary": func() error {
			_, err := c.Echo(ctx, &pb.EchoRequest{})
			return err
		},
		"named": func() error {
			_, err := c.Named(ctx, &pb.Choice{})
			return err
		},
		"server stream": func() error {
			stream, err := c.Watch(ctx, &pb.WatchRequest{})
			if err != nil {
				return err
			}
			_, err = stream.Recv()
			return err
		},
		"client stream": func() error {
			stream, err := c.Upload(ctx)
			if err != nil {
				return err
			}
			if err := stream.Send(&pb.UploadStreamingRequest{}); err != nil {
				return err
			}
			_, err = stream.CloseAndRecv()
			return err
		},
		"bidirectional stream": func() error {
			stream, err := c.Relay(ctx)
			if err != nil {
				return err
			}
			if err := stream.Send(&pb.RelayStreamingRequest{}); err != nil {
				return err
			}
			_, err = stream.Recv()
			return err
		},
	}
	for name, call := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, codes.InvalidArgument, status.Code(call()))
		})
	}
}
`, "example.com/grpcunionmessage")

var unionMessageBranchNameRoundTripHarness = fmt.Sprintf(`package roundtrip

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	branchname "%[1]s/gen/branchname"
	"%[1]s/gen/grpc/branchname/client"
	pb "%[1]s/gen/grpc/branchname/pb"
	"%[1]s/gen/grpc/branchname/server"
)

type service struct{}

func (service) Echo(_ context.Context, p *branchname.FieldOrOther) (*branchname.FieldOrOther, error) {
	return p, nil
}

func (service) Clash(_ context.Context, p *branchname.FieldOrFieldOneof) (*branchname.FieldOrFieldOneof, error) {
	return p, nil
}

func (service) Relay(_ context.Context, stream branchname.RelayServerStream) error {
	for {
		p, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.Close()
		}
		if err != nil {
			return err
		}
		if err := stream.Send(p); err != nil {
			return err
		}
	}
}

func dial(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterBranchnameServer(srv, server.New(branchname.NewEndpoints(service{}), nil, nil))
	go func() {
		if err := srv.Serve(listener); err != nil {
			t.Error(err)
		}
	}()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return conn
}

func TestRoundTrip(t *testing.T) {
	c := client.NewClient(dial(t))
	ctx := context.Background()
	name := "field"
	count := 42
	flag := true
	field := &branchname.FieldOrOther{}
	field.SetField(&branchname.Field{Name: &name})
	other := &branchname.FieldOrOther{}
	other.SetOther(&branchname.Other{Flag: &flag})
	for label, want := range map[string]*branchname.FieldOrOther{"field": field, "other": other} {
		t.Run("unary/"+label, func(t *testing.T) {
			got, err := c.Echo()(ctx, want)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
		t.Run("bidirectional stream/"+label, func(t *testing.T) {
			res, err := c.Relay()(ctx, nil)
			require.NoError(t, err)
			stream := res.(branchname.RelayClientStream)
			require.NoError(t, stream.Send(want))
			got, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, want, got)
			require.NoError(t, stream.Close())
			_, err = stream.Recv()
			require.ErrorIs(t, err, io.EOF)
		})
	}
	clashField := &branchname.FieldOrFieldOneof{}
	clashField.SetField(&branchname.Field{Name: &name})
	clashOneof := &branchname.FieldOrFieldOneof{}
	clashOneof.SetFieldOneof(&branchname.FieldOneof{Count: &count})
	for label, want := range map[string]*branchname.FieldOrFieldOneof{"field": clashField, "field oneof": clashOneof} {
		t.Run("clash/"+label, func(t *testing.T) {
			got, err := c.Clash()(ctx, want)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestUnsetBranchRejected(t *testing.T) {
	c := pb.NewBranchnameClient(dial(t))
	ctx := context.Background()
	_, err := c.Echo(ctx, &pb.EchoRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = c.Clash(ctx, &pb.ClashRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	stream, err := c.Relay(ctx)
	require.NoError(t, err)
	require.NoError(t, stream.Send(&pb.RelayStreamingRequest{}))
	_, err = stream.Recv()
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
`, "example.com/grpcunionbranchname")

// TestProtoFilesUnionMessage checks that a union used directly as the
// payload or result of a method, unary or streaming, anonymous or named with
// Type, generates a message that wraps the union in a oneof named "field"
// whose branches are numbered consecutively from 1, and that protoc accepts
// the result.
func TestProtoFilesUnionMessage(t *testing.T) {
	const oneof = " {\n\toneof field {\n\t\tLeaf leaf = 1;\n\t\tOther other = 2;\n\t}\n}"
	code := protoFileCode(t, testdata.UnionMessageDSL)

	cases := []struct {
		name    string
		rpc     string
		message string
	}{
		{"unary payload", "rpc Echo (EchoRequest) returns (EchoResponse);", "EchoRequest"},
		{"unary result", "rpc Echo (EchoRequest) returns (EchoResponse);", "EchoResponse"},
		{"named payload and result", "rpc Named (Choice) returns (Choice);", "Choice"},
		{"payload of server stream", "rpc Watch (WatchRequest) returns (stream WatchResponse);", "WatchRequest"},
		{"server streaming result", "rpc Watch (WatchRequest) returns (stream WatchResponse);", "WatchResponse"},
		{"client streaming payload", "rpc Upload (stream UploadStreamingRequest) returns (UploadResponse);", "UploadStreamingRequest"},
		{"result of client stream", "rpc Upload (stream UploadStreamingRequest) returns (UploadResponse);", "UploadResponse"},
		{"bidirectional streaming payload", "rpc Relay (stream RelayStreamingRequest) returns (stream RelayResponse);", "RelayStreamingRequest"},
		{"bidirectional streaming result", "rpc Relay (stream RelayStreamingRequest) returns (stream RelayResponse);", "RelayResponse"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Contains(t, code, c.rpc)
			assert.Contains(t, code, "message "+c.message+oneof)
		})
	}
	assert.NotContains(t, code, "= 0;")
	assert.NotContains(t, code, "leaf_or_other")
	testutil.AssertString(t, "testdata/golden/proto_protofiles-union-message.proto.golden", code)
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestGeneratedUnionMessageRoundTrip compiles and vets a generated module
// whose methods use unions directly as payloads and results in every unary
// and streaming position. It serves the generated server over an in-memory
// gRPC connection and sends each union branch through the generated client,
// and checks that the server rejects a request message with no branch set.
func TestGeneratedUnionMessageRoundTrip(t *testing.T) {
	const modulePath = "example.com/grpcunionmessage"
	root := RunGRPCDSL(t, testdata.UnionMessageDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(unionMessageRoundTripHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

// TestProtoFilesUnionMessageBranchName checks that the oneof of a message
// that wraps a union takes the name "field" followed by "_oneof" as many
// times as needed to differ from the names of the union branches, and that
// protoc accepts the result.
func TestProtoFilesUnionMessageBranchName(t *testing.T) {
	code := protoFileCode(t, testdata.UnionMessageBranchNameDSL)

	cases := []struct {
		name    string
		message string
	}{
		{"branch named field", "message EchoRequest {\n\toneof field_oneof {\n\t\tField field = 1;\n\t\tOther other = 2;\n\t}\n}"},
		{"branches named field and field_oneof", "message ClashRequest {\n\toneof field_oneof_oneof {\n\t\tField field = 1;\n\t\tFieldOneof field_oneof = 2;\n\t}\n}"},
		{"streaming payload", "message RelayStreamingRequest {\n\toneof field_oneof {\n\t\tField field = 1;\n\t\tOther other = 2;\n\t}\n}"},
		{"streaming result", "message RelayResponse {\n\toneof field_oneof {\n\t\tField field = 1;\n\t\tOther other = 2;\n\t}\n}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Contains(t, code, c.message)
		})
	}
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestGeneratedUnionMessageBranchNameRoundTrip compiles and vets a generated
// module whose union payloads and results have branches named after the
// oneof of the wrapping message, and round-trips each branch over an
// in-memory gRPC connection.
func TestGeneratedUnionMessageBranchNameRoundTrip(t *testing.T) {
	const modulePath = "example.com/grpcunionbranchname"
	root := RunGRPCDSL(t, testdata.UnionMessageBranchNameDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(unionMessageBranchNameRoundTripHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}
