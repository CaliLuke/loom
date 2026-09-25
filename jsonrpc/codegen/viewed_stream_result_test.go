package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
)

// TestJSONRPCViewedStreamResultGeneratedModule covers the JSON-RPC WebSocket
// and SSE methods whose result or streaming result is a result type with
// views, a collection of it, or a view that the design fixes. The server
// projects each result with its view before it renders the response body of
// that view: the view set with SetView on a WebSocket method stream, the
// fixed view, or the default view for the streams without SetView and the
// streaming-payload methods that return their result.
func TestJSONRPCViewedStreamResultGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcViewedStreamResultDSL)
	dir := t.TempDir()
	const modulePath = "example.com/jsonrpcviewed"
	renderJSONRPCModule(t, dir, modulePath, root)
	data := CreateJSONRPCServices(root)
	for _, svc := range root.Services {
		if views := servicecodegen.ViewsFile(modulePath+"/gen", svc, data.ServicesData); views != nil {
			renderCodegenFiles(t, dir, []*cg.File{views})
		}
	}

	ws := readGeneratedFile(t, dir, "gen/jsonrpc/files/server/websocket.go")
	for _, want := range []string{
		// The method streams render with the view set with SetView.
		"func (w *talkStreamWrapper) SetView(view string) {",
		"vres, err := files.NewViewedNote(res, w.view)",
		"vres, err := files.NewViewedNoteCollection(res, w.view)",
		// The connection stream renders with the default view.
		"vres, err := files.NewViewedNote(result, \"default\")",
		// A fixed view renders with the body of that view.
		"vres, err := files.NewViewedNote(result, \"tiny\")",
		"body := NewTalkTinyResponseBodyTiny(vres.Projected)",
		"case \"tiny\":\n\t\tbody = NewTalkResponseBodyTiny(vres.Projected)",
	} {
		assertGeneratedContains(t, "websocket.go", ws, want)
	}
	if strings.Contains(ws, "func (w *talkTinyStreamWrapper) SetView") {
		t.Error("websocket.go: the stream of a method with a fixed view has SetView")
	}
	stream := readGeneratedFile(t, dir, "gen/jsonrpc/feed/server/stream.go")
	for _, want := range []string{
		"vres, err := feed.NewViewedNote(result, \"default\")",
		"vres, err := feed.NewViewedNote(result, \"tiny\")",
		"vres, err := feed.NewViewedNoteCollection(result, \"default\")",
	} {
		assertGeneratedContains(t, "stream.go", stream, want)
	}
	assertGeneratedContains(t, "sse.go", readGeneratedFile(t, dir, "gen/jsonrpc/feed/server/sse.go"), "vres, err := feed.NewViewedNote(v, \"default\")")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "viewed_test.go"), []byte(jsonRPCViewedStreamResultHarness), 0o600))
	serverDir := filepath.Join(dir, "gen", "jsonrpc", "feed", "server")
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, "service_stream_test.go"), []byte(jsonRPCViewedSSEServiceStreamHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

// readGeneratedFile returns the content of the file at the slash-separated
// path rel under dir.
func readGeneratedFile(t *testing.T, dir, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	require.NoError(t, err)
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

// assertGeneratedContains reports an error when the generated code of the
// file name does not contain want.
func assertGeneratedContains(t *testing.T, name, code, want string) {
	t.Helper()
	if !strings.Contains(code, want) {
		t.Errorf("%s does not contain %q", name, want)
	}
}

func jsonrpcViewedStreamResultDSL() {
	dsl.API("viewed", func() {
		dsl.JSONRPC(func() {})
	})
	note := dsl.ResultType("application/vnd.note", func() {
		dsl.TypeName("Note")
		dsl.Attributes(func() {
			dsl.Attribute("id", dsl.String)
			dsl.Attribute("title", dsl.String)
			dsl.Attribute("body", dsl.String)
			dsl.Required("id")
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
	tiny := func() {
		dsl.View("tiny")
	}
	dsl.Service("files", func() {
		dsl.JSONRPC(func() {
			dsl.GET("/ws")
		})
		dsl.Method("upload", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(note)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("uploadTiny", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(note, tiny)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("uploadList", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.Result(dsl.CollectionOf(note))
			dsl.JSONRPC(func() {})
		})
		dsl.Method("talk", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.StreamingResult(note)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("talkTiny", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.StreamingResult(note, tiny)
			dsl.JSONRPC(func() {})
		})
		dsl.Method("talkList", func() {
			dsl.StreamingPayload(dsl.String)
			dsl.StreamingResult(dsl.CollectionOf(note))
			dsl.JSONRPC(func() {})
		})
		dsl.Method("watch", func() {
			dsl.Payload(dsl.String)
			dsl.StreamingResult(note)
			dsl.JSONRPC(func() {})
		})
	})
	dsl.Service("feed", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/feed")
		})
		dsl.Method("follow", func() {
			dsl.Payload(dsl.String)
			dsl.StreamingResult(note)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
		dsl.Method("followTiny", func() {
			dsl.Payload(dsl.String)
			dsl.StreamingResult(note, tiny)
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
		dsl.Method("followList", func() {
			dsl.Payload(dsl.String)
			dsl.StreamingResult(dsl.CollectionOf(note))
			dsl.JSONRPC(func() {
				dsl.ServerSentEvents()
			})
		})
	})
}

// jsonRPCViewedSSEServiceStreamHarness sends a viewed result on the
// service-level SSE stream, which has no view to select and renders the
// default view.
const jsonRPCViewedSSEServiceStreamHarness = `package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	feed "example.com/jsonrpcviewed/gen/feed"
	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

func TestServiceStreamSendsDefaultView(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/feed", nil)
	s := &feedSSEStream{
		w:      rec,
		r:      req,
		writer: loomhttp.NewSSEStreamWriter(rec, req.Context(), loomtransport.TransportJSONRPC, loomhttp.StreamWritePolicy{}),
	}
	title, body := "t", "b"
	if err := s.Send(context.Background(), &feed.Note{ID: "n1", Title: &title, Body: &body}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if want := ` + "`" + `"params":{"id":"n1","title":"t","body":"b"}` + "`" + `; !strings.Contains(rec.Body.String(), want) {
		t.Errorf("event %q does not contain %s", rec.Body.String(), want)
	}
}
`

const jsonRPCViewedStreamResultHarness = `package jsonrpcviewed_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	feed "example.com/jsonrpcviewed/gen/feed"
	files "example.com/jsonrpcviewed/gen/files"
	feedclient "example.com/jsonrpcviewed/gen/jsonrpc/feed/client"
	feedserver "example.com/jsonrpcviewed/gen/jsonrpc/feed/server"
	filesclient "example.com/jsonrpcviewed/gen/jsonrpc/files/client"
	filesserver "example.com/jsonrpcviewed/gen/jsonrpc/files/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

const (
	fullJSON = ` + "`" + `{"id":"n1","title":"t","body":"b"}` + "`" + `
	tinyJSON = ` + "`" + `{"id":"n1"}` + "`" + `
)

func note() *files.Note {
	title, body := "t", "b"
	return &files.Note{ID: "n1", Title: &title, Body: &body}
}

func feedNote() *feed.Note {
	title, body := "t", "b"
	return &feed.Note{ID: "n1", Title: &title, Body: &body}
}

var (
	full = note()
	tiny = &files.Note{ID: "n1"}
)

// filesService sets the view named by the payload of a method stream, when
// not empty, and sends the note as a notification and as the response.
type filesService struct {
	// greet sends a notification on the connection stream first.
	greet bool
}

func (s filesService) HandleStream(ctx context.Context, stream files.Stream) error {
	if s.greet {
		if err := stream.SendTalkNotification(ctx, note()); err != nil {
			return err
		}
	}
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}

func (filesService) Upload(context.Context, string) (*files.Note, error) {
	return note(), nil
}

func (filesService) UploadTiny(context.Context, string) (*files.Note, error) {
	return note(), nil
}

func (filesService) UploadList(context.Context, string) (files.NoteCollection, error) {
	return files.NoteCollection{note(), note()}, nil
}

func (filesService) Talk(ctx context.Context, p string, st files.TalkServerStream) error {
	if p != "" {
		st.SetView(p)
	}
	if err := st.SendNotification(ctx, note()); err != nil {
		return err
	}
	return st.SendResponse(ctx, note())
}

func (filesService) TalkTiny(ctx context.Context, _ string, st files.TalkTinyServerStream) error {
	if err := st.SendNotification(ctx, note()); err != nil {
		return err
	}
	return st.SendResponse(ctx, note())
}

func (filesService) TalkList(ctx context.Context, p string, st files.TalkListServerStream) error {
	if p != "" {
		st.SetView(p)
	}
	return st.SendResponse(ctx, files.NoteCollection{note(), note()})
}

func (filesService) Watch(ctx context.Context, p string, st files.WatchServerStream) error {
	if p != "" {
		st.SetView(p)
	}
	return st.SendResponse(ctx, note())
}

type feedService struct{}

func (feedService) Follow(ctx context.Context, _ string, st feed.FollowServerStream) error {
	if err := st.Send(ctx, feedNote()); err != nil {
		return err
	}
	return st.SendAndClose(ctx, feedNote())
}

func (feedService) FollowTiny(ctx context.Context, _ string, st feed.FollowTinyServerStream) error {
	if err := st.Send(ctx, feedNote()); err != nil {
		return err
	}
	return st.SendAndClose(ctx, feedNote())
}

func (feedService) FollowList(ctx context.Context, _ string, st feed.FollowListServerStream) error {
	if err := st.Send(ctx, feed.NoteCollection{feedNote()}); err != nil {
		return err
	}
	return st.SendAndClose(ctx, feed.NoteCollection{feedNote(), feedNote()})
}

func serveFiles(t *testing.T, svc filesService) string {
	t.Helper()
	mux := loomhttp.NewMuxer()
	filesserver.Mount(mux, filesserver.New(svc.HandleStream, files.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return strings.TrimPrefix(hs.URL, "http://")
}

func newFilesClient(t *testing.T) *filesclient.Client {
	t.Helper()
	host := serveFiles(t, filesService{})
	c := filesclient.NewClient("ws", host, nil, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})
	return c
}

// exchange opens a stream with open, sends p and returns the response that
// recv receives.
func exchange[S interface {
	SendWithContext(context.Context, string) error
	Close() error
}, R any](t *testing.T, ctx context.Context, open func(context.Context, any) (any, error), p string, recv func(S, context.Context) (R, error)) (R, error) {
	t.Helper()
	res, err := open(ctx, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	stream := res.(S)
	defer func() {
		if err := stream.Close(); err != nil {
			t.Errorf("close stream: %v", err)
		}
	}()
	if err := stream.SendWithContext(ctx, p); err != nil {
		t.Fatalf("send %q: %v", p, err)
	}
	return recv(stream, ctx)
}

func TestWebSocketClientViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := newFilesClient(t)
	check := func(name string, got any, err error, want any) {
		t.Helper()
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got (%#v, %v), want %#v", name, got, err, want)
		}
	}

	got, err := exchange(t, ctx, c.Upload(), "x", (*filesclient.UploadClientStream).CloseAndRecvWithContext)
	check("upload", got, err, full)
	got, err = exchange(t, ctx, c.UploadTiny(), "x", (*filesclient.UploadTinyClientStream).CloseAndRecvWithContext)
	check("uploadTiny", got, err, tiny)
	list, err := exchange(t, ctx, c.UploadList(), "x", (*filesclient.UploadListClientStream).CloseAndRecvWithContext)
	check("uploadList", list, err, files.NoteCollection{full, full})

	for _, tc := range []struct {
		view string
		want *files.Note
	}{{"", full}, {"default", full}, {"tiny", tiny}} {
		got, err = exchange(t, ctx, c.Talk(), tc.view, (*filesclient.TalkClientStream).RecvWithContext)
		check("talk "+tc.view, got, err, tc.want)
		got, err = exchange(t, ctx, c.Watch(), tc.view, (*filesclient.WatchClientStream).RecvWithContext)
		check("watch "+tc.view, got, err, tc.want)
	}
	got, err = exchange(t, ctx, c.TalkTiny(), "x", (*filesclient.TalkTinyClientStream).RecvWithContext)
	check("talkTiny", got, err, tiny)
	list, err = exchange(t, ctx, c.TalkList(), "tiny", (*filesclient.TalkListClientStream).RecvWithContext)
	check("talkList tiny", list, err, files.NoteCollection{tiny, tiny})
	list, err = exchange(t, ctx, c.TalkList(), "", (*filesclient.TalkListClientStream).RecvWithContext)
	check("talkList", list, err, files.NoteCollection{full, full})

	// A view that the result type does not define fails the method.
	if got, err := exchange(t, ctx, c.Talk(), "bogus", (*filesclient.TalkClientStream).RecvWithContext); err == nil {
		t.Errorf("talk bogus: got %#v, want an error", got)
	}
}

type frame struct {
	ID     any            ` + "`json:\"id\"`" + `
	Method string         ` + "`json:\"method\"`" + `
	Params jsontext.Value ` + "`json:\"params\"`" + `
	Result jsontext.Value ` + "`json:\"result\"`" + `
	Error  *struct {
		Message string ` + "`json:\"message\"`" + `
	} ` + "`json:\"error\"`" + `
}

// TestWebSocketWireViews checks the notifications and responses on the wire:
// the connection stream renders the default view, and a method stream the
// view set with SetView or the fixed view.
func TestWebSocketWireViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	host := serveFiles(t, filesService{greet: true})
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws://"+host+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close handshake body: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil && !errors.Is(err, websocket.ErrCloseSent) {
			t.Errorf("close conn: %v", err)
		}
	}()
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	read := func() frame {
		t.Helper()
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var f frame
		if err := json.Unmarshal(data, &f); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
		return f
	}
	if f := read(); f.ID != nil || f.Method != "talk" || string(f.Params) != fullJSON {
		t.Errorf("greeting: got %+v, want a talk notification with %s", f, fullJSON)
	}
	for i, tc := range []struct {
		method, view string
		notification bool
		want         string
	}{
		{"talk", "", true, fullJSON},
		{"talk", "tiny", true, tinyJSON},
		{"talkTiny", "", true, tinyJSON},
		{"watch", "tiny", false, tinyJSON},
		{"talkList", "tiny", false, "[" + tinyJSON + "," + tinyJSON + "]"},
		{"upload", "", false, fullJSON},
		{"uploadTiny", "", false, tinyJSON},
	} {
		id := i + 1
		req := fmt.Sprintf(` + "`" + `{"jsonrpc":"2.0","id":%d,"method":%q,"params":%q}` + "`" + `, id, tc.method, tc.view)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(req)); err != nil {
			t.Fatalf("write %s: %v", req, err)
		}
		if tc.notification {
			if f := read(); f.ID != nil || f.Method != tc.method || string(f.Params) != tc.want {
				t.Errorf("%s %q notification: got %+v, want %s", tc.method, tc.view, f, tc.want)
			}
		}
		if f := read(); f.ID != float64(id) || f.Error != nil || string(f.Result) != tc.want {
			t.Errorf("%s %q response: got %+v, want %s", tc.method, tc.view, f, tc.want)
		}
	}
	req := ` + "`" + `{"jsonrpc":"2.0","id":99,"method":"talk","params":"bogus"}` + "`" + `
	if err := conn.WriteMessage(websocket.TextMessage, []byte(req)); err != nil {
		t.Fatalf("write %s: %v", req, err)
	}
	if f := read(); f.ID != float64(99) || f.Error == nil || len(f.Result) != 0 {
		t.Errorf("talk bogus: got %+v, want an error response", f)
	}
}

func TestSSEViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	feedserver.Mount(mux, feedserver.New(feed.NewEndpoints(feedService{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler))
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := feedclient.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	fullNote, tinyNote := feedNote(), &feed.Note{ID: "n1"}

	recvAll := func(name string, open func(context.Context, any) (any, error), recv func(any, context.Context) (any, error), want []any) {
		t.Helper()
		res, err := open(ctx, "x")
		if err != nil {
			t.Fatalf("%s: open: %v", name, err)
		}
		for i, w := range want {
			got, err := recv(res, ctx)
			if err != nil || !reflect.DeepEqual(got, w) {
				t.Errorf("%s: recv %d: got (%#v, %v), want %#v", name, i, got, err, w)
			}
		}
		if _, err := recv(res, ctx); !errors.Is(err, io.EOF) {
			t.Errorf("%s: got %v, want io.EOF", name, err)
		}
		if err := res.(io.Closer).Close(); err != nil {
			t.Errorf("%s: close: %v", name, err)
		}
	}
	recvAll("follow", c.Follow(), func(s any, ctx context.Context) (any, error) {
		return s.(*feedclient.FollowClientStream).Recv(ctx)
	}, []any{fullNote, fullNote})
	recvAll("followTiny", c.FollowTiny(), func(s any, ctx context.Context) (any, error) {
		return s.(*feedclient.FollowTinyClientStream).Recv(ctx)
	}, []any{tinyNote, tinyNote})
	recvAll("followList", c.FollowList(), func(s any, ctx context.Context) (any, error) {
		return s.(*feedclient.FollowListClientStream).Recv(ctx)
	}, []any{feed.NoteCollection{fullNote}, feed.NoteCollection{fullNote, fullNote}})
}
`
