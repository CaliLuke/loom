package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	. "github.com/CaliLuke/loom/dsl"
)

var protoStructMetaRoundTripHarness = fmt.Sprintf(`package roundtrip

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"%[1]s/gen/grpc/menusvc/client"
	pb "%[1]s/gen/grpc/menusvc/pb"
	"%[1]s/gen/grpc/menusvc/server"
	"%[1]s/gen/menu"
	"%[1]s/gen/menusvc"
)

// The protocol buffer messages live in the pb package, not in the
// struct:pkg:path package of the service types. The top-level messages of a
// type take its struct:name:proto name, converted by protoc-gen-go for
// node_tree, and nested messages keep the name of the type.
var (
	_ *pb.MenuProto
	_ *pb.InnerProto
	_ *pb.Inner
	_ *pb.Leaf
	_ *pb.Choice
	_ *pb.FaultProto
	_ *pb.Other
	_ *pb.Tags
	_ *pb.NodeTree
	_ *pb.Node
	_ *pb.EventProto
	_ *pb.Event
	_ *pb.BidiStreamingRequest_StreamItem
	_ *pb.RelayResponse
)

type service struct{}

func (service) Show(_ context.Context, p *menu.Menu) (*menu.Menu, error) {
	if p.Name == "fail" {
		return nil, &menu.Fault{Name: "invalid", Message: "bad menu"}
	}
	return p, nil
}

func (service) Pick(_ context.Context, p *menu.Choice) (*menu.Choice, error) {
	return p, nil
}

func (service) Branch(_ context.Context, p *menusvc.ChoiceOrInner) (*menusvc.ChoiceOrInner, error) {
	res := &menusvc.ChoiceOrInner{}
	if c, ok := p.AsChoice(); ok {
		res.SetChoice(c)
	}
	if i, ok := p.AsInner(); ok {
		res.SetInner(i)
	}
	return res, nil
}

func (service) Tree(_ context.Context, p *menu.Node) (*menu.Node, error) {
	return p, nil
}

func (service) TagsEndpoint(_ context.Context, p menu.Tags) (*menusvc.TagsResult, error) {
	return &menusvc.TagsResult{Tags: p}, nil
}

// Bidi echoes the stream items with the initial payload prepended to their
// identifiers.
func (service) Bidi(_ context.Context, p *menusvc.BidiPayload, stream menusvc.BidiServerStream) error {
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.Close()
		}
		if err != nil {
			return err
		}
		id := *p.Q + *ev.ID
		if err := stream.Send(&menusvc.Event{ID: &id}); err != nil {
			return err
		}
	}
}

// Relay echoes the stream items nested in results.
func (service) Relay(_ context.Context, _ *menusvc.RelayPayload, stream menusvc.RelayServerStream) error {
	for {
		ev, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.Close()
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&menusvc.RelayResult{Event: ev}); err != nil {
			return err
		}
	}
}

func (service) Watch(_ context.Context, p *menu.Inner, stream menusvc.WatchServerStream) error {
	for range 2 {
		if err := stream.Send(&menu.Menu{Name: "watch", Inner: p}); err != nil {
			return err
		}
	}
	return stream.Close()
}

func dial(t *testing.T) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb.RegisterMenusvcServer(srv, server.New(menusvc.NewEndpoints(service{}), nil, nil))
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
	count := 7
	flag := true
	leafName := "leaf"
	inner := &menu.Inner{Count: &count}
	leafChoice := &menu.Choice{}
	leafChoice.SetLeaf(&menu.Leaf{Name: &leafName})
	otherChoice := &menu.Choice{}
	otherChoice.SetOther(&menu.Other{Flag: &flag})
	root, leafLabel := "root", "leaf"
	tree := &menu.Node{
		Label:    &root,
		Children: []*menu.Node{{Label: &leafLabel, Children: []*menu.Node{{Label: &root}}}},
	}

	t.Run("show", func(t *testing.T) {
		want := &menu.Menu{
			Name:       "lunch",
			Inner:      inner,
			Inners:     []*menu.Inner{inner, {}},
			InnerIndex: map[string]*menu.Inner{"a": inner},
			Choice:     leafChoice,
			Tags:       menu.Tags{"a", "b"},
			Tree:       tree,
		}
		got, err := c.Show()(ctx, want)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
	t.Run("show error", func(t *testing.T) {
		_, err := c.Show()(ctx, &menu.Menu{Name: "fail"})
		var fault *menu.Fault
		require.True(t, errors.As(err, &fault), "%%v", err)
		require.Equal(t, &menu.Fault{Name: "invalid", Message: "bad menu"}, fault)
	})
	for name, want := range map[string]*menu.Choice{"leaf": leafChoice, "other": otherChoice} {
		t.Run("pick/"+name, func(t *testing.T) {
			got, err := c.Pick()(ctx, want)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
		t.Run("branch choice/"+name, func(t *testing.T) {
			p := &menusvc.ChoiceOrInner{}
			p.SetChoice(want)
			got, err := c.Branch()(ctx, p)
			require.NoError(t, err)
			res := got.(*menusvc.ChoiceOrInner)
			choice, ok := res.AsChoice()
			require.True(t, ok)
			require.Equal(t, want, choice)
		})
	}
	t.Run("branch inner", func(t *testing.T) {
		p := &menusvc.ChoiceOrInner{}
		p.SetInner(inner)
		got, err := c.Branch()(ctx, p)
		require.NoError(t, err)
		res := got.(*menusvc.ChoiceOrInner)
		i, ok := res.AsInner()
		require.True(t, ok)
		require.Equal(t, inner, i)
	})
	t.Run("tree", func(t *testing.T) {
		got, err := c.Tree()(ctx, tree)
		require.NoError(t, err)
		require.Equal(t, tree, got)
	})
	t.Run("tags", func(t *testing.T) {
		got, err := c.TagsEndpoint()(ctx, menu.Tags{"a", "b"})
		require.NoError(t, err)
		require.Equal(t, &menusvc.TagsResult{Tags: menu.Tags{"a", "b"}}, got)
	})
	t.Run("bidi", func(t *testing.T) {
		q, id := "q-", "1"
		res, err := c.Bidi()(ctx, &menusvc.BidiPayload{Q: &q})
		require.NoError(t, err)
		stream := res.(menusvc.BidiClientStream)
		require.NoError(t, stream.Send(&menusvc.Event{ID: &id}))
		got, err := stream.Recv()
		require.NoError(t, err)
		require.Equal(t, "q-1", *got.ID)
		require.NoError(t, stream.Close())
		_, err = stream.Recv()
		require.ErrorIs(t, err, io.EOF)
	})
	t.Run("relay", func(t *testing.T) {
		q, id := "q", "2"
		res, err := c.Relay()(ctx, &menusvc.RelayPayload{Q: &q})
		require.NoError(t, err)
		stream := res.(menusvc.RelayClientStream)
		require.NoError(t, stream.Send(&menusvc.Event{ID: &id}))
		got, err := stream.Recv()
		require.NoError(t, err)
		require.Equal(t, &menusvc.RelayResult{Event: &menusvc.Event{ID: &id}}, got)
		require.NoError(t, stream.Close())
		_, err = stream.Recv()
		require.ErrorIs(t, err, io.EOF)
	})
	t.Run("watch", func(t *testing.T) {
		res, err := c.Watch()(ctx, inner)
		require.NoError(t, err)
		stream := res.(menusvc.WatchClientStream)
		for range 2 {
			got, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, &menu.Menu{Name: "watch", Inner: inner}, got)
		}
		_, err = stream.Recv()
		require.ErrorIs(t, err, io.EOF)
	})
}
`, "example.com/grpcstructmeta")

