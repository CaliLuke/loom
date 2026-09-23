package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const sseFieldPresenceHarness = `package ssefieldpresence_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	svc "example.com/ssefieldpresence/gen/sse_field_presence"
	client "example.com/ssefieldpresence/gen/http/sse_field_presence/client"
	server "example.com/ssefieldpresence/gen/http/sse_field_presence/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

var errPreStream = errors.New("rejected before stream")

func ptr[T any](v T) *T {
	return &v
}

// Each list holds the events the service sends, with the mapped fields set
// first and unset afterwards.
var (
	requiredEvents = []*svc.StreamRequiredResult{
		{ID: "r1", Event: "tick", Retry: 250, Text: "a"},
		{Text: "b"},
	}
	optionalEvents = []*svc.StreamOptionalResult{
		{ID: ptr("o1"), Event: ptr("tock"), Retry: ptr(int64(500)), Text: "a"},
		{Text: "b"},
		// The server writes id: and event: only for non-empty values, so an
		// empty id is omitted and cannot reset the client's last event ID. An
		// empty event would mean the default type, so it is omitted as well.
		{ID: ptr(""), Event: ptr(""), Retry: ptr(int64(0)), Text: "c"},
	}
	defaultedEvents = []*svc.StreamDefaultedResult{
		{ID: "d1", Event: "custom", Retry: 750, Text: "a"},
		{Text: "b"},
	}
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

func (s *service) StreamRequired(ctx context.Context, stream svc.StreamRequiredServerStream) error {
	return sendAll(ctx, stream.SendWithContext, requiredEvents)
}

func (s *service) StreamOptional(ctx context.Context, stream svc.StreamOptionalServerStream) error {
	if s.fail {
		return errPreStream
	}
	return sendAll(ctx, stream.SendWithContext, optionalEvents)
}

func (s *service) StreamDefaulted(ctx context.Context, stream svc.StreamDefaultedServerStream) error {
	return sendAll(ctx, stream.SendWithContext, defaultedEvents)
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

func receiveAll[T any](t *testing.T, name string, endpoint loom.Endpoint, n int) []T {
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

// The generated client does not decode the SSE retry field: retry is a
// reconnection hint that the wire-level test below verifies instead.
func TestSSEFieldPresenceRoundTrip(t *testing.T) {
	c, _ := start(t, &service{})

	gotRequired := receiveAll[*svc.StreamRequiredResult](t, "required", c.StreamRequired(), len(requiredEvents))
	wantRequired := []*svc.StreamRequiredResult{
		{ID: "r1", Event: "tick", Text: "a"},
		{Text: "b"},
	}
	if !reflect.DeepEqual(gotRequired, wantRequired) {
		t.Errorf("required: got %s, want %s", dump(gotRequired), dump(wantRequired))
	}

	gotOptional := receiveAll[*svc.StreamOptionalResult](t, "optional", c.StreamOptional(), len(optionalEvents))
	wantOptional := []*svc.StreamOptionalResult{
		{ID: ptr("o1"), Event: ptr("tock"), Text: "a"},
		{Text: "b"},
		{Text: "c"},
	}
	if !reflect.DeepEqual(gotOptional, wantOptional) {
		t.Errorf("optional: got %s, want %s", dump(gotOptional), dump(wantOptional))
	}

	gotDefaulted := receiveAll[*svc.StreamDefaultedResult](t, "defaulted", c.StreamDefaulted(), len(defaultedEvents))
	wantDefaulted := []*svc.StreamDefaultedResult{
		{ID: "d1", Event: "custom", Text: "a"},
		{ID: "default-id", Event: "default-event", Text: "b"},
	}
	if !reflect.DeepEqual(gotDefaulted, wantDefaulted) {
		t.Errorf("defaulted: got %s, want %s", dump(gotDefaulted), dump(wantDefaulted))
	}
}

type frame struct {
	id, event, retry, data string
	hasID, hasEvent, hasRetry bool
}

func readFrames(t *testing.T, url string) []frame {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("%s: request: %v", url, err)
	}
	body, err := io.ReadAll(resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil {
		t.Fatalf("%s: close: %v", url, closeErr)
	}
	if err != nil {
		t.Fatalf("%s: read: %v", url, err)
	}
	var frames []frame
	for _, raw := range strings.Split(strings.TrimSpace(string(body)), "\n\n") {
		var f frame
		for _, line := range strings.Split(raw, "\n") {
			name, value, _ := strings.Cut(line, ":")
			value = strings.TrimPrefix(value, " ")
			switch name {
			case "id":
				f.id, f.hasID = value, true
			case "event":
				f.event, f.hasEvent = value, true
			case "retry":
				f.retry, f.hasRetry = value, true
			case "data":
				f.data = value
			}
		}
		frames = append(frames, f)
	}
	return frames
}

func TestSSEFieldPresenceWireFrames(t *testing.T) {
	_, base := start(t, &service{})
	cases := []struct {
		path string
		want []frame
	}{
		{"/required", []frame{
			{id: "r1", event: "tick", retry: "250", data: "a", hasID: true, hasEvent: true, hasRetry: true},
			{data: "b"},
		}},
		{"/optional", []frame{
			{id: "o1", event: "tock", retry: "500", data: "a", hasID: true, hasEvent: true, hasRetry: true},
			{data: "b"},
			{data: "c"},
		}},
		{"/defaulted", []frame{
			{id: "d1", event: "custom", retry: "750", data: "a", hasID: true, hasEvent: true, hasRetry: true},
			{data: "b"},
		}},
	}
	for _, tc := range cases {
		got := readFrames(t, base+tc.path)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got frames %+v, want %+v", tc.path, got, tc.want)
		}
	}
}

func TestSSEFieldPresencePreStreamFailureIsNotAStream(t *testing.T) {
	c, _ := start(t, &service{fail: true})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := c.StreamOptional()(ctx, nil)
	if err == nil {
		t.Fatalf("expected pre-stream failure, got stream %T", res)
	}
	if !strings.Contains(err.Error(), "unexpected status from SSE endpoint") {
		t.Errorf("unexpected error: %v", err)
	}
}

// dump renders events with pointer fields dereferenced so failures show values.
func dump[T any](values []T) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		rv := reflect.Indirect(reflect.ValueOf(v))
		fields := make([]string, 0, rv.NumField())
		for i := range rv.NumField() {
			f := rv.Field(i)
			value := "nil"
			if f.Kind() != reflect.Pointer {
				value = fmt.Sprintf("%q", fmt.Sprint(f.Interface()))
			} else if !f.IsNil() {
				value = fmt.Sprintf("&%q", fmt.Sprint(f.Elem().Interface()))
			}
			fields = append(fields, rv.Type().Field(i).Name+":"+value)
		}
		parts = append(parts, "{"+strings.Join(fields, " ")+"}")
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
`

