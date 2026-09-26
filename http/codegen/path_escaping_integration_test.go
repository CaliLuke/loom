package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestPathEscapingRoundTripIntegration generates a client and a server for
// routes with segment, catch-all, array, alias, bytes, and numeric path params
// and a route literal that needs escaping. It sends adversarial path values
// through the generated client and checks that each reaches the generated
// server route, which decodes the value that the client encoded.
func TestPathEscapingRoundTripIntegration(t *testing.T) {
	const modulePath = "example.com/pathescapeit"

	root := RunHTTPDSL(t, pathEscapingRoundTripDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "path_escaping_test.go"),
		[]byte(pathEscapingRoundTripHarness),
		0o600,
	))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}

func pathEscapingRoundTripDSL() {
	code := Type("Code", String)
	Service("pathesc", func() {
		Method("item", func() {
			Payload(func() {
				Attribute("id", String)
				Required("id")
			})
			Result(String)
			HTTP(func() {
				GET("/items/{id}")
			})
		})
		Method("file", func() {
			Payload(func() {
				Attribute("id", Int)
				Attribute("path", String)
				Required("id", "path")
			})
			Result(String)
			HTTP(func() {
				GET("/items/{id}/files/{*path}")
			})
		})
		Method("tags", func() {
			Payload(func() {
				Attribute("tags", ArrayOf(String))
				Attribute("name", String)
				Required("tags", "name")
			})
			Result(ArrayOf(String))
			HTTP(func() {
				GET("/tags/{name}/{tags}")
			})
		})
		Method("alias", func() {
			Payload(func() {
				Attribute("code", code)
				Required("code")
			})
			Result(String)
			HTTP(func() {
				GET("/alias/{code}")
			})
		})
		Method("blob", func() {
			Payload(func() {
				Attribute("raw", Bytes)
				Required("raw")
			})
			Result(String)
			HTTP(func() {
				GET("/blobs/{raw}")
			})
		})
		Method("literal", func() {
			Payload(func() {
				Attribute("id", String)
				Required("id")
			})
			Result(String)
			HTTP(func() {
				GET("/my files/{id}")
			})
		})
	})
}

