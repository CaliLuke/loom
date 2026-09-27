package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestExtendedMethodTypesGenerated(t *testing.T) {
	for _, tc := range []struct {
		name string
		dsl  func()
	}{
		{"direct", testdata.ExtendedMethodTypesDSL},
		{"aliased", testdata.ExtendedAliasedMethodTypesDSL},
		{"result-type", testdata.ExtendedResultMethodTypesDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.dsl)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/extended", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "ownership_test.go"), []byte(extendedMethodTypesHarness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "./...")
		})
	}
}

const extendedMethodTypesHarness = `package extended_test

import (
 "context"
 "errors"
 "io"
 "net/http"
 "net/http/httptest"
 "reflect"
 "strings"
 "sync/atomic"
 "testing"
 "time"

 "github.com/gorilla/websocket"
 "github.com/stretchr/testify/require"
 "example.com/extended/gen/fields"
 client "example.com/extended/gen/http/fields/client"
 server "example.com/extended/gen/http/fields/server"
 loomhttp "github.com/CaliLuke/loom/http"
)

type service struct {
 calls atomic.Int32
}

// Plain preserves the original type.
func (s *service) Plain(_ context.Context, p *fields.Value) (*fields.Value, error) {
 return p, nil
}

// Send reads the required field added to this payload only.
func (s *service) Send(_ context.Context, p *fields.ValueSendPayload) (*fields.Value, error) {
 s.calls.Add(1)
 return &fields.Value{Right: &p.Left}, nil
}

// Show returns the required field added to this result only.
func (s *service) Show(_ context.Context, p *fields.Value) (*fields.ValueShowResult, error) {
 return &fields.ValueShowResult{Left: "response", Right: p.Right}, nil
}

// Upload reads the extended stream item and returns the original type.
func (s *service) Upload(ctx context.Context, stream fields.UploadServerStream) error {
 value, err := stream.RecvWithContext(ctx)
 if err != nil {
  return err
 }
 if _, err := stream.RecvWithContext(ctx); !errors.Is(err, io.EOF) {
  return errors.New("expected end of input after one item")
 }
 return stream.SendAndCloseWithContext(ctx, &fields.Value{Right: &value.Left})
}

// Watch sends the extended result and closes the stream.
func (s *service) Watch(ctx context.Context, stream fields.WatchServerStream) error {
 if err := stream.SendWithContext(ctx, &fields.ValueWatchStreamingResult{Left: "stream"}); err != nil {
  return err
 }
 return stream.Close()
}

func TestMethodContracts(t *testing.T) {
 _, widened := reflect.TypeFor[fields.Value]().FieldByName("Left")
 require.False(t, widened, "plain uses must keep the original type")
 svc := &service{}
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(fields.NewEndpoints(svc), mux,
  loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
 host := httptest.NewServer(mux)
 defer host.Close()
 ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 defer cancel()
 newClient := func(scheme string) *client.Client {
  return client.NewClient(scheme, strings.TrimPrefix(host.URL, "http://"), http.DefaultClient,
   loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
 }
 httpClient := newClient("http")
 right := "original"
 plain, err := httpClient.Plain()(ctx, &fields.Value{Right: &right})
 require.NoError(t, err)
 require.Equal(t, right, *plain.(*fields.Value).Right)
 sent, err := httpClient.Send()(ctx, &fields.ValueSendPayload{Left: "request"})
 require.NoError(t, err)
 require.Equal(t, "request", *sent.(*fields.Value).Right)
 shown, err := httpClient.Show()(ctx, &fields.Value{Right: &right})
 require.NoError(t, err)
 require.Equal(t, "response", shown.(*fields.ValueShowResult).Left)
 require.Equal(t, right, *shown.(*fields.ValueShowResult).Right)

 wsClient := newClient("ws")
 raw, err := wsClient.Upload()(ctx, nil)
 require.NoError(t, err)
 upload := raw.(fields.UploadClientStream)
 require.NoError(t, upload.SendWithContext(ctx, &fields.ValueUploadStreamingPayload{Left: "upload"}))
 uploaded, err := upload.CloseAndRecvWithContext(ctx)
 require.NoError(t, err)
 require.Equal(t, "upload", *uploaded.Right)
 raw, err = wsClient.Watch()(ctx, nil)
 require.NoError(t, err)
 watch := raw.(fields.WatchClientStream)
 watched, err := watch.RecvWithContext(ctx)
 require.NoError(t, err)
 require.Equal(t, "stream", watched.Left)
 _, err = watch.RecvWithContext(ctx)
 require.ErrorIs(t, err, io.EOF)

 for _, tc := range []struct {
  wire string
  code string
 }{
  {"{}", "missing_field"},
  {"{\"left\":null}", "missing_field"},
  {"{\"left\":\"\"}", "invalid_length"},
 } {
  t.Run(tc.wire, func(t *testing.T) {
   before := svc.calls.Load()
   req, err := http.NewRequestWithContext(ctx, "POST", host.URL+"/send", strings.NewReader(tc.wire))
   require.NoError(t, err)
   req.Header.Set("Content-Type", "application/json")
   response, err := http.DefaultClient.Do(req)
   require.NoError(t, err)
   body, readErr := io.ReadAll(response.Body)
   require.NoError(t, response.Body.Close())
   require.NoError(t, readErr)
   require.Equal(t, http.StatusBadRequest, response.StatusCode, string(body))
   require.Contains(t, string(body), "\"code\":\""+tc.code+"\"")
   require.Equal(t, before, svc.calls.Load(), "invalid inherited fields must not reach the service")
  })
 }
}
`
