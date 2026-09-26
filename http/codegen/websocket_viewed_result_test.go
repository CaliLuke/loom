package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestWebSocketClientViewedResultKeyedLiteral checks that the WebSocket
// client streams build the viewed result of a streamed or final result type
// or collection with keyed fields, so that go vet accepts them.
func TestWebSocketClientViewedResultKeyedLiteral(t *testing.T) {
	cases := []struct {
		service string
		want    string
	}{
		{service: "feed", want: "vres := feedviews.ItemCollection{Projected: res, View: s.view}"},
		{service: "tinyfeed", want: `vres := tinyfeedviews.ItemCollection{Projected: res, View: "tiny"}`},
		{service: "item", want: "vres := &itemviews.Item{Projected: res, View: s.view}"},
		{service: "upload", want: "vres := &uploadviews.Item{Projected: res, View: s.view}"},
		{service: "uploads", want: "vres := uploadsviews.ItemCollection{Projected: res, View: s.view}"},
		{service: "chat", want: "vres := chatviews.ItemCollection{Projected: res, View: s.view}"},
	}
	root := RunHTTPDSL(t, webSocketViewedResultDSL)
	services := CreateHTTPServices(root)
	for _, c := range cases {
		t.Run(c.service, func(t *testing.T) {
			data := services.Get(c.service)
			require.NotNil(t, data)
			require.Len(t, data.Endpoints, 1)
			ws := data.Endpoints[0].ClientWebSocket
			require.NotNil(t, ws)
			code := codegen.SectionCode(t, websocketRecvSection(ws))
			require.Contains(t, code, c.want)
			require.NotContains(t, code, "{res, ")
		})
	}
}

// TestWebSocketViewedResultGeneratedIntegration generates WebSocket services
// that stream viewed result types and collections, and that return them
// after a streaming payload. It compiles and vets them, then checks that the
// generated client receives the results rendered with the view that the
// server and the client set.
func TestWebSocketViewedResultGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, webSocketViewedResultDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/viewedresultws", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "viewed_result_ws_test.go"), []byte(webSocketViewedResultHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

// webSocketViewedResultDSL declares WebSocket services whose streamed or
// final results are result types with two views: streamed collections with
// and without a fixed view, a streamed result type, a result type and a
// collection returned after a streaming payload, and a bidirectional stream
// of collections.
func webSocketViewedResultDSL() {
	item := ResultType("application/vnd.item", func() {
		TypeName("Item")
		Attributes(func() {
			Attribute("id", String)
			Attribute("name", String)
			Required("id")
		})
		View("default", func() {
			Attribute("id")
			Attribute("name")
		})
		View("tiny", func() {
			Attribute("id")
		})
	})
	frame := Type("Frame", func() {
		Attribute("id", String)
		Required("id")
	})
	stream := func(service string, method func()) {
		Service(service, func() {
			Method("talk", func() {
				method()
				HTTP(func() {
					GET("/" + service)
				})
			})
		})
	}
	stream("feed", func() {
		StreamingResult(CollectionOf(item))
	})
	stream("tinyfeed", func() {
		StreamingResult(CollectionOf(item), func() {
			View("tiny")
		})
	})
	stream("item", func() {
		StreamingResult(item)
	})
	stream("upload", func() {
		StreamingPayload(frame)
		Result(item)
	})
	stream("uploads", func() {
		StreamingPayload(frame)
		Result(CollectionOf(item))
	})
	stream("chat", func() {
		StreamingPayload(frame)
		StreamingResult(CollectionOf(item))
	})
}

