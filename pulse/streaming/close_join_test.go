package streaming

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCloseJoinsEveryCaller(t *testing.T) {
	for _, kind := range []string{"reader", "sink"} {
		t.Run(kind, func(t *testing.T) {
			rdb := startTestRedis(t)
			gate := &subscriberReadGate{entered: make(chan struct{}), release: make(chan struct{})}
			rdb.AddHook(gate)
			owner := newSubscriberOwner(t, rdb, kind)
			owner.subscribe()
			<-gate.entered
			first, second := make(chan struct{}), make(chan struct{})
			var release sync.Once
			t.Cleanup(func() {
				release.Do(func() {
					close(gate.release)
				})
				for _, done := range []chan struct{}{first, second} {
					select {
					case <-done:
					case <-time.After(time.Second):
						t.Error("Close did not join released worker")
					}
				}
			})
			go func() {
				owner.close()
				close(first)
			}()
			require.Eventually(t, owner.closing, time.Second, time.Millisecond)
			go func() {
				owner.close()
				close(second)
			}()
			select {
			case <-second:
				t.Error("second Close returned while the owner still held a live worker")
			case <-time.After(100 * time.Millisecond):
			}
			release.Do(func() {
				close(gate.release)
			})
		})
	}
}
