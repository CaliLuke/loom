package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

// TestJSONRPCSSEResultTypesGeneratedModuleRoundTrip compiles the generated
// service, server, and client for every JSON-RPC SSE result shape and streams
// adversarial values through each one. Primitive and collection events are
// aliases of their result types, so services send plain Go values.
func TestJSONRPCSSEResultTypesGeneratedModuleRoundTrip(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCSSEResultTypesDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcsseresulttypes", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sse_result_types_test.go"), []byte(jsonRPCSSEResultTypesHarness), 0o600))
	serverDir := filepath.Join(dir, "gen", "jsonrpc", "jsonrpcsse_result_types", "server")
	require.NoError(t, os.WriteFile(filepath.Join(serverDir, "service_stream_test.go"), []byte(jsonRPCSSEServiceStreamHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

// jsonRPCSSEServiceStreamHarness exercises the service-level Stream from
// inside the generated server package. Its Event is any because the service
// has built-in results, so Send must reject a value of any other type before
// writing an event.
const jsonRPCSSEServiceStreamHarness = `package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

func newServiceStream() (*jSONRPCSSEResultTypesSSEStream, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rpc", nil)
	return &jSONRPCSSEResultTypesSSEStream{
		w:      rec,
		r:      req,
		writer: loomhttp.NewSSEStreamWriter(rec, req.Context(), loomtransport.TransportJSONRPC, loomhttp.StreamWritePolicy{}),
	}, rec
}

func TestServiceStreamRejectsUnknownEventType(t *testing.T) {
	s, rec := newServiceStream()
	err := s.Send(context.Background(), float64(1))
	if err == nil || err.Error() != "unknown event type: float64" {
		t.Errorf("got error %v, want unknown event type: float64", err)
	}
	if rec.Body.Len() != 0 || rec.Header().Get("Content-Type") != "" {
		t.Errorf("rejected event wrote %q with headers %v", rec.Body.String(), rec.Header())
	}
}

func TestServiceStreamSendsBuiltInEvent(t *testing.T) {
	s, rec := newServiceStream()
	if err := s.Send(context.Background(), "\r\n"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "\"params\":\"\\r\\n\"") {
		t.Errorf("unexpected event %q", rec.Body.String())
	}
}
`

// TestJSONRPCSSEExternalResultTypeModuleRoundTrip compiles every generated
// package of a JSON-RPC SSE method whose result type lives in a
// struct:pkg:path package: the service, the external type package, and the
// JSON-RPC server, SSE, and client transport. The service package cannot
// declare marker methods on that type, so its event type is an alias. The
// harness streams values through the generated server and client.
func TestJSONRPCSSEExternalResultTypeModuleRoundTrip(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		dsl.API("jsonrpc-sse-external", func() {
			dsl.JSONRPC(func() {})
		})
		item := dsl.Type("Item", func() {
			dsl.Attribute("text", dsl.String)
			dsl.Required("text")
			dsl.Meta("struct:pkg:path", "common")
		})
		dsl.Service("Items", func() {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
			dsl.Method("Watch", func() {
				dsl.Payload(func() {
					dsl.ID("id", dsl.String)
				})
				dsl.StreamingResult(item)
				dsl.JSONRPC(func() {
					dsl.ServerSentEvents()
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/jsonrpcsseexternal", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen", "items", "event_test.go"), []byte(jsonRPCSSEExternalEventHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "external_test.go"), []byte(jsonRPCSSEExternalRoundTripHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const jsonRPCSSEExternalRoundTripHarness = `package jsonrpcsseexternal_test

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

	common "example.com/jsonrpcsseexternal/gen/common"
	items "example.com/jsonrpcsseexternal/gen/items"
	client "example.com/jsonrpcsseexternal/gen/jsonrpc/items/client"
	server "example.com/jsonrpcsseexternal/gen/jsonrpc/items/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

var values = []*common.Item{{Text: "\r\n"}, {Text: ""}}

type service struct{}

func (service) Watch(ctx context.Context, _ *items.WatchPayload, st items.WatchServerStream) error {
	if err := st.Send(ctx, values[0]); err != nil {
		return err
	}
	return st.SendAndClose(ctx, values[1])
}

func TestExternalRoundTrip(t *testing.T) {
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	srv := server.New(items.NewEndpoints(service{}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler)
	server.Mount(mux, srv)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	c := client.NewClient("http", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id := "req-1"
	res, err := c.Watch()(ctx, &items.WatchPayload{ID: &id})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	s := res.(*client.WatchClientStream)
	defer func() {
		if err := s.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	}()
	for i, want := range values {
		got, err := s.Recv(ctx)
		if err != nil {
			t.Fatalf("recv %d: %v", i, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("recv %d: got %#v, want %#v", i, got, want)
		}
	}
	if _, err := s.Recv(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF, got %v", err)
	}
}
`

const jsonRPCSSEExternalEventHarness = `package items_test

import (
	common "example.com/jsonrpcsseexternal/gen/common"
	items "example.com/jsonrpcsseexternal/gen/items"
)

var (
	_ items.WatchEvent = &common.Item{}
	_ items.Event      = &common.Item{}
)
`

const jsonRPCSSEResultTypesHarness = `package jsonrpcsseresulttypes_test

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	svc "example.com/jsonrpcsseresulttypes/gen/jsonrpcsse_result_types"
	client "example.com/jsonrpcsseresulttypes/gen/jsonrpc/jsonrpcsse_result_types/client"
	server "example.com/jsonrpcsseresulttypes/gen/jsonrpc/jsonrpcsse_result_types/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

var (
	errPreStream = errors.New("rejected before stream")
	errMidStream = errors.New("failed mid stream")

	stringValues = []string{
		"",
		"\r\n",
		"trailing\n",
		"a\rb",
		"\n\n",
		"data: fake\n\nevent: x",
		"{\"q\":\"\\\"<>&\\\\ \"}",
		"tab\tunicode é  ",
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
		loom.JSONValue("1.5"),
	}
	arrayValues = [][]string{{"", "a\r\nb", "\n"}, {"x"}}
	mapValues   = []map[string]int{{"": 0, "a\nb": -1, "\r": 2}, {"k": 1}}
	noteValues  = []*svc.Note{{Text: ""}, {Text: "a\r\nb\n"}}
)

type service struct {
	failBefore bool
	failMid    bool
}

// stream sends every value but the last as a notification and the last as
// the final response, exercising both the Send and SendAndClose paths of the
// method event type.
func stream[T any](ctx context.Context, s *service, send, final func(context.Context, T) error, sendError func(context.Context, any, error) error, values []T) error {
	if s.failBefore {
		return errPreStream
	}
	for _, v := range values[:len(values)-1] {
		if err := send(ctx, v); err != nil {
			return err
		}
		if s.failMid {
			return sendError(ctx, nil, errMidStream)
		}
	}
	return final(ctx, values[len(values)-1])
}

func (s *service) StreamString(ctx context.Context, _ *svc.StreamStringPayload, st svc.StreamStringServerStream) error {
	return stream(ctx, s, st.Send, st.SendAndClose, st.SendError, stringValues)
}

func (s *service) StreamInt(ctx context.Context, _ *svc.StreamIntPayload, st svc.StreamIntServerStream) error {
	return stream(ctx, s, st.Send, st.SendAndClose, st.SendError, intValues)
}

func (s *service) StreamBool(ctx context.Context, _ *svc.StreamBoolPayload, st svc.StreamBoolServerStream) error {
	return stream(ctx, s, st.Send, st.SendAndClose, st.SendError, boolValues)
}

func (s *service) StreamBytes(ctx context.Context, _ *svc.StreamBytesPayload, st svc.StreamBytesServerStream) error {
	return stream(ctx, s, st.Send, st.SendAndClose, st.SendError, bytesValues)
}

func (s *service) StreamAny(ctx context.Context, _ *svc.StreamAnyPayload, st svc.StreamAnyServerStream) error {
	return stream(ctx, s, st.Send, st.SendAndClose, st.SendError, anyValues)
}

func (s *service) StreamArray(ctx context.Context, _ *svc.StreamArrayPayload, st svc.StreamArrayServerStream) error {
	return stream(ctx, s, st.Send, st.SendAndClose, st.SendError, arrayValues)
}

func (s *service) StreamMap(ctx context.Context, _ *svc.StreamMapPayload, st svc.StreamMapServerStream) error {
	return stream(ctx, s, st.Send, st.SendAndClose, st.SendError, mapValues)
}

func (s *service) StreamNote(ctx context.Context, _ *svc.StreamNotePayload, st svc.StreamNoteServerStream) error {
	send := func(ctx context.Context, v *svc.Note) error { return st.Send(ctx, v) }
	final := func(ctx context.Context, v *svc.Note) error { return st.SendAndClose(ctx, v) }
	return stream(ctx, s, send, final, st.SendError, noteValues)
}

func start(t *testing.T, impl *service) (*client.Client, string) {
	t.Helper()
	mux := loomhttp.NewMuxer()
	errhandler := func(context.Context, http.ResponseWriter, error) {}
	srv := server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, errhandler)
	server.Mount(mux, srv)
	hs := httptest.NewServer(mux)
	t.Cleanup(hs.Close)
	return newClient(hs), hs.URL
}

func newClient(hs *httptest.Server) *client.Client {
	host := strings.TrimPrefix(hs.URL, "http://")
	return client.NewClient("http", host, hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
}

type recvStream[T any] interface {
	Recv(context.Context) (T, error)
	Close() error
}

func open[T any](t *testing.T, name string, endpoint loom.Endpoint, payload any) recvStream[T] {
	t.Helper()
	res, err := endpoint(context.Background(), payload)
	if err != nil {
		t.Fatalf("%s: connect: %v", name, err)
	}
	s, ok := res.(recvStream[T])
	if !ok {
		t.Fatalf("%s: unexpected stream type %T", name, res)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("%s: close: %v", name, err)
		}
	})
	return s
}

// requireRoundTrip receives exactly len(want) events followed by io.EOF and
// requires each event to equal the value the service sent.
func requireRoundTrip[T any](t *testing.T, name string, endpoint loom.Endpoint, payload any, want []T, equal func(T, T) bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := open[T](t, name, endpoint, payload)
	for i, w := range want {
		got, err := s.Recv(ctx)
		if err != nil {
			t.Errorf("%s[%d]: recv: %v", name, i, err)
			return
		}
		if !equal(got, w) {
			t.Errorf("%s[%d]: got %#v, want %#v", name, i, got, w)
		}
	}
	if _, err := s.Recv(ctx); !errors.Is(err, io.EOF) {
		t.Errorf("%s: expected io.EOF after %d events, got %v", name, len(want), err)
	}
}

func deepEqual[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

func id() *string {
	v := "req-1"
	return &v
}

func TestJSONRPCSSEResultTypesRoundTrip(t *testing.T) {
	c, _ := start(t, &service{})
	requireRoundTrip(t, "string", c.StreamString(), &svc.StreamStringPayload{ID: id()}, stringValues, deepEqual[string])
	requireRoundTrip(t, "int", c.StreamInt(), &svc.StreamIntPayload{ID: id()}, intValues, deepEqual[int])
	requireRoundTrip(t, "bool", c.StreamBool(), &svc.StreamBoolPayload{ID: id()}, boolValues, deepEqual[bool])
	requireRoundTrip(t, "bytes", c.StreamBytes(), &svc.StreamBytesPayload{ID: id()}, bytesValues, bytes.Equal)
	requireRoundTrip(t, "any", c.StreamAny(), &svc.StreamAnyPayload{ID: id()}, anyValues, func(a, b loom.JSONValue) bool {
		return bytes.Equal(a, b)
	})
	requireRoundTrip(t, "array", c.StreamArray(), &svc.StreamArrayPayload{ID: id()}, arrayValues, deepEqual[[]string])
	requireRoundTrip(t, "map", c.StreamMap(), &svc.StreamMapPayload{ID: id()}, mapValues, deepEqual[map[string]int])
	requireRoundTrip(t, "note", c.StreamNote(), &svc.StreamNotePayload{ID: id()}, noteValues, deepEqual[*svc.Note])
}

// TestJSONRPCSSEResultTypesWireEncoding requires primitive events to be JSON
// values inside the JSON-RPC envelope so SSE framing cannot split or alter
// them: CR/LF are escaped and bytes are base64 encoded.
func TestJSONRPCSSEResultTypesWireEncoding(t *testing.T) {
	_, url := start(t, &service{})
	cases := []struct {
		method string
		want   []string
	}{
		{"StreamString", []string{"\"\"", "\"\\r\\n\"", "\"trailing\\n\""}},
		{"StreamBytes", []string{"\"\"", "\"AA3/Cg==\"", "\"Cg==\""}},
		{"StreamArray", []string{"[\"\",\"a\\r\\nb\",\"\\n\"]"}},
	}
	for _, tc := range cases {
		body := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":\"w\",\"method\":\"" + tc.method + "\"}")
		req, err := http.NewRequest(http.MethodPost, url+"/rpc", body)
		if err != nil {
			t.Fatalf("%s: build request: %v", tc.method, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: request: %v", tc.method, err)
		}
		events, err := loomhttp.ParseSSEStream(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatalf("%s: close: %v", tc.method, closeErr)
		}
		if err != nil {
			t.Fatalf("%s: parse: %v", tc.method, err)
		}
		for i, want := range tc.want {
			if i >= len(events) {
				t.Errorf("%s[%d]: missing event in %#v", tc.method, i, events)
				continue
			}
			var envelope map[string]jsontext.Value
			if err := json.Unmarshal([]byte(events[i].Data), &envelope); err != nil {
				t.Errorf("%s[%d]: decode envelope %q: %v", tc.method, i, events[i].Data, err)
				continue
			}
			if got := string(envelope["params"]); got != want {
				t.Errorf("%s[%d]: got params %s, want %s", tc.method, i, got, want)
			}
		}
	}
}

func TestJSONRPCSSEResultTypesPreStreamFailure(t *testing.T) {
	c, _ := start(t, &service{failBefore: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := open[string](t, "string", c.StreamString(), &svc.StreamStringPayload{ID: id()})
	if _, err := s.Recv(ctx); err == nil || !strings.Contains(err.Error(), "JSON-RPC error") {
		t.Errorf("expected JSON-RPC error before any event, got %v", err)
	}
}

func TestJSONRPCSSEResultTypesSendErrorAfterEvent(t *testing.T) {
	c, _ := start(t, &service{failMid: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s := open[[]byte](t, "bytes", c.StreamBytes(), &svc.StreamBytesPayload{ID: id()})
	got, err := s.Recv(ctx)
	if err != nil || len(got) != 0 {
		t.Fatalf("first event: got %#v, %v", got, err)
	}
	if _, err := s.Recv(ctx); err == nil || !strings.Contains(err.Error(), "JSON-RPC error") {
		t.Errorf("expected JSON-RPC error event, got %v", err)
	}
}

// TestJSONRPCSSEResultTypesClientRejectsMismatchedParams requires the client
// to reject a notification whose params do not decode into the result type.
func TestJSONRPCSSEResultTypesClientRejectsMismatchedParams(t *testing.T) {
	frames := map[string]string{
		"string": "123",
		"bytes":  "\"not base64!\"",
		"int":    "\"42\"",
		"array":  "{\"a\":1}",
		"map":    "{\"a\":\"x\"}",
	}
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		name := r.URL.Query().Get("case")
		_, err := io.WriteString(w, "data: {\"jsonrpc\":\"2.0\",\"method\":\"JSONRPCSSEResultTypes/stream.event\",\"params\":"+frames[name]+"}\n\n")
		if err != nil {
			t.Errorf("write frame: %v", err)
		}
	}))
	t.Cleanup(hs.Close)
	c := newClient(hs)
	recvErr := func(name string, endpoint loom.Endpoint, payload any, recv func(any) error) {
		c.Doer = caseDoer{name: name, doer: hs.Client()}
		res, err := endpoint(context.Background(), payload)
		if err != nil {
			t.Fatalf("%s: connect: %v", name, err)
		}
		if err := recv(res); err == nil {
			t.Errorf("%s: expected decode error", name)
		}
	}
	ctx := context.Background()
	recvErr("string", c.StreamString(), &svc.StreamStringPayload{ID: id()}, func(res any) error {
		_, err := res.(recvStream[string]).Recv(ctx)
		return err
	})
	recvErr("bytes", c.StreamBytes(), &svc.StreamBytesPayload{ID: id()}, func(res any) error {
		_, err := res.(recvStream[[]byte]).Recv(ctx)
		return err
	})
	recvErr("int", c.StreamInt(), &svc.StreamIntPayload{ID: id()}, func(res any) error {
		_, err := res.(recvStream[int]).Recv(ctx)
		return err
	})
	recvErr("array", c.StreamArray(), &svc.StreamArrayPayload{ID: id()}, func(res any) error {
		_, err := res.(recvStream[[]string]).Recv(ctx)
		return err
	})
	recvErr("map", c.StreamMap(), &svc.StreamMapPayload{ID: id()}, func(res any) error {
		_, err := res.(recvStream[map[string]int]).Recv(ctx)
		return err
	})
}

type caseDoer struct {
	name string
	doer loomhttp.Doer
}

func (d caseDoer) Do(req *http.Request) (*http.Response, error) {
	q := req.URL.Query()
	q.Set("case", d.name)
	req.URL.RawQuery = q.Encode()
	return d.doer.Do(req)
}
`
