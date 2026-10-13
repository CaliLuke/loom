package transport

import (
	"context"
	"sync"
	"time"
)

type (
	// StreamObserver retains immutable request identity independently of the
	// request span. Transport owners record physical lifecycle transitions under
	// their own locks, then call Drain outside those locks. At most one opening
	// and one closing event are retained; failures are delivered synchronously.
	StreamObserver struct {
		ctx      context.Context
		identity Event
		start    time.Time
		capture  *ResponseCapture
		mu       sync.Mutex
		opened   bool
		closed   bool
		draining bool
		open     *Event
		close    *Event
	}

	streamObserverKey struct {
	}
)

// NewStreamObserver snapshots the request identity attached to ctx. If none is
// attached, kind identifies the transport and other identity fields stay empty.
// Ending or changing the request observer does not change this binding.
func NewStreamObserver(ctx context.Context, kind TransportKind) *StreamObserver {
	if ctx == nil {
		ctx = context.Background()
	}
	s := &StreamObserver{ctx: ctx, identity: Event{Transport: kind}, start: time.Now()}
	if o := RequestObserverFromContext(ctx); o != nil {
		o.mu.Lock()
		s.identity = o.eventLocked(EventKindStreamOpen, ReasonOK)
		s.start, s.capture = o.start, o.capture
		o.mu.Unlock()
	}
	return s
}

// WithStreamObserver attaches immutable stream identity without changing ctx's
// cancellation lifetime. Retained operation wrappers use their own binding even
// when later writes receive a different per-call context.
func WithStreamObserver(ctx context.Context, observer *StreamObserver) context.Context {
	return context.WithValue(ctx, streamObserverKey{}, observer)
}

// StreamObserverFromContext returns a retained binding, or snapshots the request
// observer when no binding is attached. It never changes a connection binding.
func StreamObserverFromContext(ctx context.Context, kind TransportKind) *StreamObserver {
	if ctx != nil {
		if s, ok := ctx.Value(streamObserverKey{}).(*StreamObserver); ok && s != nil {
			return s
		}
	}
	return NewStreamObserver(ctx, kind)
}

// ForContext selects an explicitly retained operation binding from ctx, falling
// back to this connection binding. It does not infer identity from arbitrary
// inbound request observers carried into an outgoing client operation.
func (s *StreamObserver) ForContext(ctx context.Context) *StreamObserver {
	if ctx != nil {
		if binding, ok := ctx.Value(streamObserverKey{}).(*StreamObserver); ok && binding != nil {
			return binding
		}
	}
	return s
}

// JSONRPC returns a separate immutable operation binding. It shares connection
// identity and the observer, but not physical lifecycle state or method identity.
func (s *StreamObserver) JSONRPC(method, id string, notification bool) *StreamObserver {
	binding := &StreamObserver{ctx: s.ctx, identity: s.identity, start: time.Now(), capture: s.capture}
	binding.identity.Method = method
	binding.identity.JSONRPCMethod = method
	binding.identity.JSONRPCID = id
	binding.identity.Notification = notification
	return binding
}

// RecordOpen records successful stream commitment once, without calling user
// code. A later Drain delivers the event on its caller's stack.
func (s *StreamObserver) RecordOpen() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opened {
		return
	}
	s.opened = true
	event := s.event(EventKindStreamOpen, ReasonOK)
	s.open = &event
}

// RecordClose records physical closure only after a recorded opening. Closing
// an unopened stream and repeated closure do not produce events.
func (s *StreamObserver) RecordClose() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.opened || s.closed {
		return
	}
	s.closed = true
	event := s.event(EventKindStreamClose, ReasonOK)
	s.close = &event
}

// Drain synchronously delivers pending lifecycle events outside all observer
// locks. Concurrent or reentrant drains return; the active drainer delivers any
// newly recorded closure after opening. A callback panic propagates and releases
// the drain claim, leaving remaining metadata available to foreground cleanup.
func (s *StreamObserver) Drain() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.draining {
		s.mu.Unlock()
		return
	}
	s.draining = true
	s.mu.Unlock()
	claimed := true
	defer func() {
		if !claimed {
			return
		}
		s.mu.Lock()
		s.draining = false
		s.mu.Unlock()
	}()
	for {
		s.mu.Lock()
		event := s.open
		if event != nil {
			s.open = nil
		} else {
			event, s.close = s.close, nil
		}
		if event == nil {
			// Release atomically with checking for pending work. Otherwise a
			// concurrent recorder's Drain could return before this release.
			s.draining = false
			claimed = false
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		Observe(s.ctx, *event)
	}
}

// Failure synchronously reports a write/flush failure without changing physical
// lifecycle or request classification. Call it outside transport ownership locks.
func (s *StreamObserver) Failure(reason Reason) {
	if s != nil {
		Observe(s.ctx, s.event(EventKindStreamFailure, reason))
	}
}

// Suppressed reports the existing logical final-response suppression marker.
// It does not claim or deduplicate the physical connection's closing event.
func (s *StreamObserver) Suppressed() {
	if s != nil {
		Observe(s.ctx, s.event(EventKindStreamClose, ReasonStreamFinalResponseSuppressed))
	}
}

func (s *StreamObserver) event(kind EventKind, reason Reason) Event {
	event := s.identity
	event.Kind, event.Reason, event.Duration = kind, reason, time.Since(s.start)
	if s.capture != nil {
		event.StatusCode = s.capture.StatusCode()
		event.BytesWritten = s.capture.BytesWritten()
	}
	return event
}
