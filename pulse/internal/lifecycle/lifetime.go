// Package lifecycle owns local stop admission and synchronous cleanup joining.
package lifecycle

import (
	"sync"
	"sync/atomic"
)

// Lifetime separates stop signaling from cleanup. Its zero value is ready to use.
// Stop callbacks must not acquire locks held during I/O or wait for owned workers.
type Lifetime struct {
	stopping atomic.Bool
	stop     sync.Once
	join     sync.Once
}

// Stopping reports whether admission has been sealed.
func (l *Lifetime) Stopping() bool {
	return l.stopping.Load()
}

// Stop seals admission and runs the cancellation callback once. It does not join
// workers, so an owned worker may call Stop without waiting for itself.
func (l *Lifetime) Stop(cancel func()) {
	l.stop.Do(func() {
		l.stopping.Store(true)
		cancel()
	})
}

// Close stops admission and runs cleanup once. Every caller waits for that same
// cleanup to finish, including callers arriving after shutdown has started.
func (l *Lifetime) Close(cancel, cleanup func()) {
	l.Stop(cancel)
	l.join.Do(cleanup)
}
