package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestResultTypeDefaultClientIntegration checks that the HTTP client of a
// result type with defaulted attributes compiles and passes go vet, that a
// result round trips through the generated server and client, and that the
// client assigns the defaults of the attributes that a response omits.
func TestResultTypeDefaultClientIntegration(t *testing.T) {
	const modulePath = "example.com/resultdefaultit"

	root := RunHTTPDSL(t, resultTypeDefaultDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	harness := filepath.Join(dir, "integration_test.go")
	require.NoError(t, os.WriteFile(harness, []byte(resultTypeDefaultHarness), 0o644))

	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}

// resultTypeDefaultDSL declares an HTTP method whose result type has a
// defaulted primitive, string and array attribute.
func resultTypeDefaultDSL() {
	var Counter = ResultType("application/vnd.counter", func() {
		TypeName("Counter")
		Attributes(func() {
			Attribute("count", Int, func() {
				Default(3)
			})
			Attribute("label", String, func() {
				Default("none")
			})
			Attribute("tags", ArrayOf(String), func() {
				Default([]string{"a"})
			})
		})
	})
	Service("counter", func() {
		Method("show", func() {
			Result(Counter)
			HTTP(func() {
				GET("/")
			})
		})
	})
}

const resultTypeDefaultHarness = `package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"

	counter "example.com/resultdefaultit/gen/counter"
	counterclient "example.com/resultdefaultit/gen/http/counter/client"
	counterserver "example.com/resultdefaultit/gen/http/counter/server"
)

type service struct{}

func (service) Show(context.Context) (*counter.Counter, error) {
	return &counter.Counter{Count: 5, Label: "five", Tags: []string{"x", "y"}}, nil
}

func show(t *testing.T, srv *httptest.Server) *counter.Counter {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	c := counterclient.NewClient(u.Scheme, u.Host, srv.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	res, err := c.Show()(context.Background(), nil)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	return res.(*counter.Counter)
}

func TestRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	counterserver.New(counter.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil).Mount(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	res := show(t, srv)
	if res.Count != 5 || res.Label != "five" || !slices.Equal(res.Tags, []string{"x", "y"}) {
		t.Errorf("result = %#v, want count 5, label five, tags [x y]", res)
	}
}

func TestOmittedAttributesTakeDefaults(t *testing.T) {
	cases := map[string]struct {
		Body  string
		Count int
		Label string
		Tags  []string
	}{
		"empty":   {Body: "{}", Count: 3, Label: "none", Tags: []string{"a"}},
		"partial": {Body: ` + "`" + `{"label":"set"}` + "`" + `, Count: 3, Label: "set", Tags: []string{"a"}},
		"zero":    {Body: ` + "`" + `{"count":0,"tags":[]}` + "`" + `, Count: 0, Label: "none", Tags: []string{}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(c.Body)); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer srv.Close()

			res := show(t, srv)
			if res.Count != c.Count || res.Label != c.Label || !slices.Equal(res.Tags, c.Tags) {
				t.Errorf("result = %#v, want count %d, label %q, tags %v", res, c.Count, c.Label, c.Tags)
			}
		})
	}
}
`
