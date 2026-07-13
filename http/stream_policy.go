package http

import (
	"errors"
	"time"
)

type (
	// StreamWritePolicy configures the maximum duration of each server-stream
	// network write or flush. The zero value preserves unbounded writes.
	StreamWritePolicy struct {
		timeout time.Duration
	}
)

var (
	// ErrInvalidStreamWriteTimeout is returned when a stream write timeout is negative.
	ErrInvalidStreamWriteTimeout = errors.New("loom http stream write timeout must not be negative")
	// ErrStreamWriteDeadlineUnsupported is returned when a streaming transport
	// cannot install the deadline required by a write policy or caller context.
	ErrStreamWriteDeadlineUnsupported = errors.New("loom http stream write deadline unsupported")
)

// NewStreamWritePolicy returns an immutable server-stream write policy.
func NewStreamWritePolicy(timeout time.Duration) (StreamWritePolicy, error) {
	if timeout < 0 {
		return StreamWritePolicy{}, ErrInvalidStreamWriteTimeout
	}
	return StreamWritePolicy{timeout: timeout}, nil
}

// Timeout returns the maximum duration allowed for each write or flush.
func (p StreamWritePolicy) Timeout() time.Duration {
	return p.timeout
}