const webSocketViewedResultHarness = `package viewedresultws

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	chat "example.com/viewedresultws/gen/chat"
	feed "example.com/viewedresultws/gen/feed"
	chatclient "example.com/viewedresultws/gen/http/chat/client"
	chatserver "example.com/viewedresultws/gen/http/chat/server"
	feedclient "example.com/viewedresultws/gen/http/feed/client"
	feedserver "example.com/viewedresultws/gen/http/feed/server"
	itemclient "example.com/viewedresultws/gen/http/item/client"
	itemserver "example.com/viewedresultws/gen/http/item/server"
	tinyfeedclient "example.com/viewedresultws/gen/http/tinyfeed/client"
	tinyfeedserver "example.com/viewedresultws/gen/http/tinyfeed/server"
	uploadclient "example.com/viewedresultws/gen/http/upload/client"
	uploadserver "example.com/viewedresultws/gen/http/upload/server"
	uploadsclient "example.com/viewedresultws/gen/http/uploads/client"
	uploadsserver "example.com/viewedresultws/gen/http/uploads/server"
	item "example.com/viewedresultws/gen/item"
	tinyfeed "example.com/viewedresultws/gen/tinyfeed"
	upload "example.com/viewedresultws/gen/upload"
	uploads "example.com/viewedresultws/gen/uploads"
	loomhttp "github.com/CaliLuke/loom/http"
)

func name(s string) *string {
	return &s
}

type feedService struct{ view string }

func (s *feedService) Talk(ctx context.Context, stream feed.TalkServerStream) error {
	stream.SetView(s.view)
	if err := stream.SendWithContext(ctx, feed.ItemCollection{{ID: "a", Name: name("A")}, {ID: "b"}}); err != nil {
		return err
	}
	return stream.Close()
}

type tinyfeedService struct{}

func (tinyfeedService) Talk(ctx context.Context, stream tinyfeed.TalkServerStream) error {
	if err := stream.SendWithContext(ctx, tinyfeed.ItemCollection{{ID: "a", Name: name("A")}}); err != nil {
		return err
	}
	return stream.Close()
}

type itemService struct{ view string }

func (s *itemService) Talk(ctx context.Context, stream item.TalkServerStream) error {
	stream.SetView(s.view)
	if err := stream.SendWithContext(ctx, &item.Item{ID: "a", Name: name("A")}); err != nil {
		return err
	}
	return stream.Close()
}

type uploadService struct{ view string }

func (s *uploadService) Talk(ctx context.Context, stream upload.TalkServerStream) error {
	stream.SetView(s.view)
	var ids []string
	for {
		frame, err := stream.RecvWithContext(ctx)
		if errors.Is(err, io.EOF) {
			return stream.SendAndCloseWithContext(ctx, &upload.Item{ID: strings.Join(ids, ","), Name: name("A")})
		}
		if err != nil {
			return err
		}
		ids = append(ids, frame.ID)
	}
}

type uploadsService struct{ view string }

func (s *uploadsService) Talk(ctx context.Context, stream uploads.TalkServerStream) error {
	stream.SetView(s.view)
	var res uploads.ItemCollection
	for {
		frame, err := stream.RecvWithContext(ctx)
		if errors.Is(err, io.EOF) {
			return stream.SendAndCloseWithContext(ctx, res)
		}
		if err != nil {
			return err
		}
		res = append(res, &uploads.Item{ID: frame.ID, Name: name(strings.ToUpper(frame.ID))})
	}
}

type chatService struct{ view string }

func (s *chatService) Talk(ctx context.Context, stream chat.TalkServerStream) error {
	stream.SetView(s.view)
	for {
		frame, err := stream.RecvWithContext(ctx)
		if errors.Is(err, io.EOF) {
			return stream.Close()
		}
		if err != nil {
			return err
		}
		if err := stream.SendWithContext(ctx, chat.ItemCollection{{ID: frame.ID, Name: name("A")}}); err != nil {
			return err
		}
	}
}

func serve(t *testing.T, view string) string {
	t.Helper()
	mux := loomhttp.NewMuxer()
	dec, enc := loomhttp.RequestDecoder, loomhttp.ResponseEncoder
	up := &websocket.Upgrader{}
	feedserver.Mount(mux, feedserver.New(feed.NewEndpoints(&feedService{view: view}), mux, dec, enc, nil, nil, up, nil))
	tinyfeedserver.Mount(mux, tinyfeedserver.New(tinyfeed.NewEndpoints(tinyfeedService{}), mux, dec, enc, nil, nil, up, nil))
	itemserver.Mount(mux, itemserver.New(item.NewEndpoints(&itemService{view: view}), mux, dec, enc, nil, nil, up, nil))
	uploadserver.Mount(mux, uploadserver.New(upload.NewEndpoints(&uploadService{view: view}), mux, dec, enc, nil, nil, up, nil))
	uploadsserver.Mount(mux, uploadsserver.New(uploads.NewEndpoints(&uploadsService{view: view}), mux, dec, enc, nil, nil, up, nil))
	chatserver.Mount(mux, chatserver.New(chat.NewEndpoints(&chatService{view: view}), mux, dec, enc, nil, nil, up, nil))
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return strings.TrimPrefix(hs.URL, "http://")
}

func open[T any](t *testing.T, endpoint func(context.Context, any) (any, error)) T {
	t.Helper()
	raw, err := endpoint(context.Background(), nil)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	stream, ok := raw.(T)
	if !ok {
		t.Fatalf("stream type %T", raw)
	}
	return stream
}

// names returns the names of items, "-" for an absent name.
func names[T any](items []T, get func(T) *string) string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = "-"
		if n := get(it); n != nil {
			out[i] = *n
		}
	}
	return strings.Join(out, ",")
}

func TestViewedResults(t *testing.T) {
	cases := []struct {
		view  string
		names string
	}{
		{view: "default", names: "A"},
		{view: "tiny", names: "-"},
	}
	for _, c := range cases {
		t.Run(c.view, func(t *testing.T) {
			host := serve(t, c.view)
			enc, dec := loomhttp.RequestEncoder, loomhttp.ResponseDecoder

			fs := open[feed.TalkClientStream](t, feedclient.NewClient("ws", host, nil, enc, dec, false, websocket.DefaultDialer, nil).Talk())
			fs.(*feedclient.TalkClientStream).SetView(c.view)
			got, err := fs.Recv()
			if err != nil {
				t.Fatalf("feed recv: %v", err)
			}
			wantFeed := c.names + ",-"
			if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" || names(got, func(i *feed.Item) *string { return i.Name }) != wantFeed {
				t.Errorf("feed: got %+v, want ids a,b and names %s", got, wantFeed)
			}
			if _, err := fs.Recv(); !errors.Is(err, io.EOF) {
				t.Errorf("feed: got %v, want EOF", err)
			}

			ts := open[tinyfeed.TalkClientStream](t, tinyfeedclient.NewClient("ws", host, nil, enc, dec, false, websocket.DefaultDialer, nil).Talk())
			tiny, err := ts.Recv()
			if err != nil {
				t.Fatalf("tinyfeed recv: %v", err)
			}
			if len(tiny) != 1 || tiny[0].ID != "a" || tiny[0].Name != nil {
				t.Errorf("tinyfeed: got %+v, want id a without name", tiny)
			}
			if _, err := ts.Recv(); !errors.Is(err, io.EOF) {
				t.Errorf("tinyfeed: got %v, want EOF", err)
			}

			is := open[item.TalkClientStream](t, itemclient.NewClient("ws", host, nil, enc, dec, false, websocket.DefaultDialer, nil).Talk())
			is.(*itemclient.TalkClientStream).SetView(c.view)
			one, err := is.Recv()
			if err != nil {
				t.Fatalf("item recv: %v", err)
			}
			if one.ID != "a" || names([]*item.Item{one}, func(i *item.Item) *string { return i.Name }) != c.names {
				t.Errorf("item: got %+v, want id a and name %s", one, c.names)
			}
			if _, err := is.Recv(); !errors.Is(err, io.EOF) {
				t.Errorf("item: got %v, want EOF", err)
			}

			us := open[upload.TalkClientStream](t, uploadclient.NewClient("ws", host, nil, enc, dec, false, websocket.DefaultDialer, nil).Talk())
			us.(*uploadclient.TalkClientStream).SetView(c.view)
			for _, id := range []string{"x", "y"} {
				if err := us.Send(&upload.Frame{ID: id}); err != nil {
					t.Fatalf("upload send: %v", err)
				}
			}
			up, err := us.CloseAndRecv()
			if err != nil {
				t.Fatalf("upload close and recv: %v", err)
			}
			if up.ID != "x,y" || names([]*upload.Item{up}, func(i *upload.Item) *string { return i.Name }) != c.names {
				t.Errorf("upload: got %+v, want id x,y and name %s", up, c.names)
			}

			uss := open[uploads.TalkClientStream](t, uploadsclient.NewClient("ws", host, nil, enc, dec, false, websocket.DefaultDialer, nil).Talk())
			uss.(*uploadsclient.TalkClientStream).SetView(c.view)
			for _, id := range []string{"x", "y"} {
				if err := uss.Send(&uploads.Frame{ID: id}); err != nil {
					t.Fatalf("uploads send: %v", err)
				}
			}
			ups, err := uss.CloseAndRecv()
			if err != nil {
				t.Fatalf("uploads close and recv: %v", err)
			}
			wantUploads := "X,Y"
			if c.view == "tiny" {
				wantUploads = "-,-"
			}
			if len(ups) != 2 || ups[0].ID != "x" || ups[1].ID != "y" || names(ups, func(i *uploads.Item) *string { return i.Name }) != wantUploads {
				t.Errorf("uploads: got %+v, want ids x,y and names %s", ups, wantUploads)
			}

			cs := open[chat.TalkClientStream](t, chatclient.NewClient("ws", host, nil, enc, dec, false, websocket.DefaultDialer, nil).Talk())
			cs.(*chatclient.TalkClientStream).SetView(c.view)
			if err := cs.Send(&chat.Frame{ID: "z"}); err != nil {
				t.Fatalf("chat send: %v", err)
			}
			msg, err := cs.Recv()
			if err != nil {
				t.Fatalf("chat recv: %v", err)
			}
			if len(msg) != 1 || msg[0].ID != "z" || names(msg, func(i *chat.Item) *string { return i.Name }) != c.names {
				t.Errorf("chat: got %+v, want id z and name %s", msg, c.names)
			}
			if err := cs.Close(); err != nil {
				t.Fatalf("chat close: %v", err)
			}
		})
	}
}

func TestInvalidViewedResultFails(t *testing.T) {
	host := serve(t, "default")
	enc, dec := loomhttp.RequestEncoder, loomhttp.ResponseDecoder
	fs := open[feed.TalkClientStream](t, feedclient.NewClient("ws", host, nil, enc, dec, false, websocket.DefaultDialer, nil).Talk())
	fs.(*feedclient.TalkClientStream).SetView("unknown")
	if _, err := fs.Recv(); err == nil {
		t.Error("feed recv with an unknown client view: got no error")
	}
}
`