const pathEscapingRoundTripHarness = `package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	pathesc "example.com/pathescapeit/gen/pathesc"
	pathescclient "example.com/pathescapeit/gen/http/pathesc/client"
	pathescserver "example.com/pathescapeit/gen/http/pathesc/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

// segmentValues change the route or lose their meaning when they are
// formatted into a path without escaping.
var segmentValues = []string{
	"plain",
	"a/b",
	"/leading",
	"trailing/",
	"100%",
	"%2F",
	"%zz",
	"a b",
	"a+b",
	"日本語",
	".",
	"..",
	"a/../b",
	"?q=1",
	"#frag",
	"a;b",
	"@:=&$",
	"a//b",
}

type pathService struct {
	calls int
}

func (s *pathService) Item(_ context.Context, p *pathesc.ItemPayload) (string, error) {
	s.calls++
	return p.ID, nil
}

func (s *pathService) File(_ context.Context, p *pathesc.FilePayload) (string, error) {
	s.calls++
	return fmt.Sprintf("%d:%s", p.ID, p.Path), nil
}

func (s *pathService) Tags(_ context.Context, p *pathesc.TagsPayload) ([]string, error) {
	s.calls++
	return append([]string{p.Name}, p.Tags...), nil
}

func (s *pathService) Alias(_ context.Context, p *pathesc.AliasPayload) (string, error) {
	s.calls++
	return string(p.Code), nil
}

func (s *pathService) Blob(_ context.Context, p *pathesc.BlobPayload) (string, error) {
	s.calls++
	return string(p.Raw), nil
}

func (s *pathService) Literal(_ context.Context, p *pathesc.LiteralPayload) (string, error) {
	s.calls++
	return p.ID, nil
}

func TestGeneratedPathsRoundTripAdversarialValues(t *testing.T) {
	service, client, doer := newPathClient(t)
	for _, value := range segmentValues {
		t.Run(value, func(t *testing.T) {
			before := service.calls
			got, err := client.Item()(t.Context(), &pathesc.ItemPayload{ID: value})
			if err != nil {
				t.Fatalf("Item(%q): %v", value, err)
			}
			if got != value {
				t.Errorf("Item(%q) decoded %q", value, got)
			}
			if want := "/items/" + loomhttp.EscapePathSegment(value); doer.escapedPath != want {
				t.Errorf("Item(%q) sent path %q, want %q", value, doer.escapedPath, want)
			}
			if doer.status != http.StatusOK {
				t.Errorf("Item(%q) status = %d, want %d", value, doer.status, http.StatusOK)
			}

			got, err = client.File()(t.Context(), &pathesc.FilePayload{ID: 7, Path: value})
			if err != nil {
				t.Fatalf("File(%q): %v", value, err)
			}
			if want := "7:" + value; got != want {
				t.Errorf("File(%q) decoded %q, want %q", value, got, want)
			}

			got, err = client.Alias()(t.Context(), &pathesc.AliasPayload{Code: pathesc.Code(value)})
			if err != nil {
				t.Fatalf("Alias(%q): %v", value, err)
			}
			if got != value {
				t.Errorf("Alias(%q) decoded %q", value, got)
			}

			got, err = client.Blob()(t.Context(), &pathesc.BlobPayload{Raw: []byte(value)})
			if err != nil {
				t.Fatalf("Blob(%q): %v", value, err)
			}
			if got != value {
				t.Errorf("Blob(%q) decoded %q", value, got)
			}
			if calls := service.calls - before; calls != 4 {
				t.Errorf("service calls = %d, want 4", calls)
			}
		})
	}
}

// TestGeneratedPathsRoundTripEscapedLiteral checks that the client sends the
// escaped literal of a route with a space. It uses values whose escaped path
// equals the default escaping of the path: the muxer matches a path with
// other escapes, such as an escaped "/", against the unescaped route literal
// and finds no route.
func TestGeneratedPathsRoundTripEscapedLiteral(t *testing.T) {
	_, client, doer := newPathClient(t)
	for _, value := range []string{"plain", "a b", "100%", "日本語", "%2F"} {
		t.Run(value, func(t *testing.T) {
			got, err := client.Literal()(t.Context(), &pathesc.LiteralPayload{ID: value})
			if err != nil {
				t.Fatalf("Literal(%q): %v", value, err)
			}
			if got != value {
				t.Errorf("Literal(%q) decoded %q", value, got)
			}
			if want := "/my%20files/" + loomhttp.EscapePathSegment(value); doer.escapedPath != want {
				t.Errorf("Literal(%q) sent path %q, want %q", value, doer.escapedPath, want)
			}
		})
	}
}

func TestGeneratedPathsRoundTripCatchAllRemainders(t *testing.T) {
	_, client, doer := newPathClient(t)
	for _, value := range []string{"a/b/c", "a/../b", "../etc/passwd", "a//b", "/lead", "tail/", "x y/100%/日本", "./a/."} {
		t.Run(value, func(t *testing.T) {
			got, err := client.File()(t.Context(), &pathesc.FilePayload{ID: 1, Path: value})
			if err != nil {
				t.Fatalf("File(%q): %v", value, err)
			}
			if want := "1:" + value; got != want {
				t.Errorf("File(%q) decoded %q, want %q", value, got, want)
			}
			if want := "/items/1/files/" + loomhttp.EscapePathRemainder(value); doer.escapedPath != want {
				t.Errorf("File(%q) sent path %q, want %q", value, doer.escapedPath, want)
			}
		})
	}
}

func TestGeneratedPathsRoundTripArrayElements(t *testing.T) {
	_, client, _ := newPathClient(t)
	tags := []string{"a/b", "100%", "a b", "..", "日本"}
	got, err := client.Tags()(t.Context(), &pathesc.TagsPayload{Name: "x/y", Tags: tags})
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	decoded, ok := got.([]string)
	if !ok {
		t.Fatalf("Tags result type %T, want []string", got)
	}
	if want := append([]string{"x/y"}, tags...); !slices.Equal(decoded, want) {
		t.Errorf("Tags decoded %q, want %q", got, want)
	}
}

// TestUnescapedSlashChangesRoute shows the failure that the escaping
// prevents: formatting a value with "/" into the path without escaping
// reaches no route.
func TestUnescapedSlashChangesRoute(t *testing.T) {
	_, server := newPathServer(t)
	u := &url.URL{Path: "/items/" + "a/b"}
	resp, err := server.Client().Get(server.URL + u.String())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close body: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unescaped slash status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

type recordingDoer struct {
	base        loomhttp.Doer
	escapedPath string
	status      int
}

func (d *recordingDoer) Do(req *http.Request) (*http.Response, error) {
	d.escapedPath = req.URL.EscapedPath()
	resp, err := d.base.Do(req)
	if err == nil {
		d.status = resp.StatusCode
	}
	return resp, err
}

func newPathServer(t *testing.T) (*pathService, *httptest.Server) {
	t.Helper()
	service := &pathService{}
	mux := loomhttp.NewMuxer()
	transport := pathescserver.New(pathesc.NewEndpoints(service), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	pathescserver.Mount(mux, transport)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return service, server
}

func newPathClient(t *testing.T) (*pathService, *pathescclient.Client, *recordingDoer) {
	t.Helper()
	service, server := newPathServer(t)
	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	doer := &recordingDoer{base: server.Client()}
	client := pathescclient.NewClient(u.Scheme, u.Host, doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return service, client, doer
}
`