// TestProtoFilesStructMeta checks the protocol buffer messages of types with
// struct:pkg:path and struct:name:proto metadata. The messages generated for
// a type used directly as a payload, result, error, stream or stream item
// take its struct:name:proto name, as before. The messages of the type in
// nested positions, such as message fields, array elements, map values and
// union branches, and the messages that wrap a named union or a named array
// keep the name of the type. No message refers to the struct:pkg:path
// package. protoc must accept the result.
func TestProtoFilesStructMeta(t *testing.T) {
	code := protoFileCode(t, codegentestdata.ProtoStructMetaDSL)
	for _, want := range []string{
		"rpc Show (MenuProto) returns (MenuProto);",
		"rpc Pick (Choice) returns (Choice);",
		"rpc Tree (node_tree) returns (node_tree);",
		"rpc TagsEndpoint (Tags) returns (TagsResponse);",
		"rpc Bidi (stream BidiStreamingRequest) returns (stream EventProto);",
		"rpc Relay (stream RelayStreamingRequest) returns (stream RelayResponse);",
		"rpc Watch (InnerProto) returns (stream MenuProto);",
		"message FaultProto {",
		"message MenuProto {\n\tstring name = 1;\n\tInner inner = 2;\n\trepeated Inner inners = 3;\n\tmap<string, Inner> inner_index = 4;\n" +
			"\toneof choice {\n\t\tLeaf leaf = 5;\n\t\tOther other = 6;\n\t}\n\tTags tags = 7;\n\tNode tree = 8;\n}",
		"message InnerProto {",
		"message Inner {",
		"message Choice {\n\toneof field {\n\t\tLeaf leaf = 1;",
		"message BranchRequest {\n\toneof field {\n\t\tChoice choice = 1;\n\t\tInner inner = 2;",
		"message node_tree {\n\toptional string label = 1;\n\trepeated Node children = 2;\n}",
		"message Node {\n\toptional string label = 1;\n\trepeated Node children = 2;\n}",
		"message BidiStreamingRequest {\n\toneof body {\n\t\tBidiRequest initial_payload = 1;\n\t\tEventProto stream_item = 2;\n\t}\n}",
		"message RelayResponse {\n\tEvent event = 1;\n}",
	} {
		assert.Contains(t, code, want)
	}
	for _, unwanted := range []string{"LeafProto", "ChoiceProto", "TagList", "menu."} {
		assert.NotContains(t, code, unwanted)
	}
	fpath := codegen.CreateTempFile(t, code)
	assert.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
}

