package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestSSEResultTypesServerSkipsMissingBodyConstructor(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEResultTypesDSL)
	services := CreateHTTPServices(root)
	data := services.Get("SSEResultTypes")
	require.NotNil(t, data)

	cases := []struct {
		Method   string
		WithBody bool
	}{
		{"StreamString", false},
		{"StreamInt", false},
		{"StreamBool", false},
		{"StreamBytes", false},
		{"StreamAny", false},
		{"StreamArray", false},
		{"StreamMap", false},
		{"StreamNote", true},
	}
	for _, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			ed := data.Endpoint(c.Method)
			require.NotNil(t, ed)
			require.NotNil(t, ed.SSE)
			if !c.WithBody {
				require.Nil(t, ed.SSE.ResponseBody)
				return
			}
			require.NotNil(t, ed.SSE.ResponseBody)
			require.NotNil(t, ed.SSE.ResponseBody.Init)
			require.Equal(t, "NewStreamNoteResponseBody", ed.SSE.ResponseBody.Init.Name)
		})
	}
}

func TestSSEResultTypesGeneratedModuleCompiles(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEResultTypesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/sseresulttypes", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sse_result_types_test.go"), []byte(sseResultTypesHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}

const sseResultTypesHarness = `package sseresulttypes_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	svc "example.com/sseresulttypes/gen/sse_result_types"
	client "example.com/sseresulttypes/gen/http/sse_result_types/client"
	server "example.com/sseresulttypes/gen/http/sse_result_types/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

var (
	errPreStream = errors.New("rejected before stream")

	stringValues = []string{
		"",
		"\r\n",
		"trailing\n",
		"a\rb",
		"hello\nworld",
		"\n\n",
		"data: fake\n\nevent: x",
		"{\"q\":\"\\\"<>&\\\\ \"}",
		"tab\tunicode é  ",
	}
	intValues   = []int{0, -1, 42}
	boolValues  = []bool{true, false}
	bytesValues = [][]byte{
		{},
		{0x00, '\r', 0xff, '\n'},
		{'\n'},
		[]byte("raw bytes"),
	}
	anyValues = []loom.JSONValue{
		loom.JSONValue("{\"k\":[1,\"x\",null]}"),
		loom.JSONValue("\"\\r\\n\""),
		loom.JSONValue("null"),
		loom.JSONValue("1.50"),
	}
	arrayValues = [][]string{{}, {"", "a\r\nb", "\n"}}
	mapValues   = []map[string]int{{}, {"": 0, "a\nb": -1, "\r": 2}}
	noteValues  = []*svc.Note{{Text: ""}, {Text: "a\r\nb\n"}}
)

type service struct {
	fail bool
}

func sendAll[T any](ctx context.Context, send func(context.Context, T) error, values []T) error {
	for _, v := range values {
		if err := send(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) StreamString(ctx context.Context, stream svc.StreamStringServerStream) error {
	if s.fail {
		return errPreStream
	}
	return sendAll(ctx, stream.SendWithContext, stringValues)
}

func (s *service) StreamInt(ctx context.Context, stream svc.StreamIntServerStream) error {
	return sendAll(ctx, stream.SendWithContext, intValues)
}

func (s *service) StreamBool(ctx context.Context, stream svc.StreamBoolServerStream) error {
	return sendAll(ctx, stream.SendWithContext, boolValues)
}

func (s *service) StreamBytes(ctx context.Context, stream svc.StreamBytesServerStream) error {
	return sendAll(ctx, stream.SendWithContext, bytesValues)
}

func (s *service) StreamAny(ctx context.Context, stream svc.StreamAnyServerStream) error {
	return sendAll(ctx, stream.SendWithContext, anyValues)
}

func (s *service) StreamArray(ctx context.Context, stream svc.StreamArrayServerStream) error {
	return sendAll(ctx, stream.SendWithContext, arrayValues)
}

func (s *service) StreamMap(ctx context.Context, stream svc.StreamMapServerStream) error {
	return sendAll(ctx, stream.SendWithContext, mapValues)
}

func (s *service) StreamNote(ctx context.Context, stream svc.StreamNoteServerStream) error {
	return sendAll(ctx, stream.SendWithContext, noteValues)
}

func start(t *testing.T, impl *service) *client.Client {
	t.Helper()
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	srv := server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler, nil)
	server.Mount(mux, srv)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	host := strings.TrimPrefix(hs.URL, "http://")
	return client.NewClient("http", host, hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
}

type recvStream[T any] interface {
	Recv(context.Context) (T, error)
	Close() error
}

// requireRoundTrip receives exactly len(want) events followed by io.EOF and
// requires each event to equal the value the service sent.
func requireRoundTrip[T any](t *testing.T, name string, endpoint loom.Endpoint, want []T, equal func(T, T) bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := endpoint(ctx, nil)
	if err != nil {
		t.Fatalf("%s: connect: %v", name, err)
	}
	stream, ok := res.(recvStream[T])
	if !ok {
		t.Fatalf("%s: unexpected stream type %T", name, res)
	}
	for i, w := range want {
		got, err := stream.Recv(ctx)
		if err != nil {
			t.Errorf("%s[%d]: recv: %v", name, i, err)
			return
		}
		if !equal(got, w) {
			t.Errorf("%s[%d]: got %#v, want %#v", name, i, got, w)
		}
	}
	if _, err := stream.Recv(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("%s: expected io.EOF after %d events, got %v", name, len(want), err)
	}
}

func deepEqual[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

func TestSSEResultTypesRoundTrip(t *testing.T) {
	c := start(t, &service{})
	requireRoundTrip(t, "string", c.StreamString(), stringValues, deepEqual[string])
	requireRoundTrip(t, "int", c.StreamInt(), intValues, deepEqual[int])
	requireRoundTrip(t, "bool", c.StreamBool(), boolValues, deepEqual[bool])
	requireRoundTrip(t, "bytes", c.StreamBytes(), bytesValues, bytes.Equal)
	requireRoundTrip(t, "any", c.StreamAny(), anyValues, func(a, b loom.JSONValue) bool {
		return bytes.Equal(a, b)
	})
	requireRoundTrip(t, "array", c.StreamArray(), arrayValues, deepEqual[[]string])
	requireRoundTrip(t, "map", c.StreamMap(), mapValues, deepEqual[map[string]int])
	requireRoundTrip(t, "note", c.StreamNote(), noteValues, deepEqual[*svc.Note])
}

func TestSSEResultTypesWholeEventDataIsJSON(t *testing.T) {
	c := start(t, &service{})
	cases := []struct {
		path string
		want []string
	}{
		{"/string", []string{"\"\"", "\"\\r\\n\""}},
		{"/int", []string{"0", "-1"}},
		{"/bytes", []string{"\"\"", "\"AA3/Cg==\""}},
		{"/array", []string{"[]"}},
	}
	for _, tc := range cases {
		req, err := c.BuildStreamStringRequest(context.Background(), nil)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		req.URL.Path = tc.path
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: request: %v", tc.path, err)
		}
		events, err := loomhttp.ParseSSEStream(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatalf("%s: close: %v", tc.path, closeErr)
		}
		if err != nil {
			t.Fatalf("%s: parse: %v", tc.path, err)
		}
		for i, want := range tc.want {
			if i >= len(events) || events[i].Data != want {
				t.Errorf("%s[%d]: got events %#v, want data %q", tc.path, i, events, want)
			}
		}
	}
}

func TestSSEResultTypesPreStreamFailureIsNotAStream(t *testing.T) {
	c := start(t, &service{fail: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.StreamString()(ctx, nil)
	if err == nil {
		t.Fatalf("expected pre-stream failure, got stream %T", res)
	}
	if !strings.Contains(err.Error(), "unexpected status from SSE endpoint") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSSEResultTypesClientRejectsMalformedData(t *testing.T) {
	body := func(frame string) *http.Response {
		return &http.Response{Body: io.NopCloser(strings.NewReader(frame))}
	}
	ctx := context.Background()
	if _, err := client.NewStreamStringStream(body("data: unquoted\n\n"), loomhttp.ResponseDecoder).Recv(ctx); err == nil {
		t.Error("string: expected decode error for non-JSON data")
	}
	if _, err := client.NewStreamBytesStream(body("data: \"not base64!\"\n\n"), loomhttp.ResponseDecoder).Recv(ctx); err == nil {
		t.Error("bytes: expected decode error for invalid base64")
	}
	if _, err := client.NewStreamIntStream(body("data: forty-two\n\n"), loomhttp.ResponseDecoder).Recv(ctx); err == nil {
		t.Error("int: expected parse error")
	}
	if _, err := client.NewStreamBoolStream(body("data: yes\n\n"), loomhttp.ResponseDecoder).Recv(ctx); err == nil {
		t.Error("bool: expected decode error")
	}
	if _, err := client.NewStreamArrayStream(body("data: [\"a\",\n\n"), loomhttp.ResponseDecoder).Recv(ctx); err == nil {
		t.Error("array: expected decode error")
	}
	if _, err := client.NewStreamMapStream(body("data: {\"a\":\"x\"}\n\n"), loomhttp.ResponseDecoder).Recv(ctx); err == nil {
		t.Error("map: expected decode error")
	}
	if _, err := client.NewStreamAnyStream(body("data: {\"a\":\n\n"), loomhttp.ResponseDecoder).Recv(ctx); err == nil {
		t.Error("any: expected decode error")
	}
}
`
