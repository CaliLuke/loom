package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

// TestUnionResultCollectionRoundTrips generates the service, HTTP and
// JSON-RPC code of services whose methods return result types that hold an
// optional and a required named union, with and without views, directly and
// as collections. It vets the module and runs a harness that round-trips
// every branch through the generated clients and servers in each view.
func TestUnionResultCollectionRoundTrips(t *testing.T) {
	runDesignHarness(t, "example.com/unionresultcollection", unionResultCollectionDSL, unionResultCollectionHarness)
}

func unionResultCollectionDSL() {
	dsl.API("unionresultcollection", func() {
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
		dsl.Method("rt", func() {
			dsl.Result(rt)
			dsl.HTTP(func() {
				dsl.GET("/rt")
			})
		})
		dsl.Method("rts", func() {
			dsl.Result(dsl.CollectionOf(rt))
			dsl.HTTP(func() {
				dsl.GET("/rts")
			})
		})
		dsl.Method("vrt", func() {
			dsl.Result(vrt)
			dsl.HTTP(func() {
				dsl.GET("/vrt")
			})
		})
		dsl.Method("vrts", func() {
			dsl.Result(dsl.CollectionOf(vrt))
			dsl.HTTP(func() {
				dsl.GET("/vrts")
			})
		})
	})
	dsl.Service("rpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("rt", func() {
			dsl.Result(rt)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("rts", func() {
			dsl.Result(dsl.CollectionOf(rt))
			dsl.JSONRPC(func() {})
		})
	})
}

const unionResultCollectionHarness = `package unionresultcollection

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	svcclient "example.com/unionresultcollection/gen/http/svc/client"
	svcserver "example.com/unionresultcollection/gen/http/svc/server"
	rpcclient "example.com/unionresultcollection/gen/jsonrpc/rpc/client"
	rpcserver "example.com/unionresultcollection/gen/jsonrpc/rpc/server"
	rpc "example.com/unionresultcollection/gen/rpc"
	svc "example.com/unionresultcollection/gen/svc"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	view string
	rts  svc.RTCollection
	vrts svc.VRTCollection
}

func (s *service) Rt(context.Context) (*svc.RT, error) { return s.rts[0], nil }

func (s *service) Rts(context.Context) (svc.RTCollection, error) { return s.rts, nil }

func (s *service) Vrt(context.Context) (*svc.VRT, string, error) { return s.vrts[0], s.view, nil }

func (s *service) Vrts(context.Context) (svc.VRTCollection, string, error) {
	return s.vrts, s.view, nil
}

type rpcService struct {
	rts rpc.RTCollection
}

func (s *rpcService) Rt(context.Context) (*rpc.RT, error) { return s.rts[0], nil }

func (s *rpcService) Rts(context.Context) (rpc.RTCollection, error) { return s.rts, nil }

func ptr[T any](v T) *T { return &v }

func TestHTTPRoundTrip(t *testing.T) {
	leaf := func(name string) svc.Choice { return svc.NewChoiceLeaf(&svc.Leaf{Name: name}) }
	other := func(count int) svc.Choice { return svc.NewChoiceOther(&svc.Other{Count: ptr(count)}) }
	s := &service{
		rts: svc.RTCollection{
			{ID: ptr("a"), C: ptr(leaf("c")), D: other(1)},
			{D: leaf("d")},
		},
		vrts: svc.VRTCollection{
			{ID: ptr("a"), C: ptr(other(2)), D: leaf("d")},
			{ID: ptr("b"), D: other(3)},
		},
	}
	mux := loomhttp.NewMuxer()
	svcserver.Mount(mux, svcserver.New(svc.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := svcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := svc.NewClient(c.Rt(), c.Rts(), c.Vrt(), c.Vrts())
	ctx := context.Background()

	rt, err := client.Rt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rt, s.rts[0]) {
		t.Errorf("rt: got %#v, want %#v", rt, s.rts[0])
	}
	rts, err := client.Rts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rts, s.rts) {
		t.Errorf("rts: got %#v, want %#v", rts, s.rts)
	}
	for _, view := range []string{"default", "tiny"} {
		s.view = view
		want := s.vrts
		if view == "tiny" {
			want = svc.VRTCollection{{ID: ptr("a")}, {ID: ptr("b")}}
		}
		vrt, err := client.Vrt(ctx)
		if err != nil {
			t.Fatalf("%s: %v", view, err)
		}
		if !reflect.DeepEqual(vrt, want[0]) {
			t.Errorf("%s: vrt: got %#v, want %#v", view, vrt, want[0])
		}
		vrts, err := client.Vrts(ctx)
		if err != nil {
			t.Fatalf("%s: %v", view, err)
		}
		if !reflect.DeepEqual(vrts, want) {
			t.Errorf("%s: vrts: got %#v, want %#v", view, vrts, want)
		}
	}
}

func TestJSONRPCRoundTrip(t *testing.T) {
	s := &rpcService{rts: rpc.RTCollection{
		{ID: ptr("a"), C: ptr(rpc.NewChoiceLeaf(&rpc.Leaf{Name: "c"})), D: rpc.NewChoiceOther(&rpc.Other{Count: ptr(1)})},
		{D: rpc.NewChoiceLeaf(&rpc.Leaf{Name: "d"})},
	}}
	mux := loomhttp.NewMuxer()
	rpcserver.Mount(mux, rpcserver.New(rpc.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := rpcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := rpc.NewClient(c.Rt(), c.Rts())
	ctx := context.Background()

	rt, err := client.Rt(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rt, s.rts[0]) {
		t.Errorf("rt: got %#v, want %#v", rt, s.rts[0])
	}
	rts, err := client.Rts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rts, s.rts) {
		t.Errorf("rts: got %#v, want %#v", rts, s.rts)
	}
}
`
