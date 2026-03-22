package tests

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	sse "github.com/tmaxmax/go-sse"

	"github.com/CaliLuke/loom/http/integration_tests/harness"
)

func TestHTTPSSEFixtureStreamsToExternalClient(t *testing.T) {
	t.Parallel()

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	fixtureDir := filepath.Join("..", "fixtures", "ticktock")
	server, err := harness.StartServer(serverCtx, fixtureDir, 0)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, server.Stop())
	}()

	baseURL := server.URL()

	t.Run("tick stream", func(t *testing.T) {
		verifyTickTockStream(t, baseURL+"/tick", []sseExpectation{
			{EventType: "tick", Data: "tick-1"},
			{EventType: "tick", Data: "tick-2"},
			{EventType: "tick", Data: "tick-done"},
		})
	})

	t.Run("tock stream", func(t *testing.T) {
		verifyTickTockStream(t, baseURL+"/tock", []sseExpectation{
			{EventType: "tock", Data: "tock-a"},
			{EventType: "tock", Data: "tock-b"},
			{EventType: "tock", Data: "tock-done"},
		})
	})
}

type sseExpectation struct {
	EventType string
	Data      string
}

func verifyTickTockStream(t *testing.T, url string, expected []sseExpectation) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)

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

	errc := make(chan error, 1)
	go func() {
		errc <- conn.Connect()
	}()

	var connectErr error
	select {
	case <-done:
		connectErr = <-errc
	case err := <-errc:
		connectErr = err
	case <-ctx.Done():
		t.Fatalf("timed out waiting for SSE events from %s", url)
	}
	require.True(t, connectErr == nil || connectErr == context.Canceled, "unexpected connection result: %v", connectErr)

	mu.Lock()
	got := append([]sse.Event(nil), events...)
	mu.Unlock()

	require.Len(t, got, len(expected))
	for i, exp := range expected {
		require.Equal(t, exp.EventType, got[i].Type)
		require.Equal(t, exp.Data, got[i].Data)
	}
}
