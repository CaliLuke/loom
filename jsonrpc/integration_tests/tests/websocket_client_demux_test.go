package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/CaliLuke/loom/jsonrpc/integration_tests/framework"
	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
)

// wsDemuxHostEnv passes the address of the running server to the client
// harness inside the generated module.
const wsDemuxHostEnv = "LOOM_JSONRPC_WS_DEMUX_HOST"

// wsDemuxHarness drives the generated JSON-RPC WebSocket client of the testws
// service against the running generated server. Several streams of two
// methods share the connection of one client and exchange requests
// concurrently; every stream must receive exactly the responses to its own
// requests, in send order, and closing one stream must leave the others
// working.
const wsDemuxHarness = `package demuxcheck_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/loom/jsonrpc"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/gorilla/websocket"

	client "testservice/gen/jsonrpc/testws/client"
	testws "testservice/gen/testws"
)

type roundTripper interface {
	roundTrip(ctx context.Context, value string) (string, error)
	close() error
}

type echoStream struct{ s *client.EchoStringWsClientStream }

func (e echoStream) roundTrip(ctx context.Context, value string) (string, error) {
	if err := e.s.SendWithContext(ctx, &testws.EchoStringWsPayload{Value: value}); err != nil {
		return "", err
	}
	res, err := e.s.RecvWithContext(ctx)
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func (e echoStream) close() error { return e.s.Close() }

type transformStream struct{ s *client.TransformStringWsClientStream }

func (e transformStream) roundTrip(ctx context.Context, value string) (string, error) {
	if err := e.s.SendWithContext(ctx, &testws.TransformStringWsPayload{Value: value}); err != nil {
		return "", err
	}
	res, err := e.s.RecvWithContext(ctx)
	if err != nil {
		return "", err
	}
	return res.Value, nil
}

func (e transformStream) close() error { return e.s.Close() }

func TestStreamsShareOneConnection(t *testing.T) {
	var mu sync.Mutex
	var reported []string
	c := client.NewClient("ws", os.Getenv("` + wsDemuxHostEnv + `"), nil, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil,
		jsonrpc.WithErrorHandler(func(_ context.Context, kind jsonrpc.StreamErrorType, err error, _ *jsonrpc.RawResponse) {
			mu.Lock()
			defer mu.Unlock()
			reported = append(reported, fmt.Sprintf("%d: %v", kind, err))
		}))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	open := func(endpoint func() func(context.Context, any) (any, error)) any {
		raw, err := endpoint()(ctx, nil)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}
		return raw
	}
	streams := []struct {
		name   string
		stream roundTripper
		want   func(string) string
	}{
		{"echo-a", echoStream{open(func() func(context.Context, any) (any, error) { return c.EchoStringWs() }).(*client.EchoStringWsClientStream)}, func(v string) string { return v }},
		{"echo-b", echoStream{open(func() func(context.Context, any) (any, error) { return c.EchoStringWs() }).(*client.EchoStringWsClientStream)}, func(v string) string { return v }},
		{"transform", transformStream{open(func() func(context.Context, any) (any, error) { return c.TransformStringWs() }).(*client.TransformStringWsClientStream)}, strings.ToUpper},
	}
	run := func(rounds int, skip string) {
		var wg sync.WaitGroup
		for _, s := range streams {
			if s.name == skip {
				continue
			}
			wg.Go(func() {
				for n := range rounds {
					value := fmt.Sprintf("%s-%d", s.name, n)
					got, err := s.stream.roundTrip(ctx, value)
					if err != nil {
						t.Errorf("%s round %d: %v", s.name, n, err)
						return
					}
					if got != s.want(value) {
						t.Errorf("%s round %d: got %q, want %q", s.name, n, got, s.want(value))
					}
				}
			})
		}
		wg.Wait()
	}
	run(40, "")
	if err := streams[0].stream.close(); err != nil {
		t.Errorf("close %s: %v", streams[0].name, err)
	}
	run(10, streams[0].name)
	for _, s := range streams[1:] {
		if err := s.stream.close(); err != nil {
			t.Errorf("close %s: %v", s.name, err)
		}
	}
	if err := c.Close(); err != nil {
		t.Errorf("close client: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(reported) != 0 {
		t.Errorf("stream errors reported: %v", reported)
	}
}
`

// TestJSONRPCWebSocketClientStreamsShareConnection generates the testws
// WebSocket service, starts its server, and runs a harness in the generated
// module that uses the generated client with several streams on one
// connection, under the race detector (#399).
func TestJSONRPCWebSocketClientStreamsShareConnection(t *testing.T) {
	workDir := filepath.Join(t.TempDir(), "wsdemux")
	methods := make(map[string]framework.MethodInfo)
	for _, name := range []string{"echo_string_ws", "transform_string_ws"} {
		info, err := framework.ParseMethod(name)
		require.NoError(t, err)
		methods[name] = info
	}
	require.NoError(t, framework.NewGenerator(workDir, methods).Generate())

	server, err := harness.StartServer(t.Context(), workDir, 0)
	require.NoError(t, err)
	defer server.Stop() //nolint:errcheck

	checkDir := filepath.Join(workDir, "demuxcheck")
	require.NoError(t, os.MkdirAll(checkDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(checkDir, "demux_test.go"), []byte(wsDemuxHarness), 0o600))
	t.Setenv(wsDemuxHostEnv, strings.TrimPrefix(server.URL(), "http://"))
	out, err := testingx.RunCmd(workDir, "go", "test", "-race", "-count=1", "./demuxcheck/")
	require.NoError(t, err, out)
}
