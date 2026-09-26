package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestUnionInUnionRoundTrips generates the service, HTTP, JSON-RPC and gRPC
// code of methods whose payload and result hold unions that have a named
// union, possibly nested again or held in a collection, as a branch. It vets
// the module and runs a harness that sends every branch through each
// transport that supports it and checks that it arrives unchanged.
func TestUnionInUnionRoundTrips(t *testing.T) {
	runDesignHarness(t, "example.com/unioninunion", unionInUnionDSL, unionInUnionHarness)
}

// runDesignHarness generates the service and transport code of design into
// a module named module, vets the module and runs the tests of harness, a
// test file of the root package of the module.
func runDesignHarness(t *testing.T, module string, design func(), harness string) {
	t.Helper()
	root := codegen.RunDSL(t, design)
	roots := []eval.Root{root}
	genpkg := module + "/gen"
	serviceFiles, err := Service(genpkg, roots)
	require.NoError(t, err)
	transportFiles, err := Transport(genpkg, roots)
	require.NoError(t, err)

	dir := t.TempDir()
	for _, file := range mergeFilesByPath(append(serviceFiles, transportFiles...)) {
		_, err := file.Render(dir)
		require.NoError(t, err, file.Path)
	}
	source := loomModuleSource(t)
	goMod := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", module, source)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harness_test.go"), []byte(harness), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "-count=1", ".")
	require.NoError(t, err, output)
}

func unionInUnionDSL() {
	dsl.API("unioninunion", func() {
		dsl.JSONRPC(func() {})
	})
	leaf := dsl.Type("Leaf", func() {
		dsl.Field(1, "name", dsl.String)
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Field(1, "count", dsl.Int)
	})
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	tags := dsl.Type("Tags", func() {
		dsl.Field(1, "values", dsl.ArrayOf(dsl.String))
	})
	outer := dsl.Type("Outer", dsl.OneOf(choice, tags))
	holder := dsl.Type("Holder", func() {
		dsl.Field(1, "id", dsl.String)
		dsl.Field(2, "p", dsl.OneOf(choice, tags))
		dsl.Field(4, "r", dsl.OneOf(outer, other))
		dsl.Required("id")
	})
	// gRPC does not support unions as collection elements.
	choiceList := dsl.Type("ChoiceList", dsl.ArrayOf(choice))
	choiceMap := dsl.Type("ChoiceMap", dsl.MapOf(dsl.String, choice))
	lists := dsl.Type("Lists", func() {
		dsl.Field(1, "id", dsl.String)
		dsl.Field(2, "q", dsl.OneOf(choiceList, choiceMap))
		dsl.Required("id")
	})
	dsl.Service("nest", func() {
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.HTTP(func() {
				dsl.POST("/show")
			})
			dsl.GRPC(func() {})
		})
		dsl.Method("list", func() {
			dsl.Payload(lists)
			dsl.Result(lists)
			dsl.HTTP(func() {
				dsl.POST("/list")
			})
		})
	})
	dsl.Service("nestrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("list", func() {
			dsl.Payload(lists)
			dsl.Result(lists)
			dsl.JSONRPC(func() {})
		})
	})
}

