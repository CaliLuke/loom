package pool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
)

func TestShutdownWatcherReconcilesExistingRequest(t *testing.T) {
	rdb := startTestRedis(t)
	shutdown, err := rmap.Join(t.Context(), "shutdown-watcher", rdb)
	require.NoError(t, err)
	t.Cleanup(shutdown.Close)

	// Commit and replicate before the watcher can subscribe. There will be no
	// notification for this revision after subscription.
	_, err = shutdown.SetAndWait(t.Context(), "shutdown", "requester")
	require.NoError(t, err)
	node := &Node{
		nodeShutdownMap:   shutdown,
		workerShutdownTTL: time.Second,
		logger:            pulse.NoopLogger(),
		stop:              make(chan struct{}),
		closed:            make(chan struct{}),
	}
	t.Cleanup(func() {
		require.NoError(t, node.Close(context.Background()))
	})
	node.wg.Add(1)
	go node.watchShutdown(t.Context())
	select {
	case <-node.closed:
		require.True(t, node.IsShutdown(), "closed must publish completed shutdown state")
	case <-time.After(3 * time.Second):
		t.Error("watcher missed the shutdown request replicated before subscription")
	}
}

func TestShutdownRecordedDuringConcurrentClose(t *testing.T) {
	rdb := startTestRedis(t)
	shutdown, err := rmap.Join(t.Context(), "shutdown-concurrent-close", rdb)
	require.NoError(t, err)
	t.Cleanup(shutdown.Close)
	_, err = shutdown.SetAndWait(t.Context(), "shutdown", "requester")
	require.NoError(t, err)
	node := &Node{
		nodeShutdownMap:   shutdown,
		workerShutdownTTL: time.Second,
		logger:            pulse.NoopLogger(),
		stop:              make(chan struct{}),
		closed:            make(chan struct{}),
	}
	// Hold the watcher slot while local Close starts and waits for it.
	node.wg.Add(1)
	closed := make(chan error, 1)
	go func() {
		closed <- node.Close(t.Context())
	}()
	<-node.stop
	go node.watchShutdown(t.Context())
	select {
	case err := <-closed:
		require.NoError(t, err)
		require.True(t, node.IsShutdown())
	case <-time.After(3 * time.Second):
		t.Fatal("local Close did not join the shutdown watcher")
	}
	// Idempotent local close cannot erase the observed pool shutdown.
	require.NoError(t, node.Close(t.Context()))
	require.True(t, node.IsShutdown())
}
