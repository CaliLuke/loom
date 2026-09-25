package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

// TestOptionalUnionValidation generates the service, view, HTTP, JSON-RPC and
// gRPC code of methods whose payload and result hold optional unions, named
// and inline, next to a required one. It vets the module and runs a harness
// that validates views and client CLI payloads and sends values through HTTP
// and JSON-RPC with the optional unions absent and present, and checks that
// no validation dereferences an absent union.
func TestOptionalUnionValidation(t *testing.T) {
	runDesignHarness(t, "example.com/optunion", optionalUnionDSL, optionalUnionHarness)
}

func optionalUnionDSL() {
	dsl.API("optunion", func() {
		dsl.JSONRPC(func() {})
	})
	leaf := dsl.Type("Leaf", func() {
		dsl.Field(1, "name", dsl.String, func() {
			dsl.MinLength(1)
		})
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Field(1, "count", dsl.Int, func() {
			dsl.Minimum(0)
		})
	})
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	holder := dsl.Type("Holder", func() {
		dsl.Field(1, "id", dsl.String)
		dsl.Field(2, "named", choice)
		dsl.Field(4, "req", choice)
		dsl.Field(6, "anon", func() {
			dsl.OneOf("anon", func() {
				dsl.Field(6, "leaf", leaf)
				dsl.Field(7, "text", dsl.String, func() {
					dsl.MinLength(2)
				})
			})
		})
		dsl.Required("id", "req")
	})
	rt := dsl.ResultType("application/vnd.rt", "RT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("c", choice)
		dsl.Attribute("anon", func() {
			dsl.OneOf("anon", func() {
				dsl.Attribute("leaf", leaf)
				dsl.Attribute("text", dsl.String, func() {
					dsl.MinLength(2)
				})
			})
		})
		dsl.Required("id")
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("c")
			dsl.Attribute("anon")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	dsl.Service("opt", func() {
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.HTTP(func() {
				dsl.POST("/show")
			})
			dsl.GRPC(func() {})
		})
		dsl.Method("rt", func() {
			dsl.Payload(holder)
			dsl.Result(rt)
			dsl.HTTP(func() {
				dsl.POST("/rt")
			})
		})
	})
	dsl.Service("optrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("rt", func() {
			dsl.Payload(holder)
			dsl.Result(rt)
			dsl.JSONRPC(func() {})
		})
	})
}

