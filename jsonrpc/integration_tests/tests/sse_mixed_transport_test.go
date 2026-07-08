package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
)

func TestJSONRPCMixedSSETopLevelServerEmitsFinalResponse(t *testing.T) {
	t.Parallel()

	server := startMixedTickServer(t)
	defer server.Stop() //nolint:errcheck

	resp, body := postMixedTickSSERequest(t, server.URL()+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "Tick",
		"params":  map[string]any{},
		"id":      "tick-request-1",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"))
	require.NotEmpty(t, body)

	events, err := loomhttp.ParseSSEStream(bytes.NewReader(body))
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "message", events[0].Type)
	require.Equal(t, "message", events[1].Type)

	var notification map[string]any
	require.NoError(t, json.Unmarshal([]byte(events[0].Data), &notification))
	require.Equal(t, "2.0", notification["jsonrpc"])
	require.Equal(t, "Tick", notification["method"])
	require.Equal(t, map[string]any{"value": "tick-1"}, notification["params"])

	var response map[string]any
	require.NoError(t, json.Unmarshal([]byte(events[1].Data), &response))
	require.Equal(t, "2.0", response["jsonrpc"])
	require.Equal(t, "tick-request-1", response["id"])
	require.Equal(t, map[string]any{"value": "tick-done"}, response["result"])
}

func TestJSONRPCMixedSSERejectsCrossOriginPost(t *testing.T) {
	t.Parallel()

	server := startMixedTickServer(t)
	defer server.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reqBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "Tick",
		"params":  map[string]any{},
		"id":      "cross-origin-1",
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL()+"/rpc", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://attacker.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestJSONRPCMixedSSETopLevelServerEmitsErrors(t *testing.T) {
	t.Parallel()

	server := startMixedTickServer(t)
	defer server.Stop() //nolint:errcheck

	resp, body := postMixedTickSSERequest(t, server.URL()+"/rpc", map[string]any{
		"jsonrpc": "2.0",
		"method":  "Tick",
		"params":  "wrong-shape",
		"id":      "tick-error-1",
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"))
	require.NotEmpty(t, body)

	events, err := loomhttp.ParseSSEStream(bytes.NewReader(body))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "message", events[0].Type)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(events[0].Data), &envelope))
	require.Equal(t, "2.0", envelope["jsonrpc"])
	require.Equal(t, "tick-error-1", envelope["id"])

	errorObj, ok := envelope["error"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, -32602, errorObj["code"])
}

func startMixedTickServer(t *testing.T) *harness.Server {
	t.Helper()

	srcDir := filepath.Join("..", "fixtures", "mixedtick")
	workDir := filepath.Join(t.TempDir(), "mixedtick")
	require.NoError(t, testingx.CopyTree(srcDir, workDir))
	require.NoError(t, testingx.PinLocalReplace(workDir, testingx.RepoRoot()))

	_, err := testingx.RunCmd(workDir, "go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/mixedtick/design", "-o", ".")
	require.NoError(t, err)
	testingx.RequireTreeMatches(t, filepath.Join(srcDir, "gen"), filepath.Join(workDir, "gen"))

	serverCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	server, err := harness.StartServer(serverCtx, workDir, 0)
	require.NoError(t, err)
	return server
}

func postMixedTickSSERequest(t *testing.T, url string, payload map[string]any) (*http.Response, []byte) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reqBody, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		resp.Body.Close() //nolint:errcheck
	})

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, body
}