// TestStructMetaProtoTypeRefs checks that the Go references to the protocol
// buffer messages of types with struct:pkg:path and struct:name:proto
// metadata are qualified with the pb package, not the struct:pkg:path
// package, and name the Go types that protoc-gen-go generates, and that the
// response contracts name the generated messages.
func TestStructMetaProtoTypeRefs(t *testing.T) {
	root := RunGRPCDSL(t, codegentestdata.ProtoStructMetaDSL)
	sd := CreateGRPCServices(root).Get("menusvc")
	require.NotNil(t, sd)

	refs := make(map[string]string, len(sd.Messages))
	for _, m := range sd.Messages {
		refs[m.VarName] = m.Ref
	}
	assert.Equal(t, map[string]string{
		"MenuProto":             "*menusvcpb.MenuProto",
		"InnerProto":            "*menusvcpb.InnerProto",
		"Inner":                 "*menusvcpb.Inner",
		"Leaf":                  "*menusvcpb.Leaf",
		"Other":                 "*menusvcpb.Other",
		"Choice":                "*menusvcpb.Choice",
		"FaultProto":            "*menusvcpb.FaultProto",
		"BranchRequest":         "*menusvcpb.BranchRequest",
		"BranchResponse":        "*menusvcpb.BranchResponse",
		"Tags":                  "*menusvcpb.Tags",
		"node_tree":             "*menusvcpb.NodeTree",
		"Node":                  "*menusvcpb.Node",
		"TagsResponse":          "*menusvcpb.TagsResponse",
		"BidiStreamingRequest":  "*menusvcpb.BidiStreamingRequest",
		"BidiRequest":           "*menusvcpb.BidiRequest",
		"EventProto":            "*menusvcpb.EventProto",
		"Event":                 "*menusvcpb.Event",
		"RelayStreamingRequest": "*menusvcpb.RelayStreamingRequest",
		"RelayRequest":          "*menusvcpb.RelayRequest",
		"RelayResponse":         "*menusvcpb.RelayResponse",
	}, refs)

	contracts := make(map[string][2]string)
	for _, e := range sd.Endpoints {
		for _, c := range e.ResponseContractCases {
			contracts[c.ID] = [2]string{c.MessageType, c.DetailType}
		}
		if e.Request.ServerConvert != nil {
			assert.Equal(t, "*menusvcpb."+protoRequestMessageName(e.Method.Name), e.Request.ServerConvert.SrcRef, e.Method.Name)
		}
	}
	assert.Equal(t, map[string][2]string{
		"menusvc.show.success.0":       {"menusvc.MenuProto", ""},
		"menusvc.show.error.invalid.3": {"", "menusvc.FaultProto"},
		"menusvc.pick.success.0":       {"menusvc.Choice", ""},
		"menusvc.branch.success.0":     {"menusvc.BranchResponse", ""},
		"menusvc.tags.success.0":       {"menusvc.TagsResponse", ""},
		"menusvc.tree.success.0":       {"menusvc.node_tree", ""},
		"menusvc.watch.success.0":      {"menusvc.MenuProto", ""},
	}, contracts)
}

