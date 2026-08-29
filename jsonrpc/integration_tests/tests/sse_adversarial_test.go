package tests

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sse "github.com/tmaxmax/go-sse"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
)

func TestJSONRPCSSEExternalClientReceivesDecodeErrorsOnMessageEvent(t *testing.T) {
	t.Parallel()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	fixtureDir := filepath.Join("..", "fixtures", "ticktock")
	server, err := harness.StartServer(serverCtx, fixtureDir, 0)
	require.NoError(t, err)
	defer server.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reqBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "Tick",
		"params":  "wrong-shape",
		"id":      "bad-params-1",
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL()+"/rpc", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	conn := sse.NewConnection(req)
	eventCh := make(chan sse.Event, 1)
	errCh := make(chan error, 1)
	conn.SubscribeToAll(func(ev sse.Event) {
		select {
		case eventCh <- ev:
		default:
		}
		cancel()
	})

	go func() {
		errCh <- conn.Connect()
	}()

	var ev sse.Event
	select {
	case ev = <-eventCh:
	case err := <-errCh:
		require.NoError(t, err)
		t.Fatal("connection closed before receiving SSE error event")
	case <-ctx.Done():
		t.Fatal("timed out waiting for SSE decode error event")
	}

	require.Equal(t, "message", ev.Type)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(ev.Data), &envelope))
	require.Equal(t, "2.0", envelope["jsonrpc"])
	require.Equal(t, "bad-params-1", envelope["id"])

	errorObj, ok := envelope["error"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, -32602, errorObj["code"])
}

func TestJSONRPCSSEExternalClientReceivesOversizedErrorOnMessageEvent(t *testing.T) {
	t.Parallel()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	fixtureDir := filepath.Join("..", "fixtures", "ticktock")
	server, err := harness.StartServer(serverCtx, fixtureDir, 0)
	require.NoError(t, err)
	defer server.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body := strings.NewReader(`{"jsonrpc":"2.0","method":"Tick","params":{"id":"` +
		strings.Repeat("x", loomhttp.DefaultMaxRequestBodyBytes) + `"},"id":"oversized"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL()+"/rpc", body)
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	conn := sse.NewConnection(req)
	eventCh := make(chan sse.Event, 1)
	errCh := make(chan error, 1)
	conn.SubscribeToAll(func(event sse.Event) {
		select {
		case eventCh <- event:
		default:
		}
		cancel()
	})

	go func() {
		errCh <- conn.Connect()
	}()

	var event sse.Event
	select {
	case event = <-eventCh:
	case err := <-errCh:
		require.NoError(t, err)
		t.Fatal("connection closed before receiving oversized SSE error event")
	case <-ctx.Done():
		t.Fatal("timed out waiting for oversized SSE error event")
	}

	require.Equal(t, "message", event.Type)
	var envelope struct {
		JSONRPC string `json:"jsonrpc"`
		ID      any    `json:"id"`
		Error   struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Name string `json:"name"`
			} `json:"data"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(event.Data), &envelope))
	require.Equal(t, "2.0", envelope.JSONRPC)
	require.Nil(t, envelope.ID)
	require.Equal(t, -32600, envelope.Error.Code)
	require.Equal(t, "request body too large", envelope.Error.Message)
	require.Equal(t, "request_too_large", envelope.Error.Data.Name)
}

func TestJSONRPCSSEFixtureRegeneratesAndBuilds(t *testing.T) {
	t.Parallel()

	srcDir := filepath.Join("..", "fixtures", "ticktock")
	workDir := filepath.Join(t.TempDir(), "ticktock")
	require.NoError(t, testingx.CopyTree(srcDir, workDir))
	require.NoError(t, testingx.PinLocalReplace(workDir, testingx.RepoRoot()))

	_, err := testingx.RunCmd(workDir, "go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/ticktock/design", "-o", ".")
	require.NoError(t, err)
	testingx.RequireTreeMatches(t, filepath.Join(srcDir, "gen"), filepath.Join(workDir, "gen"))

	_, err = testingx.RunCmd(workDir, "go", "test", "./...")
	require.NoError(t, err)
}
