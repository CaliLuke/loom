package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

// TestRequiredUnionViewsCompile generates the service and HTTP code of a
// service whose methods return a result type with views that requires a
// named union, a collection of it and a result type that holds it. It vets
// the module and runs a harness that converts the results through every
// view, validates the projected types and round-trips them over HTTP.
func TestRequiredUnionViewsCompile(t *testing.T) {
	runDesignHarness(t, "example.com/requiredunionviews", requiredUnionViewsDSL, requiredUnionViewsHarness)
}

func requiredUnionViewsDSL() {
	dsl.API("requiredunionviews", func() {})
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
		dsl.Required("id", "c")
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("c")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	parent := dsl.ResultType("application/vnd.parent", "Parent", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Attribute("child", rt)
		dsl.View("default", func() {
			dsl.Attribute("name")
			dsl.Attribute("child")
		})
	})
	dsl.Service("svc", func() {
		dsl.Method("get", func() {
			dsl.Result(rt)
			dsl.HTTP(func() {
				dsl.GET("/get")
			})
		})
		dsl.Method("list", func() {
			dsl.Result(dsl.CollectionOf(rt))
			dsl.HTTP(func() {
				dsl.GET("/list")
			})
		})
		dsl.Method("family", func() {
			dsl.Result(parent)
			dsl.HTTP(func() {
				dsl.GET("/family")
			})
		})
	})
}

const requiredUnionViewsHarness = `package requiredunionviews

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	svcclient "example.com/requiredunionviews/gen/http/svc/client"
	svcserver "example.com/requiredunionviews/gen/http/svc/server"
	svc "example.com/requiredunionviews/gen/svc"
	"example.com/requiredunionviews/gen/svc/views"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	view   string
	rt     *svc.RT
	list   svc.RTCollection
	parent *svc.Parent
}

func (s *service) Get(context.Context) (*svc.RT, string, error) { return s.rt, s.view, nil }

func (s *service) List(context.Context) (svc.RTCollection, string, error) {
	return s.list, s.view, nil
}

func (s *service) Family(context.Context) (*svc.Parent, error) { return s.parent, nil }

func ptr[T any](v T) *T { return &v }

func leaf() *svc.RT {
	return &svc.RT{ID: "a", C: svc.NewChoiceLeaf(&svc.Leaf{Name: "leaf"})}
}

func other() *svc.RT {
	return &svc.RT{ID: "b", C: svc.NewChoiceOther(&svc.Other{Count: ptr(2)})}
}

func TestViews(t *testing.T) {
	cases := []struct {
		view string
		want *svc.RT
	}{
		{"default", leaf()},
		{"tiny", &svc.RT{ID: "a"}},
	}
	for _, c := range cases {
		vres, err := svc.NewViewedRT(leaf(), c.view)
		if err != nil {
			t.Fatalf("%s: %v", c.view, err)
		}
		if err := views.ValidateRT(vres); err != nil {
			t.Errorf("%s: validate: %v", c.view, err)
		}
		got, err := svc.NewRT(vres)
		if err != nil {
			t.Fatalf("%s: %v", c.view, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %#v, want %#v", c.view, got, c.want)
		}
	}

	list := svc.RTCollection{leaf(), other()}
	vlist, err := svc.NewViewedRTCollection(list, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := views.ValidateRTCollection(vlist); err != nil {
		t.Errorf("collection: validate: %v", err)
	}
	gotList, err := svc.NewRTCollection(vlist)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotList, list) {
		t.Errorf("collection: got %#v, want %#v", gotList, list)
	}

	missing, err := svc.NewViewedRT(&svc.RT{ID: "a"}, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := views.ValidateRT(missing); err == nil || !strings.Contains(err.Error(), "\"c\"") {
		t.Errorf("missing union: got %v, want a missing c error", err)
	}

	parent := &svc.Parent{Name: ptr("p"), Child: other()}
	vparent, err := svc.NewViewedParent(parent, "default")
	if err != nil {
		t.Fatal(err)
	}
	if err := views.ValidateParent(vparent); err != nil {
		t.Errorf("parent: validate: %v", err)
	}
	gotParent, err := svc.NewParent(vparent)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotParent, parent) {
		t.Errorf("parent: got %#v, want %#v", gotParent, parent)
	}
}

func TestHTTPRoundTrip(t *testing.T) {
	s := &service{
		rt:     leaf(),
		list:   svc.RTCollection{leaf(), other()},
		parent: &svc.Parent{Name: ptr("p"), Child: other()},
	}
	mux := loomhttp.NewMuxer()
	svcserver.Mount(mux, svcserver.New(svc.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := svcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := svc.NewClient(c.Get(), c.List(), c.Family())
	ctx := context.Background()

	for _, view := range []string{"default", "tiny"} {
		s.view = view
		wantRT, wantList := s.rt, s.list
		if view == "tiny" {
			wantRT = &svc.RT{ID: "a"}
			wantList = svc.RTCollection{{ID: "a"}, {ID: "b"}}
		}
		rt, err := client.Get(ctx)
		if err != nil {
			t.Fatalf("%s: %v", view, err)
		}
		if !reflect.DeepEqual(rt, wantRT) {
			t.Errorf("%s: get: got %#v, want %#v", view, rt, wantRT)
		}
		list, err := client.List(ctx)
		if err != nil {
			t.Fatalf("%s: %v", view, err)
		}
		if !reflect.DeepEqual(list, wantList) {
			t.Errorf("%s: list: got %#v, want %#v", view, list, wantList)
		}
	}
	parent, err := client.Family(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parent, s.parent) {
		t.Errorf("parent: got %#v, want %#v", parent, s.parent)
	}
}
`
