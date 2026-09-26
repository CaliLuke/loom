package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestSSEOptionalDataCodegen checks that an SSE data field mapped from an
// optional attribute without a default value follows the pointer semantics
// of its service type field. The server writes the dereferenced text of an
// optional String, or empty data when it is nil, and the client leaves the
// field nil for empty data. Other optional primitives are JSON literals, null
// when nil, and the client decodes them with the decoder.
func TestSSEOptionalDataCodegen(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEOptionalDataDSL)
	services := CreateHTTPServices(root)
	data := services.Get("SSEOptionalData")
	require.NotNil(t, data)

	cases := []struct {
		Method        string
		Pointer       bool
		ServerPresent []string
		ClientPresent []string
		ClientAbsent  []string
	}{
		{
			Method:        "StreamText",
			Pointer:       true,
			ServerPresent: []string{"\tpayload = \"\"\n\tif body.Text != nil {\n\t\tpayload = *body.Text\n\t}\n"},
			ClientPresent: []string{"\tif dataContent != \"\" {\n\t\tevent.Text = &dataContent\n\t}\n"},
			ClientAbsent:  []string{"event.Text = dataContent", "Decode(&event.Text)"},
		},
		{
			Method:        "StreamCount",
			Pointer:       true,
			ServerPresent: []string{"\tpayload = body.Count\n"},
			ClientPresent: []string{"Decode(&event.Count)"},
			ClientAbsent:  []string{"strconv.Atoi", "event.Count = v"},
		},
		{
			Method:        "StreamFlag",
			Pointer:       true,
			ServerPresent: []string{"\tpayload = body.Flag\n"},
			ClientPresent: []string{"Decode(&event.Flag)"},
		},
	}
	for _, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			ed := data.Endpoint(c.Method)
			require.NotNil(t, ed)
			require.NotNil(t, ed.SSE)
			require.Equal(t, c.Pointer, ed.SSE.DataPointer)
			serverCode := renderSection(t, serverSSESection(ed))
			for _, want := range c.ServerPresent {
				require.Contains(t, serverCode, want)
			}
			clientCode := renderSection(t, sseClientSection(ed))
			for _, want := range c.ClientPresent {
				require.Contains(t, clientCode, want)
			}
			for _, unwanted := range c.ClientAbsent {
				require.NotContains(t, clientCode, unwanted)
			}
		})
	}
}

// TestSSEOptionalDataGeneratedModule compiles the generated server and client
// of SSEOptionalDataDSL and exercises them over HTTP: the round trip of set,
// nil, and edge values, the wire frames, a failure before the stream commits,
// and a failure after the first event.
func TestSSEOptionalDataGeneratedModule(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEOptionalDataDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/sseoptionaldata", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sse_optional_data_test.go"), []byte(sseOptionalDataHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}

