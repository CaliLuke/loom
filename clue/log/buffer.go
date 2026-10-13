package log

import "sync"

type (
	// logBuffer owns one logging scope's queue, flush state and output lock.
	// Callers acquire a logger lock before this lock, never the reverse.
	logBuffer struct {
		lock    sync.Mutex
		entries []bufferedEntry
		flushed bool
	}

	// bufferedEntry retains the immutable output configuration at admission.
	bufferedEntry struct {
		entry   *Entry
		outputs []Output
	}
)

func (e bufferedEntry) write() {
	for _, out := range e.outputs {
		out.Writer.Write(out.Format(e.entry)) // nolint: errcheck
	}
}

func (b *logBuffer) flush() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.flushLocked()
}

func (b *logBuffer) flushLocked() {
	if b.flushed {
		return
	}
	for _, e := range b.entries {
		e.write()
	}
	b.entries = nil
	b.flushed = true
}

func (b *logBuffer) log(e bufferedEntry, buffer, debug bool) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if debug {
		b.flushLocked()
	}
	if b.flushed || !buffer {
		e.write()
		return
	}
	b.entries = append(b.entries, e)
}
