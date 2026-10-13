package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const presenceCollectionHarness = `package rt

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	loomgrpc "github.com/CaliLuke/loom/grpc"
	loompb "github.com/CaliLuke/loom/grpc/pb"

	"example.com/richprobe/gen/grpc/rich/client"
	richpb "example.com/richprobe/gen/grpc/rich/pb"
	"example.com/richprobe/gen/grpc/rich/server"
	rich "example.com/richprobe/gen/rich"
)

func TestEmptyInnerCollections(t *testing.T) {
	cases := map[string]*rich.Rich{
		"populated control":          {Req: "r", Nested: [][]int{{1}}, Mapleaf: map[string][]*rich.Leaf{"k": {{Name: "value"}}}, Mm: map[string]map[string]int{"k": {"x": 1}}},
		"absent collections control": {Req: "r"},
		"named empty control":        {Req: "r", Named: []rich.Items{{}}},
		"length populated control":   {Req: "r", Nonempty: [][]int{{1}}},
		"empty nested array":         {Req: "r", Nested: [][]int{{}}},
		"empty array in map":         {Req: "r", Mapleaf: map[string][]*rich.Leaf{"k": {}}},
		"empty map in map":           {Req: "r", Mm: map[string]map[string]int{"k": {}}},
		"empty map in array":         {Req: "r", Maps: []map[string]int{{}}},
	}
	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			v.RequiredItems = rich.Items{}
			pb := client.NewProtoEchoRequest(v)
			b, err := proto.Marshal(pb)
			require.NoError(t, err)
			out := &richpb.EchoRequest{}
			require.NoError(t, proto.Unmarshal(b, out))
			t.Run("request validation", func(t *testing.T) {
				require.NoError(t, server.ValidateEchoRequest(out))
			})
			t.Run("request conversion", func(t *testing.T) {
				require.Equal(t, v, server.NewEchoPayload(out))
			})
			response := server.NewProtoEchoResponse(v)
			b, err = proto.Marshal(response)
			require.NoError(t, err)
			decoded := &richpb.EchoResponse{}
			require.NoError(t, proto.Unmarshal(b, decoded))
			t.Run("response validation", func(t *testing.T) {
				require.NoError(t, client.ValidateEchoResponse(decoded))
			})
			t.Run("response conversion", func(t *testing.T) {
				require.Equal(t, v, client.NewEchoResult(decoded))
			})
		})
	}
}
func TestCollectionRejectionControls(t *testing.T) {
	t.Run("nil wrapper element control", func(t *testing.T) {
		value := &rich.Rich{Req: "r", RequiredItems: rich.Items{}, Nested: [][]int{{1}}}
		request := client.NewProtoEchoRequest(value)
		request.Nested[0] = nil
		require.Error(t, server.ValidateEchoRequest(request))
		response := server.NewProtoEchoResponse(value)
		response.Nested[0] = nil
		require.Error(t, client.ValidateEchoResponse(response))
	})
	t.Run("required wrapper absent", func(t *testing.T) {
		value := &rich.Rich{Req: "r"}
		request := client.NewProtoEchoRequest(value)
		require.ErrorContains(t, server.ValidateEchoRequest(request), "required_items")
		require.ErrorContains(t, client.ValidateEchoResponse(server.NewProtoEchoResponse(value)), "required_items")
	})
	t.Run("named empty with MinLength1 control", func(t *testing.T) {
		value := &rich.Rich{Req: "r", RequiredItems: rich.Items{}, NamedNonempty: []rich.NonemptyItems{{}}}
		request := client.NewProtoEchoRequest(value)
		data, err := proto.Marshal(request)
		require.NoError(t, err)
		decoded := &richpb.EchoRequest{}
		require.NoError(t, proto.Unmarshal(data, decoded))
		require.ErrorContains(t, server.ValidateEchoRequest(decoded), "length")
        response := server.NewProtoEchoResponse(value)
        data, err = proto.Marshal(response)
        require.NoError(t, err)
        decodedResponse := &richpb.EchoResponse{}
        require.NoError(t, proto.Unmarshal(data, decodedResponse))
        require.ErrorContains(t, client.ValidateEchoResponse(decodedResponse), "length")
	})
	t.Run("empty with MinLength1", func(t *testing.T) {
		value := &rich.Rich{Req: "r", RequiredItems: rich.Items{}, Nonempty: [][]int{{}}}
		request := client.NewProtoEchoRequest(value)
		data, err := proto.Marshal(request)
		require.NoError(t, err)
		decoded := &richpb.EchoRequest{}
		require.NoError(t, proto.Unmarshal(data, decoded))
		require.ErrorContains(t, server.ValidateEchoRequest(decoded), "length")
        response := server.NewProtoEchoResponse(value)
        data, err = proto.Marshal(response)
        require.NoError(t, err)
        decodedResponse := &richpb.EchoResponse{}
        require.NoError(t, proto.Unmarshal(data, decodedResponse))
        require.ErrorContains(t, client.ValidateEchoResponse(decodedResponse), "length")
	})
}
`

func TestCollectionWrapperGeneratedContracts(t *testing.T) {
	const modulePath = "example.com/richprobe"
	root := RunGRPCDSL(t, collectionWrapperRichDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, modulePath, root, resolveGRPCLoomSource(t))
	modPath := filepath.Join(dir, "go.mod")
	mod, err := os.ReadFile(modPath)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(modPath, []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013020843-063198a2f427")), 0o600))
	testDir := filepath.Join(dir, "internal", "rt")
	require.NoError(t, os.MkdirAll(testDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(testDir, "rt_test.go"), []byte(presenceCollectionHarness+collectionWrapperMountedHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "./internal/rt", "-v")
}

func collectionWrapperRichDSL() {
	var Leaf = Type("Leaf", func() {
		Field(1, "name", String)
		Required("name")
	})
	items := Type("Items", ArrayOf(Int), func() {
		MinLength(0)
	})
	nonemptyItems := Type("NonemptyItems", ArrayOf(Int), func() {
		MinLength(1)
	})
	var Rich = Type("Rich", func() {
		Field(1, "req", String)
		Field(2, "nested", ArrayOf(ArrayOf(Int)))
		Field(3, "mapleaf", MapOf(String, ArrayOf(Leaf)))
		Field(4, "mm", MapOf(String, MapOf(String, Int)))
		Field(5, "named", ArrayOf(items))
		Field(6, "required_items", items)
		Field(7, "nonempty", ArrayOf(ArrayOf(Int), func() {
			MinLength(1)
		}))
		Field(8, "named_nonempty", ArrayOf(nonemptyItems))
		Field(9, "maps", ArrayOf(MapOf(String, Int)))
		Required("req", "required_items")
	})
	Service("rich", func() {
		Method("echo", func() {
			Payload(Rich)
			Result(Rich)
			GRPC(func() {
			})
		})
	})
}
