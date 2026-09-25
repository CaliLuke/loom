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

// TestNamedUnionStreamsAndViewsCompile generates the code of a service whose
// methods stream a named union as payload and result, and the service and
// HTTP code of a service whose methods return a collection of result types
// with views that hold the union and a result type that holds another one.
// It vets the module and runs a harness that converts the results through
// every view, validates the projected types and round-trips them over HTTP.
func TestNamedUnionStreamsAndViewsCompile(t *testing.T) {
	const module = "example.com/namedunionviews"
	root := codegen.RunDSL(t, namedUnionStreamsAndViewsDSL)
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
	goMod := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harness_test.go"), []byte(namedUnionStreamsAndViewsHarness), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "-race", "-count=1", ".")
	require.NoError(t, err, output)
}

func namedUnionStreamsAndViewsDSL() {
	dsl.API("namedunionviews", func() {})
	leaf := dsl.Type("Leaf", func() {
		dsl.Attribute("name", dsl.String)
		dsl.Required("name")
	})
	other := dsl.Type("Other", func() {
		dsl.Attribute("count", dsl.Int)
	})
	choice := dsl.Type("Choice", dsl.OneOf(leaf, other))
	holder := dsl.Type("Holder", func() {
		dsl.Attribute("choice", choice)
	})
	rt := dsl.ResultType("application/vnd.rt", "RT", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Attribute("c", choice)
		dsl.Required("id")
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
		dsl.Attribute("child", rt, func() {
			dsl.View("tiny")
		})
		dsl.View("default", func() {
			dsl.Attribute("name")
			dsl.Attribute("child")
		})
	})
	// The streams have no transport: HTTP WebSocket streams of a named union
	// payload are a separate issue.
	dsl.Service("stream", func() {
		dsl.Method("talk", func() {
			dsl.StreamingPayload(choice)
			dsl.StreamingResult(holder)
		})
		dsl.Method("watch", func() {
			dsl.StreamingResult(choice)
		})
	})
	dsl.Service("svc", func() {
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
		dsl.Method("create", func() {
			dsl.Payload(rt)
			dsl.HTTP(func() {
				dsl.POST("/create")
			})
		})
	})
}

const namedUnionStreamsAndViewsHarness = `package namedunionviews

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	svcclient "example.com/namedunionviews/gen/http/svc/client"
	svcserver "example.com/namedunionviews/gen/http/svc/server"
	stream "example.com/namedunionviews/gen/stream"
	svc "example.com/namedunionviews/gen/svc"
	"example.com/namedunionviews/gen/svc/views"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
	list    svc.RTCollection
	parent  *svc.Parent
	created *svc.RT
}

func (s *service) List(context.Context) (svc.RTCollection, string, error) {
	return s.list, "default", nil
}

func (s *service) Family(context.Context) (*svc.Parent, error) { return s.parent, nil }

func (s *service) Create(_ context.Context, p *svc.RT) error {
	s.created = p
	return nil
}

// talkStream and watchStream check that the stream interfaces carry the
// service union itself.
type (
	talkStream interface {
		stream.TalkServerStream
		Recv() (*stream.Choice, error)
		Send(*stream.Holder) error
	}
	watchStream interface {
		stream.WatchServerStream
		Send(*stream.Choice) error
	}
)

var (
	_ = talkStream(nil)
	_ = watchStream(nil)
)

func ptr[T any](v T) *T { return &v }

func collection() svc.RTCollection {
	return svc.RTCollection{
		{ID: "a", C: ptr(svc.NewChoiceLeaf(&svc.Leaf{Name: "leaf"}))},
		{ID: "b", C: ptr(svc.NewChoiceOther(&svc.Other{Count: ptr(2)}))},
	}
}

func TestViews(t *testing.T) {
	res := collection()
	cases := []struct {
		view string
		want svc.RTCollection
	}{
		{"default", res},
		{"tiny", svc.RTCollection{{ID: "a"}, {ID: "b"}}},
	}
	for _, c := range cases {
		vres, err := svc.NewViewedRTCollection(res, c.view)
		if err != nil {
			t.Fatalf("%s: %v", c.view, err)
		}
		if err := views.ValidateRTCollection(vres); err != nil {
			t.Errorf("%s: validate: %v", c.view, err)
		}
		got, err := svc.NewRTCollection(vres)
		if err != nil {
			t.Fatalf("%s: %v", c.view, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %#v, want %#v", c.view, got, c.want)
		}
	}

	invalid := svc.RTCollection{{ID: "a", C: ptr(svc.NewChoiceLeaf(&svc.Leaf{}))}}
	vres, err := svc.NewViewedRTCollection(invalid, "default")
	if err != nil {
		t.Fatal(err)
	}
	vres.Projected[0].C = ptr(views.NewChoiceViewLeaf(&views.LeafView{}))
	if err := views.ValidateRTCollection(vres); err == nil || !strings.Contains(err.Error(), "name") {
		t.Errorf("invalid branch: got %v, want a missing name error", err)
	}

	parent := &svc.Parent{Name: ptr("p"), Child: &svc.RT{ID: "c", C: ptr(svc.NewChoiceOther(&svc.Other{}))}}
	vparent, err := svc.NewViewedParent(parent, "default")
	if err != nil {
		t.Fatal(err)
	}
	if vparent.Projected.Child.C != nil {
		t.Errorf("tiny child view kept the union: %#v", vparent.Projected.Child.C)
	}
	gotParent, err := svc.NewParent(vparent)
	if err != nil {
		t.Fatal(err)
	}
	wantParent := &svc.Parent{Name: ptr("p"), Child: &svc.RT{ID: "c"}}
	if !reflect.DeepEqual(gotParent, wantParent) {
		t.Errorf("parent: got %#v, want %#v", gotParent, wantParent)
	}
}

func TestHTTPRoundTrip(t *testing.T) {
	s := &service{
		list:   collection(),
		parent: &svc.Parent{Name: ptr("p"), Child: &svc.RT{ID: "c"}},
	}
	mux := loomhttp.NewMuxer()
	svcserver.Mount(mux, svcserver.New(svc.NewEndpoints(s), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := svcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := svc.NewClient(c.List(), c.Family(), c.Create())
	ctx := context.Background()

	list, err := client.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(list, s.list) {
		t.Errorf("list: got %#v, want %#v", list, s.list)
	}
	parent, err := client.Family(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parent, s.parent) {
		t.Errorf("parent: got %#v, want %#v", parent, s.parent)
	}
	created := &svc.RT{ID: "n", C: ptr(svc.NewChoiceLeaf(&svc.Leaf{Name: "x"}))}
	if err := client.Create(ctx, created); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.created, created) {
		t.Errorf("create: got %#v, want %#v", s.created, created)
	}
}
`
