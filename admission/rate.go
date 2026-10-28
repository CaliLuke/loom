package admission

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type (
	// RatePolicy configures one local token bucket. Both fields must be positive,
	// and PerSecond must be below math.MaxFloat64 (the dependency's unlimited
	// sentinel). One admitted request consumes one token.
	RatePolicy struct {
		// PerSecond is the number of tokens replenished per second.
		PerSecond float64
		// Burst is the maximum token allowance, initially available in full.
		Burst int
	}

	// RateDecision reports one admission decision. A denied decision with no error
	// means quota exhaustion; backend failures are returned separately as errors.
	RateDecision struct {
		// Allowed permits the request to proceed.
		Allowed bool
		// RetryAfter is a positive advisory delay until allowance may be available.
		// Zero means unknown. A successful decision must leave this field zero.
		RetryAfter time.Duration
		// Version identifies the coherent local policy used for this decision.
		// External checkers may use zero when they do not expose a version.
		Version uint64
	}

	// Rate wraps a maintained token bucket with atomic policy updates. Construct
	// it with NewRate; it must not be copied. A Rate is one process-local budget,
	// not a keyed store. Applications own trusted grouping and bounded storage of
	// separate budgets, or supply a shared backend to NewRateAdmission.
	Rate struct {
		mu      sync.Mutex
		limiter *rate.Limiter
		policy  RatePolicy
		version uint64
	}
)

// NewRate validates policy and constructs a full local bucket at version 1.
func NewRate(policy RatePolicy) (*Rate, error) {
	if err := validateRatePolicy(policy); err != nil {
		return nil, err
	}
	return &Rate{
		limiter: rate.NewLimiter(rate.Limit(policy.PerSecond), policy.Burst),
		policy:  policy,
		version: 1,
	}, nil
}

// SetPolicy atomically applies both settings. Accrue allowance under the old
// rate up to the update, retain that allowance (clamped to the new burst), then
// use the new rate for future refill. Increasing burst adds refill headroom,
// not immediate allowance. Reapplying identical settings does not reset
// accounting or advance the version. Invalid updates leave all state unchanged.
func (r *Rate) SetPolicy(policy RatePolicy) error {
	if err := validateRatePolicy(policy); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.limiter == nil {
		return errors.New("admission: rate budget must be constructed with NewRate")
	}
	if r.policy == policy {
		return nil
	}
	now := time.Now()
	r.limiter.SetLimitAt(now, rate.Limit(policy.PerSecond))
	r.limiter.SetBurstAt(now, policy.Burst)
	r.policy = policy
	r.version++
	return nil
}

// Policy returns both effective settings and their version in one coherent read.
func (r *Rate) Policy() (RatePolicy, uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.policy, r.version
}

// Allow consumes one token if available, without waiting or reserving future
// allowance. A canceled context consumes nothing. Consumed tokens are not
// refunded if later validation, service execution or request cancellation fails.
// RetryAfter is advisory: other calls and policy changes can alter availability.
func (r *Rate) Allow(ctx context.Context) (RateDecision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := rateContextError(ctx); err != nil {
		return RateDecision{}, err
	}
	if r.limiter == nil {
		return RateDecision{}, errors.New("admission: rate budget must be constructed with NewRate")
	}
	now := time.Now()
	decision := RateDecision{Allowed: r.limiter.AllowN(now, 1), Version: r.version}
	if !decision.Allowed {
		missing := 1 - r.limiter.TokensAt(now)
		nanos := math.Ceil(missing / r.policy.PerSecond * float64(time.Second))
		switch {
		case nanos >= float64(math.MaxInt64):
			decision.RetryAfter = time.Duration(math.MaxInt64)
		case nanos < 1:
			decision.RetryAfter = time.Nanosecond
		default:
			decision.RetryAfter = time.Duration(nanos)
		}
	}
	return decision, nil
}

// RetryAfterSeconds rounds a known positive delay up to whole seconds without
// overflowing. It returns zero for an unknown delay. HTTP adapters can publish
// a positive value as Retry-After; unknown delays must omit that header.
func (d RateDecision) RetryAfterSeconds() int64 {
	if d.RetryAfter <= 0 {
		return 0
	}
	seconds := int64(d.RetryAfter / time.Second)
	if d.RetryAfter%time.Second != 0 {
		seconds++
	}
	return seconds
}

func validateRatePolicy(policy RatePolicy) error {
	if policy.PerSecond <= 0 || math.IsNaN(policy.PerSecond) || policy.PerSecond >= float64(rate.Inf) || policy.Burst <= 0 {
		return errors.New("admission: rate must be positive and below math.MaxFloat64 and burst must be positive")
	}
	return nil
}