// TestStructMetaStreamingPayloadProtoTypeRefs checks that the protocol buffer
// message of a streaming payload of a type with struct:pkg:path metadata, with
// or without struct:name:proto metadata, is referenced from the pb package,
// then compiles and vets the generated module.
func TestStructMetaStreamingPayloadProtoTypeRefs(t *testing.T) {
	root := RunGRPCDSL(t, func() {
		plain := Type("Plain", func() {
			Meta("struct:pkg:path", "menu")
			Field(1, "name", String)
		})
		named := Type("Named", func() {
			Meta("struct:pkg:path", "menu")
			Meta("struct:name:proto", "named_proto")
			Field(1, "name", String)
		})
		Service("svc", func() {
			Method("upload", func() {
				StreamingPayload(plain)
				GRPC(func() {})
			})
			Method("relay", func() {
				StreamingPayload(named)
				StreamingResult(named)
				GRPC(func() {})
			})
		})
	})
	services := CreateGRPCServices(root)
	sd := services.Get("svc")
	require.NotNil(t, sd)
	files := append(ClientTypeFiles("", services), ServerTypeFiles("", services)...)
	require.NotEmpty(t, files)
	for _, f := range files {
		code := sectionCode(t, f.AllSections()...)
		for _, unwanted := range []string{"menu.UploadStreamingRequest", "menu.NamedProto", "menu.named_proto", "menu.RelayStreamingRequest"} {
			assert.NotContains(t, code, unwanted, f.Path)
		}
	}
	expected := map[string]string{
		"upload": "*svcpb.UploadStreamingRequest",
		"relay":  "*svcpb.NamedProto",
	}
	for _, e := range sd.Endpoints {
		want := expected[e.Method.Name]
		require.NotNil(t, e.ServerStream.RecvConvert, e.Method.Name)
		assert.Equal(t, want, e.ServerStream.RecvConvert.SrcRef, e.Method.Name)
		require.NotNil(t, e.ClientStream.SendConvert, e.Method.Name)
		assert.Equal(t, want, e.ClientStream.SendConvert.TgtRef, e.Method.Name)
	}

	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/grpcstreammeta", root, resolveGRPCLoomSource(t))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
}

// TestGeneratedStructMetaRoundTrip compiles and vets a generated module whose
// types have struct:pkg:path and struct:name:proto metadata. It serves the
// generated server over an in-memory gRPC connection and round-trips every
// method, including a service error, a recursive type and a server stream,
// through the generated client.
func TestGeneratedStructMetaRoundTrip(t *testing.T) {
	const modulePath = "example.com/grpcstructmeta"
	root := RunGRPCDSL(t, codegentestdata.ProtoStructMetaDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	testDir := filepath.Join(dir, "internal", "roundtrip")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "roundtrip_test.go"), []byte(protoStructMetaRoundTripHarness), 0o600))

	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "./internal/roundtrip")
}

// protoRequestMessageName returns the name of the request message of the
// Go type of the method of ProtoStructMetaDSL named method.
func protoRequestMessageName(method string) string {
	return map[string]string{
		"show":   "MenuProto",
		"pick":   "Choice",
		"branch": "BranchRequest",
		"tree":   "NodeTree",
		"tags":   "Tags",
		"bidi":   "BidiRequest",
		"relay":  "RelayRequest",
		"watch":  "InnerProto",
	}[method]
}
