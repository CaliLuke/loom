package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sse "github.com/tmaxmax/go-sse"

	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
)

func TestJSONRPCSSEInteroperatesWithExternalClient(t *testing.T) {
	t.Parallel()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	fixtureDir := filepath.Join("..", "fixtures", "ticktock")
	server, err := harness.StartServer(serverCtx, fixtureDir, 0)
	require.NoError(t, err)
	defer server.Stop() //nolint:errcheck

	baseURL := server.URL()

	t.Run("tick", func(t *testing.T) {
		verifyExternalSSEStream(t, baseURL, "Tick", "Tick-request", []externalSSEExpectation{
			{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tick-1"}},
			{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tick-2"}},
			{EventType: "response", EnvelopeField: "result", Value: map[string]any{"value": "tick-done"}},
		})
	})

	t.Run("tock", func(t *testing.T) {
		verifyExternalSSEStream(t, baseURL, "Tock", "Tock-request", []externalSSEExpectation{
			{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tock-a"}},
			{EventType: "message", EnvelopeField: "params", Value: map[string]any{"value": "tock-b"}},
			{EventType: "response", EnvelopeField: "result", Value: map[string]any{"value": "tock-finished"}},
		})
	})
}

type externalSSEExpectation struct {
	EventType     string
	EnvelopeField string
	Value         any
}

func verifyExternalSSEStream(t *testing.T, baseURL string, method string, id string, expected []externalSSEExpectation) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reqBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  map[string]any{},
		"id":      id,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/rpc", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	conn := sse.NewConnection(req)

	var (
		mu     sync.Mutex
		events []sse.Event
	)
	done := make(chan struct{})

	conn.SubscribeToAll(func(ev sse.Event) {
		mu.Lock()
		defer mu.Unlock()

		events = append(events, ev)
		if len(events) == len(expected) {
			select {
			case <-done:
			default:
				close(done)
			}
			cancel()
		}
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- conn.Connect()
	}()

	var connectErr error
	select {
	case <-done:
		connectErr = <-errCh
	case err := <-errCh:
		connectErr = err
	case <-ctx.Done():
		t.Fatalf("timed out waiting for SSE events for %s", method)
	}
	require.True(t, connectErr == nil || connectErr == context.Canceled, "unexpected connection result: %v", connectErr)

	mu.Lock()
	got := append([]sse.Event(nil), events...)
	mu.Unlock()

	require.Len(t, got, len(expected))
	for i, exp := range expected {
		require.Equal(t, exp.EventType, got[i].Type)

		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(got[i].Data), &envelope))
		require.Equal(t, "2.0", envelope["jsonrpc"])
		if exp.EventType == "response" {
			require.Equal(t, id, envelope["id"])
		}
		require.Equal(t, exp.Value, envelope[exp.EnvelopeField])
	}
}