const unionInUnionHarness = `package unioninunion

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	grpcclient "example.com/unioninunion/gen/grpc/nest/client"
	nestpb "example.com/unioninunion/gen/grpc/nest/pb"
	grpcserver "example.com/unioninunion/gen/grpc/nest/server"
	httpclient "example.com/unioninunion/gen/http/nest/client"
	httpserver "example.com/unioninunion/gen/http/nest/server"
	rpcclient "example.com/unioninunion/gen/jsonrpc/nestrpc/client"
	rpcserver "example.com/unioninunion/gen/jsonrpc/nestrpc/server"
	nest "example.com/unioninunion/gen/nest"
	nestrpc "example.com/unioninunion/gen/nestrpc"
	loomhttp "github.com/CaliLuke/loom/http"
)

type echo struct{}

func (echo) Show(_ context.Context, p *nest.Holder) (*nest.Holder, error) { return p, nil }

func (echo) List(_ context.Context, p *nest.Lists) (*nest.Lists, error) { return p, nil }

type rpcEcho struct{}

func (rpcEcho) Show(_ context.Context, p *nestrpc.Holder) (*nestrpc.Holder, error) { return p, nil }

func (rpcEcho) List(_ context.Context, p *nestrpc.Lists) (*nestrpc.Lists, error) { return p, nil }

func ptr[T any](v T) *T { return &v }

func holders() []*nest.Holder {
	return []*nest.Holder{
		{ID: "leaf", P: ptr(nest.NewChoiceOrTagsChoice(ptr(nest.NewChoiceLeaf(&nest.Leaf{Name: "x"}))))},
		{ID: "other", P: ptr(nest.NewChoiceOrTagsChoice(ptr(nest.NewChoiceOther(&nest.Other{Count: ptr(2)}))))},
		{ID: "tags", P: ptr(nest.NewChoiceOrTagsTags(&nest.Tags{Values: []string{"a", "b"}}))},
		{ID: "deep", R: ptr(nest.NewOtherOrOuterOuter(ptr(nest.NewOuterChoice(ptr(nest.NewChoiceLeaf(&nest.Leaf{Name: "y"}))))))},
		{ID: "deep-other", R: ptr(nest.NewOtherOrOuterOther(&nest.Other{}))},
		{ID: "none"},
	}
}

func rpcHolders() []*nestrpc.Holder {
	return []*nestrpc.Holder{
		{ID: "leaf", P: ptr(nestrpc.NewChoiceOrTagsChoice(ptr(nestrpc.NewChoiceLeaf(&nestrpc.Leaf{Name: "x"}))))},
		{ID: "other", P: ptr(nestrpc.NewChoiceOrTagsChoice(ptr(nestrpc.NewChoiceOther(&nestrpc.Other{Count: ptr(2)}))))},
		{ID: "tags", P: ptr(nestrpc.NewChoiceOrTagsTags(&nestrpc.Tags{Values: []string{"a", "b"}}))},
		{ID: "deep", R: ptr(nestrpc.NewOtherOrOuterOuter(ptr(nestrpc.NewOuterChoice(ptr(nestrpc.NewChoiceLeaf(&nestrpc.Leaf{Name: "y"}))))))},
		{ID: "deep-other", R: ptr(nestrpc.NewOtherOrOuterOther(&nestrpc.Other{}))},
		{ID: "none"},
	}
}

func lists() []*nest.Lists {
	choices := []nest.Choice{nest.NewChoiceLeaf(&nest.Leaf{Name: "x"}), nest.NewChoiceOther(&nest.Other{Count: ptr(2)})}
	return []*nest.Lists{
		{ID: "list", Q: ptr(nest.NewChoiceListOrChoiceMapChoiceList(nest.ChoiceList(choices)))},
		{ID: "map", Q: ptr(nest.NewChoiceListOrChoiceMapChoiceMap(nest.ChoiceMap{"a": choices[0], "b": choices[1]}))},
		{ID: "none"},
	}
}

func rpcLists() []*nestrpc.Lists {
	choices := []nestrpc.Choice{nestrpc.NewChoiceLeaf(&nestrpc.Leaf{Name: "x"}), nestrpc.NewChoiceOther(&nestrpc.Other{Count: ptr(2)})}
	return []*nestrpc.Lists{
		{ID: "list", Q: ptr(nestrpc.NewChoiceListOrChoiceMapChoiceList(nestrpc.ChoiceList(choices)))},
		{ID: "map", Q: ptr(nestrpc.NewChoiceListOrChoiceMapChoiceMap(nestrpc.ChoiceMap{"a": choices[0], "b": choices[1]}))},
		{ID: "none"},
	}
}

func TestHTTPRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	httpserver.Mount(mux, httpserver.New(nest.NewEndpoints(echo{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := httpclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := nest.NewClient(c.Show(), c.List())
	for _, want := range holders() {
		got, err := client.Show(context.Background(), want)
		if err != nil {
			t.Errorf("%s: %v", want.ID, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %#v, want %#v", want.ID, got, want)
		}
	}
	for _, want := range lists() {
		got, err := client.List(context.Background(), want)
		if err != nil {
			t.Errorf("%s: %v", want.ID, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %#v, want %#v", want.ID, got, want)
		}
	}
}

func TestJSONRPCRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	rpcserver.Mount(mux, rpcserver.New(nestrpc.NewEndpoints(rpcEcho{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := rpcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := nestrpc.NewClient(c.Show(), c.List())
	for _, want := range rpcHolders() {
		got, err := client.Show(context.Background(), want)
		if err != nil {
			t.Errorf("%s: %v", want.ID, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %#v, want %#v", want.ID, got, want)
		}
	}
	for _, want := range rpcLists() {
		got, err := client.List(context.Background(), want)
		if err != nil {
			t.Errorf("%s: %v", want.ID, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %#v, want %#v", want.ID, got, want)
		}
	}
}

func TestGRPCConversions(t *testing.T) {
	for _, want := range holders() {
		wire, err := proto.Marshal(grpcclient.NewProtoShowRequest(want))
		if err != nil {
			t.Fatalf("%s: %v", want.ID, err)
		}
		var req nestpb.ShowRequest
		if err := proto.Unmarshal(wire, &req); err != nil {
			t.Fatalf("%s: %v", want.ID, err)
		}
		if err := grpcserver.ValidateShowRequest(&req); err != nil {
			t.Errorf("%s: validate request: %v", want.ID, err)
		}
		payload := grpcserver.NewShowPayload(&req)
		if !reflect.DeepEqual(payload, want) {
			t.Errorf("%s: payload: got %#v, want %#v", want.ID, payload, want)
		}
		wire, err = proto.Marshal(grpcserver.NewProtoShowResponse(want))
		if err != nil {
			t.Fatalf("%s: %v", want.ID, err)
		}
		var res nestpb.ShowResponse
		if err := proto.Unmarshal(wire, &res); err != nil {
			t.Fatalf("%s: %v", want.ID, err)
		}
		if err := grpcclient.ValidateShowResponse(&res); err != nil {
			t.Errorf("%s: validate response: %v", want.ID, err)
		}
		result := grpcclient.NewShowResult(&res)
		if !reflect.DeepEqual(result, want) {
			t.Errorf("%s: result: got %#v, want %#v", want.ID, result, want)
		}
	}
}
`
