package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCViewedUnaryResultGeneratedModule covers the JSON-RPC unary HTTP
// methods whose result is a result type with views or a collection of it.
// The handler renders the response body of the view that the service
// returned, as the response encoder does, and not the body of the default
// view: a method without a fixed view renders the view that the service
// selects, a method with a fixed view renders that view, and a method with
// response headers renders its view through the response encoder.
func TestJSONRPCViewedUnaryResultGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcViewedUnaryResultDSL)
	dir := t.TempDir()
	const modulePath = "example.com/jsonrpcunaryviewed"
	renderJSONRPCModule(t, dir, modulePath, root)
	data := CreateJSONRPCServices(root)
	for _, svc := range root.Services {
		if views := servicecodegen.ViewsFile(modulePath+"/gen", svc, data.ServicesData); views != nil {
			renderCodegenFiles(t, dir, []*cg.File{views})
		}
	}

	server := readGeneratedFile(t, dir, "gen/jsonrpc/notes/server/server.go")
	for _, want := range []string{
		// A method without a fixed view renders the view of the result.
		"viewedRes := res.(*notesviews.Note)\n\t\tw.Header().Set(\"loom-view\", viewedRes.View)\n\t\tvar body any\n\t\tswitch viewedRes.View {\n\t\tcase \"default\", \"\":\n\t\t\tbody = NewShowResponseBody(viewedRes.Projected)\n\t\tcase \"tiny\":\n\t\t\tbody = NewShowResponseBodyTiny(viewedRes.Projected)\n\t\t}",
		"viewedRes := res.(notesviews.NoteCollection)\n\t\tw.Header().Set(\"loom-view\", viewedRes.View)\n\t\tvar body any\n\t\tswitch viewedRes.View {\n\t\tcase \"default\", \"\":\n\t\t\tbody = NewNoteResponseCollection(viewedRes.Projected)\n\t\tcase \"tiny\":\n\t\t\tbody = NewNoteResponseTinyCollection(viewedRes.Projected)\n\t\t}",
		// A fixed view renders the body of that view alone.
		"viewedRes := res.(*notesviews.Note)\n\t\tbody := NewShowTinyResponseBodyTiny(viewedRes.Projected)",
	} {
		assertGeneratedContains(t, "server.go", server, want)
	}

	require.NoError(t, os.WriteFile(filepath.Join(dir, "viewed_test.go"), []byte(jsonRPCViewedUnaryResultHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

func jsonrpcViewedUnaryResultDSL() {
	dsl.API("unaryviewed", func() {
		dsl.JSONRPC(func() {})
	})
	note := dsl.ResultType("application/vnd.note", func() {
		dsl.TypeName("Note")
		dsl.Attributes(func() {
			dsl.Attribute("id", dsl.String)
			dsl.Attribute("title", dsl.String)
			dsl.Attribute("body", dsl.String)
			dsl.Required("id", "title")
		})
		dsl.View("default", func() {
			dsl.Attribute("id")
			dsl.Attribute("title")
			dsl.Attribute("body")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("id")
		})
	})
	dsl.Service("notes", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("show", func() {
			dsl.Payload(dsl.String)
			dsl.Result(note)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("showTiny", func() {
			dsl.Payload(dsl.String)
			dsl.Result(note, func() {
				dsl.View("tiny")
			})
			dsl.JSONRPC(func() {})
		})
		dsl.Method("list", func() {
			dsl.Payload(dsl.String)
			dsl.Result(dsl.CollectionOf(note))
			dsl.JSONRPC(func() {})
		})
		dsl.Method("showTagged", func() {
			dsl.Payload(dsl.String)
			dsl.Result(note)
			dsl.JSONRPC(func() {
				dsl.Response(func() {
					dsl.Header("id:X-Note-Id")
				})
			})
		})
	})
}

// jsonRPCViewedUnaryResultHarness calls each method with the name of the view
// that the service selects as the payload and checks the result on the wire
// and in the generated client.
const jsonRPCViewedUnaryResultHarness = `package jsonrpcunaryviewed_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	notesclient "example.com/jsonrpcunaryviewed/gen/jsonrpc/notes/client"
	notesserver "example.com/jsonrpcunaryviewed/gen/jsonrpc/notes/server"
	notes "example.com/jsonrpcunaryviewed/gen/notes"
	loomhttp "github.com/CaliLuke/loom/http"
)

const (
	fullJSON = ` + "`" + `{"id":"n1","title":"t","body":"b"}` + "`" + `
	tinyJSON = ` + "`" + `{"id":"n1"}` + "`" + `
)

func note() *notes.Note {
	body := "b"
	return &notes.Note{ID: "n1", Title: "t", Body: &body}
}

// notesService returns the note with the view named by the payload.
type notesService struct{}

func (notesService) Show(_ context.Context, view string) (*notes.Note, string, error) {
	return note(), view, nil
}

func (notesService) ShowTiny(context.Context, string) (*notes.Note, error) {
	return note(), nil
}

func (notesService) List(_ context.Context, view string) (notes.NoteCollection, string, error) {
	return notes.NoteCollection{note(), note()}, view, nil
}

func (notesService) ShowTagged(_ context.Context, view string) (*notes.Note, string, error) {
	return note(), view, nil
}

func serve(t *testing.T) *httptest.Server {
	t.Helper()
	mux := loomhttp.NewMuxer()
	errhandler := func(_ context.Context, _ http.ResponseWriter, err error) {
		t.Errorf("handler error: %v", err)
	}
	notesserver.Mount(mux, notesserver.New(notes.NewEndpoints(notesService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return hs
}

type response struct {
	ID     any            ` + "`json:\"id\"`" + `
	Result jsontext.Value ` + "`json:\"result\"`" + `
	Error  jsontext.Value ` + "`json:\"error\"`" + `
}

// TestWireViews checks the result member of the responses: the view that the
// service selects, or the view that the design fixes.
func TestWireViews(t *testing.T) {
	hs := serve(t)
	// The endpoint names the empty view default, and the attribute mapped to
	// the X-Note-Id header is not in the body.
	for i, tc := range []struct {
		method, view, want, header, loomView string
	}{
		{"show", "", fullJSON, "", "default"},
		{"show", "default", fullJSON, "", "default"},
		{"show", "tiny", tinyJSON, "", "tiny"},
		{"showTiny", "", tinyJSON, "", ""},
		{"list", "", "[" + fullJSON + "," + fullJSON + "]", "", "default"},
		{"list", "tiny", "[" + tinyJSON + "," + tinyJSON + "]", "", "tiny"},
		{"showTagged", "", ` + "`" + `{"title":"t","body":"b"}` + "`" + `, "n1", "default"},
		{"showTagged", "tiny", "{}", "n1", "tiny"},
	} {
		id := i + 1
		req := fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%d,"method":%q,"params":%q}` + "`" + `, id, tc.method, tc.view)
		resp, err := hs.Client().Post(hs.URL+"/rpc", "application/json", strings.NewReader(req))
		if err != nil {
			t.Fatalf("post %s: %v", req, err)
		}
		var got response
		err = json.UnmarshalRead(resp.Body, &got)
		if cerr := resp.Body.Close(); cerr != nil {
			t.Errorf("close body: %v", cerr)
		}
		if err != nil {
			t.Fatalf("decode %s response: %v", req, err)
		}
		if resp.StatusCode != http.StatusOK || got.ID != float64(id) || len(got.Error) != 0 || string(got.Result) != tc.want {
			t.Errorf("%s %q: got %d %+v, want result %s", tc.method, tc.view, resp.StatusCode, got, tc.want)
		}
		if h := resp.Header.Get("X-Note-Id"); h != tc.header {
			t.Errorf("%s %q: got X-Note-Id %q, want %q", tc.method, tc.view, h, tc.header)
		}
		// The loom-view header names the view that the service selects and
		// is absent when the design fixes the view.
		if h, ok := resp.Header["Loom-View"]; ok != (tc.loomView != "") || strings.Join(h, ",") != tc.loomView {
			t.Errorf("%s %q: got loom-view %q (present %t), want %q", tc.method, tc.view, h, ok, tc.loomView)
		}
	}
}

// TestClientViews checks that the generated client decodes the view that
// the server renders.
func TestClientViews(t *testing.T) {
	hs := serve(t)
	c := notesclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	full, tiny := note(), &notes.Note{ID: "n1"}
	for _, tc := range []struct {
		name string
		call func(context.Context, any) (any, error)
		view string
		want any
	}{
		{"show", c.Show(), "default", full},
		{"show", c.Show(), "tiny", tiny},
		{"showTiny", c.ShowTiny(), "x", tiny},
		{"list", c.List(), "default", notes.NoteCollection{full, full}},
		{"list", c.List(), "tiny", notes.NoteCollection{tiny, tiny}},
		{"showTagged", c.ShowTagged(), "default", full},
		{"showTagged", c.ShowTagged(), "tiny", tiny},
	} {
		got, err := tc.call(context.Background(), tc.view)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s %q: got (%#v, %v), want %#v", tc.name, tc.view, got, err, tc.want)
		}
	}
}
`