const sseOptionalDataHarness = `package sseoptionaldata_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	svc "example.com/sseoptionaldata/gen/sse_optional_data"
	client "example.com/sseoptionaldata/gen/http/sse_optional_data/client"
	server "example.com/sseoptionaldata/gen/http/sse_optional_data/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

var (
	errPreStream  = errors.New("rejected before stream")
	errMidStream  = errors.New("failed after first event")
	textEvents    = []*svc.StreamTextResult{
		{ID: "t1", Text: ptr("hello")},
		{ID: "t2"},
		// Raw text cannot tell an empty string from a missing value, so an
		// empty text is received as nil, like an empty SSE id.
		{ID: "t3", Text: ptr("")},
		{ID: "t4", Text: ptr("null")},
		{ID: "t5", Text: ptr("{\"a\":1} ünïcode")},
	}
	countEvents = []*svc.StreamCountResult{
		{ID: "c1", Count: ptr(5)},
		{ID: "c2"},
		{ID: "c3", Count: ptr(0)},
		{ID: "c4", Count: ptr(-7)},
	}
	flagEvents = []*svc.StreamFlagResult{
		{ID: "f1", Flag: ptr(true)},
		{ID: "f2"},
		{ID: "f3", Flag: ptr(false)},
	}
)

func ptr[T any](v T) *T {
	return &v
}

type service struct {
	failBefore bool
	failAfter  bool
}

func sendAll[T any](ctx context.Context, send func(context.Context, T) error, values []T) error {
	for _, v := range values {
		if err := send(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) StreamText(ctx context.Context, stream svc.StreamTextServerStream) error {
	if s.failBefore {
		return errPreStream
	}
	if s.failAfter {
		if err := stream.SendWithContext(ctx, textEvents[0]); err != nil {
			return err
		}
		return errMidStream
	}
	return sendAll(ctx, stream.SendWithContext, textEvents)
}

func (s *service) StreamCount(ctx context.Context, stream svc.StreamCountServerStream) error {
	return sendAll(ctx, stream.SendWithContext, countEvents)
}

func (s *service) StreamFlag(ctx context.Context, stream svc.StreamFlagServerStream) error {
	return sendAll(ctx, stream.SendWithContext, flagEvents)
}

func start(t *testing.T, impl *service) (*client.Client, string) {
	t.Helper()
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	srv := server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler, nil)
	server.Mount(mux, srv)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	host := strings.TrimPrefix(hs.URL, "http://")
	return client.NewClient("http", host, hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false), hs.URL
}

type recvStream[T any] interface {
	Recv(context.Context) (T, error)
	Close() error
}

func open[T any](t *testing.T, name string, endpoint loom.Endpoint) (recvStream[T], context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	res, err := endpoint(ctx, nil)
	if err != nil {
		t.Fatalf("%s: connect: %v", name, err)
	}
	stream, ok := res.(recvStream[T])
	if !ok {
		t.Fatalf("%s: unexpected stream type %T", name, res)
	}
	return stream, ctx
}

func receiveAll[T any](t *testing.T, name string, endpoint loom.Endpoint, n int) []T {
	t.Helper()
	stream, ctx := open[T](t, name, endpoint)
	got := make([]T, 0, n)
	for i := range n {
		event, err := stream.Recv(ctx)
		if err != nil {
			t.Fatalf("%s[%d]: recv: %v", name, i, err)
		}
		got = append(got, event)
	}
	if _, err := stream.Recv(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("%s: expected io.EOF after %d events, got %v", name, n, err)
	}
	return got
}

func TestSSEOptionalDataRoundTrip(t *testing.T) {
	c, _ := start(t, &service{})

	gotText := receiveAll[*svc.StreamTextResult](t, "text", c.StreamText(), len(textEvents))
	wantText := []*svc.StreamTextResult{
		{ID: "t1", Text: ptr("hello")},
		{ID: "t2"},
		{ID: "t3"},
		{ID: "t4", Text: ptr("null")},
		{ID: "t5", Text: ptr("{\"a\":1} ünïcode")},
	}
	if !reflect.DeepEqual(gotText, wantText) {
		t.Errorf("text: got %s, want %s", dumpText(gotText), dumpText(wantText))
	}

	gotCount := receiveAll[*svc.StreamCountResult](t, "count", c.StreamCount(), len(countEvents))
	if !reflect.DeepEqual(gotCount, countEvents) {
		t.Errorf("count: got %+v, want %+v", gotCount, countEvents)
	}

	gotFlag := receiveAll[*svc.StreamFlagResult](t, "flag", c.StreamFlag(), len(flagEvents))
	if !reflect.DeepEqual(gotFlag, flagEvents) {
		t.Errorf("flag: got %+v, want %+v", gotFlag, flagEvents)
	}
}

func TestSSEOptionalDataWireFrames(t *testing.T) {
	_, base := start(t, &service{})
	cases := []struct {
		path string
		want string
	}{
		{"/text", "id: t1\ndata: hello\n\nid: t2\n\nid: t3\n\nid: t4\ndata: null\n\nid: t5\ndata: {\"a\":1} ünïcode\n\n"},
		{"/count", "id: c1\ndata: 5\n\nid: c2\ndata: null\n\nid: c3\ndata: 0\n\nid: c4\ndata: -7\n\n"},
		{"/flag", "id: f1\ndata: true\n\nid: f2\ndata: null\n\nid: f3\ndata: false\n\n"},
	}
	for _, tc := range cases {
		resp, err := http.Get(base + tc.path)
		if err != nil {
			t.Fatalf("%s: request: %v", tc.path, err)
		}
		body, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatalf("%s: close: %v", tc.path, closeErr)
		}
		if err != nil {
			t.Fatalf("%s: read: %v", tc.path, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status %d, want %d", tc.path, resp.StatusCode, http.StatusOK)
		}
		if got := string(body); got != tc.want {
			t.Errorf("%s: got frames %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestSSEOptionalDataPreStreamFailureIsNotAStream(t *testing.T) {
	c, base := start(t, &service{failBefore: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.StreamText()(ctx, nil)
	if err == nil {
		t.Fatalf("expected pre-stream failure, got stream %T", res)
	}
	if !strings.Contains(err.Error(), "unexpected status from SSE endpoint") {
		t.Errorf("unexpected error: %v", err)
	}
	resp, err := http.Get(base + "/text")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
	if resp.StatusCode == http.StatusOK || strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Errorf("pre-stream failure committed the stream: status %d, content type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
}

func TestSSEOptionalDataFailureAfterFirstEventEndsStream(t *testing.T) {
	c, _ := start(t, &service{failAfter: true})
	stream, ctx := open[*svc.StreamTextResult](t, "text", c.StreamText())
	event, err := stream.Recv(ctx)
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if event.ID != "t1" || event.Text == nil || *event.Text != "hello" {
		t.Errorf("first event = %s, want t1 with text hello", dumpText([]*svc.StreamTextResult{event}))
	}
	if _, err := stream.Recv(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF after the failure, got %v", err)
	}
}

func dumpText(events []*svc.StreamTextResult) string {
	parts := make([]string, 0, len(events))
	for _, e := range events {
		text := "nil"
		if e.Text != nil {
			text = "&" + *e.Text
		}
		parts = append(parts, "{"+e.ID+" "+text+"}")
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
`
