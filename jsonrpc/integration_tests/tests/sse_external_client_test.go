package tests

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
)

func TestJSONRPCSSEInteroperatesWithExternalClient(t *testing.T) {
	t.Parallel()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	srcDir := filepath.Join("..", "fixtures", "ticktock")
	workDir := filepath.Join(t.TempDir(), "ticktock")
	require.NoError(t, testingx.CopyTree(srcDir, workDir))
	require.NoError(t, testingx.PinLocalReplace(workDir, testingx.RepoRoot()))
	server, err := harness.StartServer(serverCtx, workDir, 0)
	require.NoError(t, err)
	defer server.Stop() //nolint:errcheck

	baseURL := server.URL()

	for _, tc := range []struct {
		name   string
		params map[string]any
	}{
		{"tick omitted optional member", map[string]any{}},
		{"tick supplied optional member", map[string]any{"label": "hello"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifyExternalSSEStream(t, baseURL, "Tick", "Tick-request", tc.params, []externalSSEExpectation{
				{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tick-1"}},
				{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tick-2"}},
				{EventType: "message", EnvelopeField: "result", Value: map[string]any{"value": "tick-done"}},
			})
		})
	}

	t.Run("tock", func(t *testing.T) {
		verifyExternalSSEStream(t, baseURL, "Tock", "Tock-request", map[string]any{}, []externalSSEExpectation{
			{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tock-a"}},
			{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tock-b"}},
			{EventType: "message", EnvelopeField: "result", Value: map[string]any{"value": "tock-finished"}},
		})
	})
}

type externalSSEExpectation struct {
	EventType     string
	EnvelopeField string
	Value         any
}

func verifyExternalSSEStream(
	t *testing.T,
	baseURL, method, id string,
	params map[string]any,
	expected []externalSSEExpectation,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reqBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      id,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/rpc", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	got, err := collectExternalSSE(req, len(expected))
	require.NoError(t, err)

	require.Len(t, got, len(expected))
	for i, exp := range expected {
		require.Equal(t, exp.EventType, got[i].Type)

		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(got[i].Data), &envelope))
		require.Equal(t, "2.0", envelope["jsonrpc"])
		if exp.EnvelopeField == "result" {
			require.Equal(t, id, envelope["id"])
		}
		require.Equal(t, exp.Value, envelope[exp.EnvelopeField])
	}
}
