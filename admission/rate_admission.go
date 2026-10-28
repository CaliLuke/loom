package admission

import (
	"context"
	"errors"
	"time"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// RateCheck checks and accounts for one request in an application-selected
	// group. It must be safe for concurrent use and return promptly when ctx ends.
	// A backend owns coherent policy/accounting and the scope of its quota. Keys
	// must not be included in returned error messages, logs or metric labels.
	RateCheck func(ctx context.Context, key string) (RateDecision, error)

	// RateAdmission bounds calls to a quota backend and rejects its errors or
	// invalid decisions. Construct it with NewRateAdmission. It creates no
	// background goroutines, retries, queues or key storage. Its timeout bounds
	// cooperative backends; it cannot forcibly stop a callback that ignores ctx.
	RateAdmission struct {
		check   RateCheck
		timeout time.Duration
	}
)

// NewRateAdmission validates the checker and its positive per-check time budget.
// Applications must configure network/storage timeouts in their checker too.
func NewRateAdmission(check RateCheck, timeout time.Duration) (*RateAdmission, error) {
	if check == nil || timeout <= 0 {
		return nil, errors.New("admission: rate checker and positive timeout required")
	}
	return &RateAdmission{check: check, timeout: timeout}, nil
}

// Check runs one backend check with a child deadline, retaining the caller's
// values and earlier deadline. Caller cancellation returns ctx.Err(). Backend
// errors, backend timeouts, malformed decisions and an uninitialized admission
// return a safe *loom.ServiceError named "rate_limit_unavailable". Backend error
// text and grouping keys are never included. No error permits execution.
func (a *RateAdmission) Check(ctx context.Context, key string) (RateDecision, error) {
	if err := rateContextError(ctx); err != nil {
		return RateDecision{}, err
	}
	if a == nil || a.check == nil || a.timeout <= 0 {
		return RateDecision{}, rateUnavailable()
	}
	checkCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	decision, err := a.check(checkCtx, key)
	if callerErr := rateContextError(ctx); callerErr != nil {
		return RateDecision{}, callerErr
	}
	if err != nil || rateContextError(checkCtx) != nil || decision.RetryAfter < 0 || (decision.Allowed && decision.RetryAfter != 0) {
		return RateDecision{}, rateUnavailable()
	}
	return decision, nil
}

// Middleware admits unary endpoint calls using an application-owned grouping
// function and declared denial error. Install it through generated Endpoints.Use
// or protected constructors so grouping uses authenticated identity. A key error
// becomes "rate_limit_unavailable" without exposing its text. Denial invokes
// denied with the decision; it must return an error matching the design, including
// any known RetryAfter mapping. A nil denial error fails closed.
//
// The next endpoint receives the original context, not the bounded backend
// context. Callback and next panics propagate. Invalid construction dependencies
// panic before serving. Streams and per-message quotas are outside this helper.
func (a *RateAdmission) Middleware(key func(context.Context, any) (string, error), denied func(RateDecision) error) func(loom.Endpoint) loom.Endpoint {
	if a == nil || a.check == nil || a.timeout <= 0 || key == nil || denied == nil {
		panic("admission: initialized rate admission, grouping and denial functions required")
	}
	return func(next loom.Endpoint) loom.Endpoint {
		if next == nil {
			panic("admission: non-nil endpoint required")
		}
		return func(ctx context.Context, request any) (any, error) {
			if err := rateContextError(ctx); err != nil {
				return nil, err
			}
			group, err := key(ctx, request)
			if err != nil {
				return nil, rateUnavailable()
			}
			decision, err := a.Check(ctx, group)
			if err != nil {
				return nil, err
			}
			if !decision.Allowed {
				if err := denied(decision); err != nil {
					return nil, err
				}
				return nil, rateUnavailable()
			}
			return next(ctx, request)
		}
	}
}

// rateContextError checks clock expiry independently of timer delivery. A busy
// backend can return after its deadline before ctx.Err publishes that expiry.
func rateContextError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

func rateUnavailable() error {
	return loom.TemporaryError("rate_limit_unavailable", "rate limit check unavailable")
}
