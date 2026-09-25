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

// TestCustomizedResultCopiesCompile generates the service and HTTP code of a
// service whose methods return a result type with views, two of them through
// copies that customize its requiredness. It vets the module and runs a
// harness that converts each result through its views and round-trips them
// over HTTP.
func TestCustomizedResultCopiesCompile(t *testing.T) {
	const module = "example.com/customizedresults"
	root := codegen.RunDSL(t, customizedResultCopiesDSL)
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harness_test.go"), []byte(customizedResultCopiesHarness), 0o600))
	output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "vet", "./...")
	require.NoError(t, err, output)
	output, err = testingx.RunCmd(dir, "go", "test", "-count=1", ".")
	require.NoError(t, err, output)
}

func customizedResultCopiesDSL() {
	dsl.API("customizedresults", func() {})
	menu := dsl.ResultType("application/vnd.menu", "Menu", func() {
		dsl.Attributes(func() {
			dsl.Attribute("x", dsl.String)
			dsl.Attribute("y", dsl.String)
		})
		dsl.View("default", func() {
			dsl.Attribute("x")
			dsl.Attribute("y")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("x")
		})
	})
	dsl.Service("svc", func() {
		dsl.Method("m1", func() {
			dsl.Result(menu, func() {
				dsl.Required("x")
			})
			dsl.HTTP(func() {
				dsl.GET("/m1")
			})
		})
		dsl.Method("m2", func() {
			dsl.Result(menu, func() {
				dsl.Required("x")
				dsl.View("tiny")
			})
			dsl.HTTP(func() {
				dsl.GET("/m2")
			})
		})
		dsl.Method("m3", func() {
			dsl.Result(menu)
			dsl.HTTP(func() {
				dsl.GET("/m3")
			})
		})
	})
}

const customizedResultCopiesHarness = `package customizedresults

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	svcclient "example.com/customizedresults/gen/http/svc/client"
	svcserver "example.com/customizedresults/gen/http/svc/server"
	svc "example.com/customizedresults/gen/svc"
	loomhttp "github.com/CaliLuke/loom/http"
)

type service struct{}

func (service) M1(context.Context) (*svc.MenuM1Result, string, error) {
	return &svc.MenuM1Result{X: "one", Y: ptr("y1")}, "default", nil
}

func (service) M2(context.Context) (*svc.MenuM2Result, error) {
	return &svc.MenuM2Result{X: "two", Y: ptr("y2")}, nil
}

func (service) M3(context.Context) (*svc.Menu, string, error) {
	return &svc.Menu{Y: ptr("y3")}, "tiny", nil
}

func ptr[T any](v T) *T { return &v }

func TestViews(t *testing.T) {
	v1, err := svc.NewViewedMenuM1Result(&svc.MenuM1Result{X: "one", Y: ptr("y1")}, "tiny")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.NewMenuM1Result(v1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, &svc.MenuM1Result{X: "one"}) {
		t.Errorf("m1: got %#v", got)
	}
	v2, err := svc.NewViewedMenuM2Result(&svc.MenuM2Result{X: "two", Y: ptr("y2")}, "tiny")
	if err != nil {
		t.Fatal(err)
	}
	got2, err := svc.NewMenuM2Result(v2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got2, &svc.MenuM2Result{X: "two"}) {
		t.Errorf("m2: got %#v", got2)
	}
	v3, err := svc.NewViewedMenu(&svc.Menu{X: ptr("three"), Y: ptr("y3")}, "default")
	if err != nil {
		t.Fatal(err)
	}
	got3, err := svc.NewMenu(v3)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got3, &svc.Menu{X: ptr("three"), Y: ptr("y3")}) {
		t.Errorf("m3: got %#v", got3)
	}
}

func TestHTTPRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	svcserver.Mount(mux, svcserver.New(svc.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := svcclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	client := svc.NewClient(c.M1(), c.M2(), c.M3())
	ctx := context.Background()

	m1, err := client.M1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m1, &svc.MenuM1Result{X: "one", Y: ptr("y1")}) {
		t.Errorf("m1: got %#v", m1)
	}
	m2, err := client.M2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m2, &svc.MenuM2Result{X: "two"}) {
		t.Errorf("m2: got %#v", m2)
	}
	m3, err := client.M3(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m3, &svc.Menu{}) {
		t.Errorf("m3: got %#v", m3)
	}
}
`
