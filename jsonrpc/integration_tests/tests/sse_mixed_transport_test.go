package tests

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
)

// The mixedtick fixture copy is generated once for the whole package:
// regeneration (loom gen via go run) dominates the mixed-transport tests'
// runtime and the generated tree is read-only for the servers, so the three
// tests share it and only the servers start per test.
var (
	mixedTickOnce     sync.Once
	mixedTickTempRoot string
	mixedTickWorkDir  string
	mixedTickSetupErr error
	mixedTickStartMu  sync.Mutex
)

func TestMain(m *testing.M) {
	code := m.Run()
	if mixedTickTempRoot != "" {
		if err := os.RemoveAll(mixedTickTempRoot); err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove mixedtick work dir: %v\n", err)
		}
	}
	os.Exit(code)
}

// setupMixedTickWorkDir copies, pins, and regenerates the mixedtick fixture
// exactly once and returns the shared work dir.
func setupMixedTickWorkDir() (string, error) {
	mixedTickOnce.Do(func() {
		srcDir := filepath.Join("..", "fixtures", "mixedtick")
		tempRoot, err := os.MkdirTemp("", "mixedtick-tests-")
		if err != nil {
			mixedTickSetupErr = err
			return
		}
		mixedTickTempRoot = tempRoot
		workDir := filepath.Join(tempRoot, "mixedtick")
		if err := testingx.CopyTree(srcDir, workDir); err != nil {
			mixedTickSetupErr = err
			return
		}
		if err := testingx.PinLocalReplace(workDir, testingx.RepoRoot()); err != nil {
			mixedTickSetupErr = err
			return
		}
		if _, err := testingx.RunCmd(workDir, "go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/mixedtick/design", "-o", "."); err != nil {
			mixedTickSetupErr = err
			return
		}
		// Tidy once so the per-server tidy in harness.StartServer is a no-op
		// and the parallel tests do not race on module file writes.
		if _, err := testingx.RunCmd(workDir, "go", "mod", "tidy"); err != nil {
			mixedTickSetupErr = err
			return
		}
		mixedTickWorkDir = workDir
	})
	return mixedTickWorkDir, mixedTickSetupErr
}

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

func TestJSONRPCMixedRejectsOversizedRequestBody(t *testing.T) {
	t.Parallel()

	server := startMixedTickServer(t)
	defer server.Stop() //nolint:errcheck

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body := strings.NewReader(`{"jsonrpc":"2.0","method":"initialize","params":{"id":"` +
		strings.Repeat("x", loomhttp.DefaultMaxRequestBodyBytes) + `"},"id":"oversized"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL()+"/rpc", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, resp.Body.Close())
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
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
	require.NoError(t, json.UnmarshalRead(resp.Body, &envelope))
	require.Equal(t, "2.0", envelope.JSONRPC)
	require.Nil(t, envelope.ID)
	require.Equal(t, -32600, envelope.Error.Code)
	require.Equal(t, "request body too large", envelope.Error.Message)
	require.Equal(t, "request_too_large", envelope.Error.Data.Name)
}

func TestJSONRPCMixedSSEStreamsEnvelopeDecodeErrors(t *testing.T) {
	t.Parallel()

	server := startMixedTickServer(t)
	defer server.Stop() //nolint:errcheck

	t.Run("oversized body", func(t *testing.T) {
		body := strings.NewReader(`{"jsonrpc":"2.0","method":"Tick","params":{"id":"` +
			strings.Repeat("x", loomhttp.DefaultMaxRequestBodyBytes) + `"},"id":"oversized"}`)
		resp, respBody := postMixedTickSSERawRequest(t, server.URL()+"/rpc", body)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"))

		events, err := loomhttp.ParseSSEStream(bytes.NewReader(respBody))
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.Equal(t, "message", events[0].Type)

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
		require.NoError(t, json.Unmarshal([]byte(events[0].Data), &envelope))
		require.Equal(t, "2.0", envelope.JSONRPC)
		require.Nil(t, envelope.ID)
		require.Equal(t, -32600, envelope.Error.Code)
		require.Equal(t, "request body too large", envelope.Error.Message)
		require.Equal(t, "request_too_large", envelope.Error.Data.Name)
	})

	t.Run("malformed envelope", func(t *testing.T) {
		body := strings.NewReader(`{"jsonrpc":"2.0","method":`)
		resp, respBody := postMixedTickSSERawRequest(t, server.URL()+"/rpc", body)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream"))

		events, err := loomhttp.ParseSSEStream(bytes.NewReader(respBody))
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.Equal(t, "message", events[0].Type)

		var envelope struct {
			JSONRPC string `json:"jsonrpc"`
			ID      any    `json:"id"`
			Error   struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		require.NoError(t, json.Unmarshal([]byte(events[0].Data), &envelope))
		require.Equal(t, "2.0", envelope.JSONRPC)
		require.Nil(t, envelope.ID)
		require.Equal(t, -32700, envelope.Error.Code)
		require.Equal(t, "Parse error", envelope.Error.Message)
	})
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

	workDir, err := setupMixedTickWorkDir()
	require.NoError(t, err)
	// Tree comparison is cheap; keep the fixture-freshness assertion per
	// test so every test reports drift, not just the one that generated.
	srcDir := filepath.Join("..", "fixtures", "mixedtick")
	testingx.RequireTreeMatches(t, filepath.Join(srcDir, "gen"), filepath.Join(workDir, "gen"))

	serverCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Serialize server startup: StartServer runs go mod tidy + go run in the
	// shared work dir, and concurrent module-file writes would race.
	mixedTickStartMu.Lock()
	defer mixedTickStartMu.Unlock()
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

func postMixedTickSSERawRequest(t *testing.T, url string, body io.Reader) (*http.Response, []byte) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		resp.Body.Close() //nolint:errcheck
	})

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, respBody
}
