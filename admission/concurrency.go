// Package admission provides request admission controls for existing middleware
// hooks. Applications own budgets, grouping, placement and transport mappings.
package admission

import (
	"context"
	"errors"
	"sync"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// Concurrency bounds simultaneous endpoint invocations. Construct it with
	// NewConcurrency and share the pointer across the endpoints in one budget.
	// It must not be copied. Completion, including panic unwinding, releases the
	// permit; cancellation alone does not. It is unsuitable for streams or work
	// that continues after the wrapped endpoint returns.
	Concurrency struct {
		mu    sync.Mutex
		state ConcurrencySnapshot
	}

	// ConcurrencySnapshot is a coherent local observation with no request labels.
	// Counters cover calls reaching this admission boundary since construction.
	ConcurrencySnapshot struct {
		// Capacity is the current positive admission limit.
		Capacity int
		// InFlight counts invocations that have acquired a permit and not exited.
		// It may exceed Capacity after a reduction.
		InFlight int
		// Admitted counts acquired permits, including calls that later fail or panic.
		Admitted uint64
		// Rejected counts calls denied because capacity was unavailable.
		Rejected uint64
		// Canceled counts calls denied because their context was already canceled
		// or expired at the admission decision. These are not also Rejected.
		Canceled uint64
	}
)

// NewConcurrency constructs an admission budget. Capacity must be positive;
// invalid configuration returns an error before any middleware is constructed.
// No queues, worker goroutines, retries or cross-process coordination are used.
func NewConcurrency(capacity int) (*Concurrency, error) {
	c := new(Concurrency)
	if err := c.SetCapacity(capacity); err != nil {
		return nil, err
	}
	return c, nil
}

// SetCapacity atomically changes the local limit without changing permits or
// counters. Nonpositive values return an error and leave the prior limit intact.
// Reductions affect new admissions only; existing invocations finish normally.
func (c *Concurrency) SetCapacity(capacity int) error {
	if capacity <= 0 {
		return errors.New("admission: concurrency capacity must be positive")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Capacity = capacity
	return nil
}

// Snapshot returns the effective capacity and accounting in one synchronized read.
// Applications may export these bounded counters and gauges to their metrics system.
func (c *Concurrency) Snapshot() ConcurrencySnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Middleware wraps a unary endpoint with immediate admission. Saturation returns
// a *loom.ServiceError named "overloaded" with Temporary set. Declare that error
// in the design and map it explicitly, for example to HTTP 503. No retry occurs.
// A context canceled at admission returns ctx.Err() instead, even when saturated.
// After admission the original context and request are passed through unchanged;
// the endpoint's result, error and panic propagate unchanged.
//
// Install through generated Endpoints.Use or a protected per-method constructor
// to retain validation and authorization before admission. This does not protect
// transport decoding or authentication. It panics for an uninitialized budget
// or nil endpoint, before requests can be served.
func (c *Concurrency) Middleware(next loom.Endpoint) loom.Endpoint {
	if next == nil || c.Snapshot().Capacity <= 0 {
		panic("admission: initialized concurrency and non-nil endpoint required")
	}
	return func(ctx context.Context, request any) (any, error) {
		if err := c.acquire(ctx); err != nil {
			return nil, err
		}
		defer c.release()
		return next(ctx, request)
	}
}

func (c *Concurrency) acquire(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		c.state.Canceled++
		return err
	}
	if c.state.InFlight >= c.state.Capacity {
		c.state.Rejected++
		return loom.TemporaryError("overloaded", "service capacity exhausted")
	}
	c.state.InFlight++
	c.state.Admitted++
	return nil
}

func (c *Concurrency) release() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.InFlight--
}
