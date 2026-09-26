package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

// TestResultPayloadRoundTrips generates the service, HTTP and JSON-RPC code
// of services whose methods take an array of a result type that holds an
// optional and a required named union as payload and return the result type
// or a collection of it, with and without views. It vets the module and runs
// a harness that sends every branch through the generated clients and
// servers and checks that the results arrive unchanged in each view.
func TestResultPayloadRoundTrips(t *testing.T) {
	runDesignHarness(t, "example.com/resultpayload", resultPayloadDSL, resultPayloadHarness)
}

func resultPayloadDSL() {
	dsl.API("resultpayload", func() {
		dsl.JSONRPC(func() {})
	})
	leaf := dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Attribute("count", dsl.Int)
	})
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	rt := dsl.ResultType("application/vnd.rt", "RT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("c", choice)
		dsl.Attribute("d", choice)
		dsl.Required("d")
	})
	vrt := dsl.ResultType("application/vnd.vrt", "VRT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("c", choice)
		dsl.Attribute("d", choice)
		dsl.Required("d")
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("c")
			dsl.Attribute("d")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	dsl.Service("svc", func() {
		dsl.Method("put", func() {
			dsl.Payload(dsl.ArrayOf(rt))
			dsl.Result(rt)
			dsl.HTTP(func() {
				dsl.POST("/put")
			})
		})
		dsl.Method("puts", func() {
			dsl.Payload(dsl.ArrayOf(rt))
			dsl.Result(dsl.CollectionOf(rt))
			dsl.HTTP(func() {
				dsl.POST("/puts")
			})
		})
		dsl.Method("vput", func() {
			dsl.Payload(dsl.ArrayOf(vrt))
			dsl.Result(vrt)
			dsl.HTTP(func() {
				dsl.POST("/vput")
			})
		})
		dsl.Method("vputs", func() {
			dsl.Payload(dsl.ArrayOf(vrt))
			dsl.Result(dsl.CollectionOf(vrt))
			dsl.HTTP(func() {
				dsl.POST("/vputs")
			})
		})
	})
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("put", func() {
			dsl.Payload(dsl.ArrayOf(rt))
			dsl.Result(rt)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("puts", func() {
			dsl.Payload(dsl.ArrayOf(rt))
			dsl.Result(dsl.CollectionOf(rt))
			dsl.JSONRPC(func() {})
		})
	})
}

const resultPayloadHarness = `package resultpayload

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	svcclient "example.com/resultpayload/gen/http/svc/client"
	svcserver "example.com/resultpayload/gen/http/svc/server"
	rpcclient "example.com/resultpayload/gen/jsonrpc/rpc/client"
	rpcserver "example.com/resultpayload/gen/jsonrpc/rpc/server"
	rpc "example.com/resultpayload/gen/rpc"
	svc "example.com/resultpayload/gen/svc"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	view string
}

func (s *service) Put(_ context.Context, p []*svc.RT) (*svc.RT, error) { return p[0], nil }

func (s *service) Puts(_ context.Context, p []*svc.RT) (svc.RTCollection, error) { return p, nil }

func (s *service) Vput(_ context.Context, p []*svc.VRT) (*svc.VRT, string, error) {
	return p[0], s.view, nil
}

func (s *service) Vputs(_ context.Context, p []*svc.VRT) (svc.VRTCollection, string, error) {
	return p, s.view, nil
}

type rpcService struct{}

func (rpcService) Put(_ context.Context, p []*rpc.RT) (*rpc.RT, error) { return p[0], nil }

func (rpcService) Puts(_ context.Context, p []*rpc.RT) (rpc.RTCollection, error) { return p, nil }

func ptr[T any](v T) *T { return &v }

func TestHTTPRoundTrip(t *testing.T) {
	leaf := func(name string) svc.Choice { return svc.NewChoiceLeaf(&svc.Leaf{Name: name}) }
	other := func(count int) svc.Choice { return svc.NewChoiceOther(&svc.Other{Count: ptr(count)}) }
	s := &service{}
	mux := loomhttp.NewMuxer()
	svcserver.Mount(mux, svcserver.New(svc.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := svcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := svc.NewClient(c.Put(), c.Puts(), c.Vput(), c.Vputs())
	ctx := context.Background()

	rts := []*svc.RT{
		{ID: ptr("a"), C: ptr(leaf("c")), D: other(1)},
		{D: leaf("d")},
	}
	rt, err := client.Put(ctx, rts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rt, rts[0]) {
		t.Errorf("put: got %#v, want %#v", rt, rts[0])
	}
	got, err := client.Puts(ctx, rts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, svc.RTCollection(rts)) {
		t.Errorf("puts: got %#v, want %#v", got, rts)
	}

	vrts := []*svc.VRT{
		{ID: ptr("a"), C: ptr(other(2)), D: leaf("d")},
		{ID: ptr("b"), D: other(3)},
	}
	for _, view := range []string{"default", "tiny"} {
		s.view = view
		want := svc.VRTCollection(vrts)
		if view == "tiny" {
			want = svc.VRTCollection{{ID: ptr("a")}, {ID: ptr("b")}}
		}
		vrt, err := client.Vput(ctx, vrts)
		if err != nil {
			t.Fatalf("%s: %v", view, err)
		}
		if !reflect.DeepEqual(vrt, want[0]) {
			t.Errorf("%s: vput: got %#v, want %#v", view, vrt, want[0])
		}
		got, err := client.Vputs(ctx, vrts)
		if err != nil {
			t.Fatalf("%s: %v", view, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: vputs: got %#v, want %#v", view, got, want)
		}
	}
}

func TestJSONRPCRoundTrip(t *testing.T) {
	rts := []*rpc.RT{
		{ID: ptr("a"), C: ptr(rpc.NewChoiceLeaf(&rpc.Leaf{Name: "c"})), D: rpc.NewChoiceOther(&rpc.Other{Count: ptr(1)})},
		{D: rpc.NewChoiceLeaf(&rpc.Leaf{Name: "d"})},
	}
	mux := loomhttp.NewMuxer()
	rpcserver.Mount(mux, rpcserver.New(rpc.NewEndpoints(rpcService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := rpcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := rpc.NewClient(c.Put(), c.Puts())
	ctx := context.Background()

	rt, err := client.Put(ctx, rts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rt, rts[0]) {
		t.Errorf("put: got %#v, want %#v", rt, rts[0])
	}
	got, err := client.Puts(ctx, rts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, rpc.RTCollection(rts)) {
		t.Errorf("puts: got %#v, want %#v", got, rts)
	}
}
`
