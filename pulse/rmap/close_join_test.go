package rmap

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCloseJoinsEveryCaller(t *testing.T) {
	rdb := startTestRedis(t)
	sm, err := Join(t.Context(), "close-join", rdb)
	require.NoError(t, err)
	// Keep one owned operation alive after cancellation to distinguish stopping
	// from completed cleanup without depending on a Redis timeout.
	sm.wait.Add(1)
	first, second := make(chan struct{}), make(chan struct{})
	var release sync.Once
	t.Cleanup(func() {
		release.Do(sm.wait.Done)
		for _, done := range []chan struct{}{first, second} {
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Error("Close did not join released operation")
			}
		}
	})
	go func() {
		sm.Close()
		close(first)
	}()
	<-sm.closectx.Done()
	go func() {
		sm.Close()
		close(second)
	}()
	select {
	case <-second:
		t.Error("second Close returned while map cleanup was incomplete")
	case <-time.After(100 * time.Millisecond):
	}
	release.Do(sm.wait.Done)
}

func TestCloseJoinsSubscriberAdmission(t *testing.T) {
	rdb := startTestRedis(t)
	for range 20 {
		sm := joinMap(t, rdb, "close-subscriber-admission")
		start := make(chan struct{})
		var workers sync.WaitGroup
		workers.Add(8)
		for range 8 {
			go func() {
				defer workers.Done()
				<-start
				ch := sm.Subscribe()
				if ch != nil {
					sm.Unsubscribe(ch)
					select {
					case _, open := <-ch:
						require.False(t, open)
					case <-time.After(time.Second):
						t.Error("admitted subscriber not closed")
					}
				}
			}()
		}
		close(start)
		sm.Close()
		workers.Wait()
		require.Nil(t, sm.Subscribe())
	}
}
