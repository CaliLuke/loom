package http

import (
	"context"
	"net/http"
	"sync"
)

// ResponseMetadata owns metadata admission shared by negotiated unary and SSE
// success. Set, capture and terminal closure are serialized by one owner.
type ResponseMetadata struct {
	lock     sync.Mutex
	resolve  func(context.Context) (SSEHandshake, error)
	prepared *SSEHandshake
	sealed   bool
}

// NewResponseMetadata creates the exchange owner with an absent-value resolver.
// The resolver validates optional or required metadata before publication. A nil
// resolver supplies status 200. Resolvers must not reenter this owner or its writer.
func NewResponseMetadata(resolve func(context.Context) (SSEHandshake, error)) *ResponseMetadata {
	return &ResponseMetadata{resolve: resolve}
}

// Set validates and replaces the prepared snapshot while admission is live.
// A rejected value preserves the previous snapshot; prepare must not reenter.
func (m *ResponseMetadata) Set(prepare func() (SSEHandshake, error)) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.sealed {
		return ErrSSEResponseCommitted
	}
	value, err := prepare()
	if err != nil {
		return err
	}
	value.Header = value.Header.Clone()
	m.prepared = &value
	return nil
}

// Capture closes unary metadata admission and returns a detached snapshot.
// A failed capture also closes admission; it never publishes the response.
func (m *ResponseMetadata) Capture(ctx context.Context) (SSEHandshake, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	value, err := m.prepareLocked(ctx)
	m.sealed = true
	return value, err
}

// Close closes admission without publication, including handler error paths.
func (m *ResponseMetadata) Close() {
	m.lock.Lock()
	m.sealed = true
	m.lock.Unlock()
}

// WithSSEResponseMetadata attaches the exchange's existing metadata owner to the
// SSE writer. Both use the same admission lock; metadata must be nonnil.
func WithSSEResponseMetadata(metadata *ResponseMetadata) SSEStreamOption {
	if metadata == nil {
		panic("SSE response metadata is nil")
	}
	return func(s *SSEStreamWriter) {
		s.metadata = metadata
	}
}

// prepareLocked preserves retryable SSE preparation failures. The successful
// publisher or terminal caller seals admission while still holding this lock.
func (m *ResponseMetadata) prepareLocked(ctx context.Context) (SSEHandshake, error) {
	if m.sealed {
		return SSEHandshake{}, ErrSSEResponseCommitted
	}
	value := SSEHandshake{Status: http.StatusOK}
	switch {
	case m.prepared != nil:
		value = *m.prepared
	case m.resolve != nil:
		var err error
		value, err = m.resolve(ctx)
		if err != nil {
			return SSEHandshake{}, err
		}
	}
	value.Header = value.Header.Clone()
	return value, nil
}
