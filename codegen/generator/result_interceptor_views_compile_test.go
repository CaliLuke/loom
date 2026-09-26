package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

// TestResultInterceptorViewsRoundTrips generates the service and HTTP code
// of a service whose methods return a result type with views, a result type
// with the default view only and a user type, read and written by a server
// and a client result interceptor, with a server interceptor that does not
// access the result around them. It vets the module and runs a harness that
// calls each method over HTTP in each view and checks that both result
// interceptors read the result, that their writes reach the client and that
// the other interceptor gets the viewed result that the endpoint returns. It
// also checks that a nil result or a result of another type returned by an
// interceptor fails with a fault error instead of a panic.
func TestResultInterceptorViewsRoundTrips(t *testing.T) {
	runDesignHarness(t, "example.com/resultinterceptorviews", resultInterceptorViewsDSL, resultInterceptorViewsHarness)
}

func resultInterceptorViewsDSL() {
	dsl.API("resultinterceptorviews", func() {})
	dsl.Interceptor("pass")
	dsl.Interceptor("stamp", func() {
		dsl.ReadResult(func() {
			dsl.Attribute("id")
		})
		dsl.WriteResult(func() {
			dsl.Attribute("id")
		})
	})
	vrt := dsl.ResultType("application/vnd.vrt", "VRT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("name", dsl.String)
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("name")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	rt := dsl.ResultType("application/vnd.rt", "RT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("name", dsl.String)
	})
	ut := dsl.Type("UT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("name", dsl.String)
	})
	dsl.Service("svc", func() {
		dsl.ServerInterceptor("pass")
		dsl.ServerInterceptor("stamp")
		dsl.ClientInterceptor("stamp")
		dsl.Method("vget", func() {
			dsl.Result(vrt)
			dsl.HTTP(func() {
				dsl.GET("/vget")
			})
		})
		dsl.Method("get", func() {
			dsl.Result(rt)
			dsl.HTTP(func() {
				dsl.GET("/get")
			})
		})
		dsl.Method("show", func() {
			dsl.Result(ut)
			dsl.HTTP(func() {
				dsl.GET("/show")
			})
		})
	})
}

const resultInterceptorViewsHarness = `package resultinterceptorviews

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	svcclient "example.com/resultinterceptorviews/gen/http/svc/client"
	svcserver "example.com/resultinterceptorviews/gen/http/svc/server"
	svc "example.com/resultinterceptorviews/gen/svc"
	svcviews "example.com/resultinterceptorviews/gen/svc/views"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

type service struct {
	view string
}

func (s *service) Vget(context.Context) (*svc.VRT, string, error) {
	return &svc.VRT{ID: ptr("a"), Name: ptr("n")}, s.view, nil
}

func (s *service) Get(context.Context) (*svc.RT, error) {
	return &svc.RT{ID: ptr("a"), Name: ptr("n")}, nil
}

func (s *service) Show(context.Context) (*svc.UT, error) {
	return &svc.UT{ID: ptr("a"), Name: ptr("n")}, nil
}

// pass records the type of the result that it gets from the endpoint.
type pass struct {
	types []string
}

func (i *pass) Pass(ctx context.Context, info *svc.PassInfo, next loom.Endpoint) (any, error) {
	res, err := next(ctx, info.RawPayload())
	i.types = append(i.types, fmt.Sprintf("%T", res))
	return res, err
}

type serverInterceptors struct {
	*pass
	*stamp
}

// stamp appends its suffix to the ID of the result and records the ID that
// it reads.
type stamp struct {
	suffix string
	read   []string
}

func (i *stamp) Stamp(ctx context.Context, info *svc.StampInfo, next loom.Endpoint) (any, error) {
	res, err := next(ctx, info.RawPayload())
	if err != nil {
		return nil, err
	}
	r := info.Result(res)
	i.read = append(i.read, r.ID())
	r.SetID(r.ID() + i.suffix)
	return res, nil
}

func ptr[T any](v T) *T { return &v }

// bad returns res in place of the result of the endpoint, from the pass
// interceptor when inner is set and from the stamp interceptor otherwise.
type bad struct {
	inner bool
	res   any
}

func (b bad) Pass(ctx context.Context, info *svc.PassInfo, next loom.Endpoint) (any, error) {
	res, err := next(ctx, info.RawPayload())
	if b.inner {
		return b.res, err
	}
	return res, err
}

func (b bad) Stamp(ctx context.Context, info *svc.StampInfo, next loom.Endpoint) (any, error) {
	res, err := next(ctx, info.RawPayload())
	if err != nil || b.inner {
		return res, err
	}
	return b.res, nil
}

// TestInvalidResult checks that the server endpoint of a viewed result
// returns a fault error, and does not panic, when an interceptor returns a
// nil result or a result of another type to the stamp interceptor wrapper.
func TestInvalidResult(t *testing.T) {
	for _, c := range []struct {
		name string
		bad  bad
	}{
		{"stamp nil", bad{}},
		{"stamp typed nil", bad{res: (*svc.VRT)(nil)}},
		{"stamp other type", bad{res: "x"}},
		{"pass nil", bad{inner: true}},
		{"pass typed nil", bad{inner: true, res: (*svcviews.VRT)(nil)}},
		{"pass result type", bad{inner: true, res: &svc.VRT{}}},
	} {
		e := svc.NewEndpoints(&service{}, c.bad)
		res, err := e.Vget(context.Background(), nil)
		var serr *loom.ServiceError
		if !errors.As(err, &serr) || !serr.Fault {
			t.Errorf("%s: got result %#v and error %v, want a fault error", c.name, res, err)
		}
	}
}

func TestHTTPRoundTrip(t *testing.T) {
	s := &service{}
	si := &stamp{suffix: "-server"}
	pi := &pass{}
	ci := &stamp{suffix: "-client"}
	mux := loomhttp.NewMuxer()
	svcserver.Mount(mux, svcserver.New(svc.NewEndpoints(s, serverInterceptors{pi, si}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := svcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := svc.NewClient(c.Vget(), c.Get(), c.Show(), ci)
	ctx := context.Background()

	for _, view := range []string{"default", "tiny"} {
		s.view = view
		want := &svc.VRT{ID: ptr("a-server-client"), Name: ptr("n")}
		if view == "tiny" {
			want.Name = nil
		}
		got, err := client.Vget(ctx)
		if err != nil {
			t.Fatalf("%s: %v", view, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: vget: got %#v, want %#v", view, got, want)
		}
	}
	got, err := client.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := (&svc.RT{ID: ptr("a-server-client"), Name: ptr("n")}); !reflect.DeepEqual(got, want) {
		t.Errorf("get: got %#v, want %#v", got, want)
	}
	shown, err := client.Show(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if want := (&svc.UT{ID: ptr("a-server-client"), Name: ptr("n")}); !reflect.DeepEqual(shown, want) {
		t.Errorf("show: got %#v, want %#v", shown, want)
	}
	for _, c := range []struct {
		name string
		got  []string
		want []string
	}{
		{"server", si.read, []string{"a", "a", "a", "a"}},
		{"client", ci.read, []string{"a-server", "a-server", "a-server", "a-server"}},
		{"pass", pi.types, []string{"*views.VRT", "*views.VRT", "*views.RT", "*svc.UT"}},
	} {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s interceptor read %v, want %v", c.name, c.got, c.want)
		}
	}
}
`