const optionalUnionHarness = `package optunion

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	httpclient "example.com/optunion/gen/http/opt/client"
	httpserver "example.com/optunion/gen/http/opt/server"
	rpcclient "example.com/optunion/gen/jsonrpc/optrpc/client"
	rpcserver "example.com/optunion/gen/jsonrpc/optrpc/server"
	opt "example.com/optunion/gen/opt"
	optviews "example.com/optunion/gen/opt/views"
	optrpc "example.com/optunion/gen/optrpc"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct{ rt *opt.RT }

func (*service) Show(_ context.Context, p *opt.Holder) (*opt.Holder, error) { return p, nil }

func (s *service) Rt(context.Context, *opt.Holder) (*opt.RT, string, error) {
	return s.rt, "default", nil
}

type rpcService struct{ rt *optrpc.RT }

func (*rpcService) Show(_ context.Context, p *optrpc.Holder) (*optrpc.Holder, error) { return p, nil }

func (s *rpcService) Rt(context.Context, *optrpc.Holder) (*optrpc.RT, string, error) {
	return s.rt, "default", nil
}

func ptr[T any](v T) *T { return &v }

// noPanic runs f and reports a panic as a test error.
func noPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s: panic: %v", name, r)
		}
	}()
	f()
}

func TestViewValidation(t *testing.T) {
	cases := []struct {
		name  string
		view  *optviews.RTView
		error string
	}{
		{"absent", &optviews.RTView{ID: ptr("a")}, ""},
		{"absent inline union", &optviews.RTView{ID: ptr("a"), Anon: &struct {
			Anon *optviews.Anon ` + "`" + `json:"anon,omitempty"` + "`" + `
		}{}}, ""},
		{"valid", &optviews.RTView{ID: ptr("a"), C: ptr(optviews.NewChoiceViewLeaf(&optviews.LeafView{Name: ptr("x")}))}, ""},
		{"invalid", &optviews.RTView{ID: ptr("a"), C: ptr(optviews.NewChoiceViewLeaf(&optviews.LeafView{Name: ptr("")}))}, "result.name"},
		{"invalid inline union", &optviews.RTView{ID: ptr("a"), Anon: &struct {
			Anon *optviews.Anon ` + "`" + `json:"anon,omitempty"` + "`" + `
		}{Anon: ptr(optviews.NewAnonText("x"))}}, "result.anon.anon.value"},
	}
	for _, c := range cases {
		noPanic(t, c.name, func() {
			err := optviews.ValidateRTView(c.view)
			switch {
			case c.error == "" && err != nil:
				t.Errorf("%s: unexpected error %v", c.name, err)
			case c.error != "" && (err == nil || !strings.Contains(err.Error(), c.error)):
				t.Errorf("%s: got %v, want an error about %s", c.name, err, c.error)
			}
		})
	}
}

func TestCLIPayload(t *testing.T) {
	const body = "{\"id\":\"a\",\"req\":{\"type\":\"Other\",\"value\":{}}}"
	const invalid = "{\"id\":\"a\",\"req\":{\"type\":\"Other\",\"value\":{}},\"named\":{\"type\":\"Leaf\",\"value\":{\"name\":\"\"}}}"
	noPanic(t, "http", func() {
		if _, err := httpclient.BuildShowPayload(body); err != nil {
			t.Errorf("http: %v", err)
		}
		if _, err := httpclient.BuildShowPayload(invalid); err == nil || !strings.Contains(err.Error(), "body.name") {
			t.Errorf("http invalid: got %v", err)
		}
	})
	noPanic(t, "jsonrpc", func() {
		if _, err := rpcclient.BuildShowPayload(body); err != nil {
			t.Errorf("jsonrpc: %v", err)
		}
		if _, err := rpcclient.BuildShowPayload(invalid); err == nil || !strings.Contains(err.Error(), "body.name") {
			t.Errorf("jsonrpc invalid: got %v", err)
		}
	})
}

func TestHTTPRoundTrip(t *testing.T) {
	s := &service{rt: &opt.RT{ID: "r"}}
	mux := loomhttp.NewMuxer()
	httpserver.Mount(mux, httpserver.New(opt.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := httpclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := opt.NewClient(c.Show(), c.Rt())
	ctx := context.Background()
	holders := []*opt.Holder{
		{ID: "absent", Req: opt.NewChoiceOther(&opt.Other{})},
		{ID: "inline", Req: opt.NewChoiceOther(&opt.Other{}), Anon: &struct {
			Anon *opt.Anon ` + "`" + `json:"anon,omitempty"` + "`" + `
		}{Anon: ptr(opt.NewAnonText("xy"))}},
		{ID: "present", Req: opt.NewChoiceOther(&opt.Other{}), Named: ptr(opt.NewChoiceLeaf(&opt.Leaf{Name: "x"}))},
	}
	for _, want := range holders {
		noPanic(t, want.ID, func() {
			got, err := client.Show(ctx, want)
			if err != nil {
				t.Errorf("%s: %v", want.ID, err)
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: got %#v, want %#v", want.ID, got, want)
			}
		})
	}
	noPanic(t, "rt", func() {
		got, err := client.Rt(ctx, holders[0])
		if err != nil {
			t.Errorf("rt: %v", err)
			return
		}
		if !reflect.DeepEqual(got, s.rt) {
			t.Errorf("rt: got %#v, want %#v", got, s.rt)
		}
	})
}

func TestJSONRPCRoundTrip(t *testing.T) {
	s := &rpcService{rt: &optrpc.RT{ID: "r"}}
	mux := loomhttp.NewMuxer()
	rpcserver.Mount(mux, rpcserver.New(optrpc.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := rpcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := optrpc.NewClient(c.Show(), c.Rt())
	ctx := context.Background()
	want := &optrpc.Holder{ID: "absent", Req: optrpc.NewChoiceOther(&optrpc.Other{})}
	noPanic(t, "show", func() {
		got, err := client.Show(ctx, want)
		if err != nil {
			t.Errorf("show: %v", err)
			return
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("show: got %#v, want %#v", got, want)
		}
	})
	noPanic(t, "rt", func() {
		got, err := client.Rt(ctx, want)
		if err != nil {
			t.Errorf("rt: %v", err)
			return
		}
		if !reflect.DeepEqual(got, s.rt) {
			t.Errorf("rt: got %#v, want %#v", got, s.rt)
		}
	})
}
`
