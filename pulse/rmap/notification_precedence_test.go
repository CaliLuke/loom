package rmap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNotificationPendingPrecedence(t *testing.T) {
	for _, first := range []EventKind{EventChange, EventDelete, EventReset} {
		for _, next := range []EventKind{EventChange, EventDelete, EventReset} {
			t.Run(eventTestName(first)+"/"+eventTestName(next), func(t *testing.T) {
				m := testMap()
				c := m.Subscribe()
				m.lock.Lock()
				m.notifySubscribers(first)
				m.notifySubscribers(next)
				m.lock.Unlock()
				require.Equal(t, strongerEventKind(first, next), <-c)
				require.Empty(t, c)
				m.Unsubscribe(c)
			})
		}
	}
}

func TestNotificationConsumedKindDoesNotCarryForward(t *testing.T) {
	m := testMap()
	c := m.Subscribe()
	m.lock.Lock()
	m.notifySubscribers(EventReset)
	m.lock.Unlock()
	require.Equal(t, EventReset, <-c)
	m.lock.Lock()
	m.notifySubscribers(EventChange)
	m.lock.Unlock()
	require.Equal(t, EventChange, <-c)
	m.Unsubscribe(c)
}

func TestNotificationLiveResetSurvivesPendingChange(t *testing.T) {
	m := joinMap(t, startTestRedis(t), "notification-precedence")
	c := m.Subscribe()
	_, err := m.SetAndWait(t.Context(), "job", "pending")
	require.NoError(t, err)
	require.NoError(t, m.Reset(t.Context()))
	require.Eventually(t, func() bool {
		return m.Len() == 0
	}, 2*time.Second, time.Millisecond)
	require.Equal(t, EventReset, <-c)
	m.Unsubscribe(c)
}

func TestNotificationSubscribersRemainIndependent(t *testing.T) {
	m := testMap()
	slow := m.Subscribe()
	m.lock.Lock()
	m.notifySubscribers(EventReset)
	m.lock.Unlock()
	fresh := m.Subscribe()
	m.lock.Lock()
	for range 1000 {
		m.notifySubscribers(EventChange)
	}
	m.lock.Unlock()
	require.Equal(t, EventReset, <-slow)
	require.Equal(t, EventChange, <-fresh)
	require.Empty(t, slow)
	require.Empty(t, fresh)
	m.Unsubscribe(slow)
	m.Unsubscribe(fresh)
}

func TestNotificationConcurrentReceiverRetainsReset(t *testing.T) {
	m := testMap()
	c := m.Subscribe()
	resets := make(chan int, 1)
	go func() {
		count := 0
		for kind := range c {
			if kind == EventReset {
				count++
			}
		}
		resets <- count
	}()
	m.lock.Lock()
	m.notifySubscribers(EventReset)
	m.lock.Unlock()
	for range 1000 {
		m.lock.Lock()
		m.notifySubscribers(EventChange)
		m.lock.Unlock()
	}
	m.Unsubscribe(c)
	select {
	case count := <-resets:
		require.Equal(t, 1, count)
	case <-time.After(time.Second):
		t.Error("receiver did not finish after unsubscribe")
	}
}

func TestNotificationConcurrentUnsubscribe(t *testing.T) {
	m := testMap()
	c := m.Subscribe()
	started, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		close(started)
		for range 1000 {
			m.lock.Lock()
			m.notifySubscribers(EventReset)
			m.lock.Unlock()
		}
	}()
	<-started
	m.Unsubscribe(c)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("publisher blocked during unsubscribe")
	}
	// Unsubscribe may leave one buffered event, which remains readable.
	for range c {
	}
}

func eventTestName(kind EventKind) string {
	switch kind {
	case EventReset:
		return "reset"
	case EventDelete:
		return "delete"
	default:
		return "change"
	}
}