func TestSSEFieldPresenceCodegen(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEFieldPresenceDSL)
	services := CreateHTTPServices(root)
	data := services.Get("SSEFieldPresence")
	require.NotNil(t, data)

	cases := []struct {
		Method        string
		ServerPresent []string
		ServerAbsent  []string
		ClientPresent []string
		ClientAbsent  []string
	}{
		{
			Method: "StreamRequired",
			ServerPresent: []string{
				"if id := res.ID; id != \"\" {\n\t\tmsg.ID = id\n\t}",
				"if event := res.Event; event != \"\" {\n\t\tmsg.Type = event\n\t}",
				"if retry := res.Retry; retry > 0 {\n\t\tmsg.RetryMillis = int64(retry)\n\t}",
			},
			ClientPresent: []string{
				"event.ID = parsed.ID\n",
				"event.Event = parsed.Type\n",
			},
			ClientAbsent: []string{"&id", "&eventType", "\"default-"},
		},
		{
			Method: "StreamOptional",
			ServerPresent: []string{
				"if id := res.ID; id != nil && *id != \"\" {\n\t\tmsg.ID = *id\n\t}",
				"if event := res.Event; event != nil && *event != \"\" {\n\t\tmsg.Type = *event\n\t}",
				"if retry := res.Retry; retry != nil && *retry > 0 {\n\t\tmsg.RetryMillis = int64(*retry)\n\t}",
			},
			ServerAbsent: []string{"msg.ID = id\n", "msg.Type = event\n", "int64(retry)"},
			ClientPresent: []string{
				"if id := parsed.ID; id != \"\" {\n\t\tevent.ID = &id\n\t}",
				"if eventType := parsed.Type; eventType != \"\" {\n\t\tevent.Event = &eventType\n\t}",
			},
			ClientAbsent: []string{"event.ID = parsed.ID", "event.Event = parsed.Type", "\"default-"},
		},
		{
			Method: "StreamDefaulted",
			ServerPresent: []string{
				"if id := res.ID; id != \"\" {\n\t\tmsg.ID = id\n\t}",
				"if event := res.Event; event != \"\" {\n\t\tmsg.Type = event\n\t}",
				"if retry := res.Retry; retry > 0 {\n\t\tmsg.RetryMillis = int64(retry)\n\t}",
			},
			ClientPresent: []string{
				"event.ID = parsed.ID\n\tif event.ID == \"\" {\n\t\tevent.ID = \"default-id\"\n\t}",
				"event.Event = parsed.Type\n\tif event.Event == \"\" {\n\t\tevent.Event = \"default-event\"\n\t}",
			},
			ClientAbsent: []string{"&id", "&eventType"},
		},
	}
	for _, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			ed := data.Endpoint(c.Method)
			require.NotNil(t, ed)
			require.NotNil(t, ed.SSE)
			serverCode := renderSection(t, serverSSESection(ed))
			for _, want := range c.ServerPresent {
				require.Contains(t, serverCode, want)
			}
			for _, unwanted := range c.ServerAbsent {
				require.NotContains(t, serverCode, unwanted)
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

func TestSSEFieldPresenceGeneratedModuleCompiles(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEFieldPresenceDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/ssefieldpresence", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sse_field_presence_test.go"), []byte(sseFieldPresenceHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}
