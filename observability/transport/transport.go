package transport

import "context"

type (
	// Observer receives framework transport lifecycle events.
	// Observer implementations must be safe for concurrent use.
	Observer interface {
		// ObserveEvent is invoked with a context that is alive for the
		// request producing the event. Implementations must not retain ctx
		// past the call, and must treat e as read-only.
		ObserveEvent(ctx context.Context, e Event)
	}

	// ObserverFunc adapts an ordinary function into an Observer.
	ObserverFunc func(ctx context.Context, e Event)
)

type observerKey struct{}

// ObserveEvent calls f with the supplied event. A nil ObserverFunc is a
// no-op.
func (f ObserverFunc) ObserveEvent(ctx context.Context, e Event) {
	if f == nil {
		return
	}
	f(ctx, e)
}

// WithObserver returns a copy of ctx that carries o. A nil observer clears
// any observer already attached to ctx.
func WithObserver(ctx context.Context, o Observer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, observerKey{}, o)
}

// ObserverFromContext returns the Observer attached to ctx, or nil if none
// has been injected.
func ObserverFromContext(ctx context.Context) Observer {
	if ctx == nil {
		return nil
	}
	o, _ := ctx.Value(observerKey{}).(Observer)
	return o
}

// Observe delivers e to the Observer carried by ctx, if any. Observe is the
// single entry point transport runtimes and generated adapters use to emit
// events so that a missing observer remains a cheap no-op.
func Observe(ctx context.Context, e Event) {
	o := ObserverFromContext(ctx)
	if o == nil {
		return
	}
	o.ObserveEvent(ctx, e)
}
