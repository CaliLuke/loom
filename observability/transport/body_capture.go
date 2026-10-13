package transport

import "sync"

type bodyCapture struct {
	mu      sync.Mutex
	limit   int
	data    []byte
	omitted bool
}

func (b *bodyCapture) record(data []byte, text string, n int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recordLocked(data, text, n)
}

// recordLocked retains only the reported byte prefix. The caller holds mu so
// stream-specific observation state can be updated in the same critical section.
func (b *bodyCapture) recordLocked(data []byte, text string, n int64) {
	if b.omitted {
		return
	}
	if n > int64(b.limit-len(b.data)) {
		b.data = nil
		b.omitted = true
		return
	}
	if data != nil {
		b.data = append(b.data, data[:int(n)]...)
	} else {
		b.data = append(b.data, text[:int(n)]...)
	}
}
