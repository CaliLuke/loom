package admission

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type (
	// KeyedRate keeps a separate process-local token bucket for each key, all
	// sharing one RatePolicy, and stores at most a fixed number of keys.
	// Construct it with NewKeyedRate; it must not be copied. Its Allow method is
	// a RateCheck, so NewRateAdmission(keyed.Allow, timeout) admits requests by
	// key. Applications still own trusted grouping: keys should be short, stable
	// identifiers from a source that is costly to rotate, such as an
	// authenticated tenant or a trusted peer address.
	//
	// A bucket is evicted only when it has refilled to the full burst. A new
	// bucket for the same key also starts full, so eviction never resets spent
	// quota. When every stored bucket is partly spent and a new key arrives, the
	// new key is denied with a RetryAfter that ends when the first stored bucket
	// becomes full and can be evicted. Capacity pressure never admits a request.
	// A spent bucket holds its slot for up to Burst/PerSecond after its last
	// request, so size maxKeys for the peak number of distinct keys active within
	// that time. Buckets are ordered by the time they become full, so Allow does
	// O(log maxKeys) work even while keys rotate.
	KeyedRate struct {
		mu      sync.Mutex
		buckets map[string]*keyedBucket
		byFull  bucketQueue
		maxKeys int
		policy  RatePolicy
		version uint64
		// inspected counts buckets examined for eviction. Tests use it to
		// check that eviction work stays bounded.
		inspected uint64
	}

	keyedBucket struct {
		key     string
		limiter *rate.Limiter
		// fullAt is when the bucket's allowance reaches the burst, rounded up
		// to the next nanosecond.
		fullAt time.Time
		// index is the bucket's position in bucketQueue.
		index int
	}

	// bucketQueue is a min-heap of buckets ordered by fullAt.
	bucketQueue []*keyedBucket
)

// NewKeyedRate validates policy and a positive maxKeys and constructs an empty
// keyed budget at version 1. Each key's bucket starts full on first use.
func NewKeyedRate(policy RatePolicy, maxKeys int) (*KeyedRate, error) {
	if err := validateRatePolicy(policy); err != nil {
		return nil, err
	}
	if maxKeys <= 0 {
		return nil, errors.New("admission: keyed rate requires a positive key limit")
	}
	return &KeyedRate{
		buckets: make(map[string]*keyedBucket),
		maxKeys: maxKeys,
		policy:  policy,
		version: 1,
	}, nil
}

// Allow consumes one token from key's bucket if available, with the same
// semantics as Rate.Allow. A canceled context consumes nothing and stores no
// key. All keys share one mutex.
func (k *KeyedRate) Allow(ctx context.Context, key string) (RateDecision, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := rateContextError(ctx); err != nil {
		return RateDecision{}, err
	}
	if k.buckets == nil {
		return RateDecision{}, errors.New("admission: keyed rate budget must be constructed with NewKeyedRate")
	}
	now := time.Now()
	bucket, stored := k.buckets[key]
	if !stored {
		if len(k.buckets) >= k.maxKeys {
			k.evictFull(now)
			if len(k.buckets) >= k.maxKeys {
				return RateDecision{RetryAfter: k.byFull[0].fullAt.Sub(now), Version: k.version}, nil
			}
		}
		bucket = &keyedBucket{key: key, limiter: rate.NewLimiter(rate.Limit(k.policy.PerSecond), k.policy.Burst)}
	}
	decision := RateDecision{Allowed: bucket.limiter.AllowN(now, 1), Version: k.version}
	tokens := bucket.limiter.TokensAt(now)
	if !decision.Allowed {
		decision.RetryAfter = refillDelay(1-tokens, k.policy.PerSecond)
	}
	bucket.fullAt = k.fullAt(now, tokens)
	if stored {
		heap.Fix(&k.byFull, bucket.index)
	} else {
		k.buckets[key] = bucket
		heap.Push(&k.byFull, bucket)
	}
	return decision, nil
}

// SetPolicy atomically applies both settings to every stored bucket and to
// buckets created later, with the same accounting rules as Rate.SetPolicy.
// Reapplying identical settings does not reset accounting or advance the
// version. Invalid updates leave all state unchanged.
func (k *KeyedRate) SetPolicy(policy RatePolicy) error {
	if err := validateRatePolicy(policy); err != nil {
		return err
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.buckets == nil {
		return errors.New("admission: keyed rate budget must be constructed with NewKeyedRate")
	}
	if k.policy == policy {
		return nil
	}
	now := time.Now()
	k.policy = policy
	k.version++
	for _, bucket := range k.byFull {
		bucket.limiter.SetLimitAt(now, rate.Limit(policy.PerSecond))
		bucket.limiter.SetBurstAt(now, policy.Burst)
		bucket.fullAt = k.fullAt(now, bucket.limiter.TokensAt(now))
	}
	// One shared policy maps every bucket's allowance through the same
	// monotone clamp, so the order already holds; reinitializing keeps the
	// heap valid without relying on that argument.
	heap.Init(&k.byFull)
	return nil
}

// Policy returns both effective settings and their version in one coherent read.
func (k *KeyedRate) Policy() (RatePolicy, uint64) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.policy, k.version
}

// Len returns the number of stored keys, which never exceeds the key limit.
func (k *KeyedRate) Len() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.buckets)
}

// Len implements heap.Interface.
func (q bucketQueue) Len() int {
	return len(q)
}

// Less implements heap.Interface.
func (q bucketQueue) Less(i, j int) bool {
	return q[i].fullAt.Before(q[j].fullAt)
}

// Swap implements heap.Interface.
func (q bucketQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

// Push implements heap.Interface.
func (q *bucketQueue) Push(x any) {
	bucket, ok := x.(*keyedBucket)
	if !ok {
		panic("admission: bucket queue accepts only *keyedBucket")
	}
	bucket.index = len(*q)
	*q = append(*q, bucket)
}

// Pop implements heap.Interface.
func (q *bucketQueue) Pop() any {
	old := *q
	last := len(old) - 1
	bucket := old[last]
	old[last] = nil
	*q = old[:last]
	return bucket
}

// fullAt returns when a bucket holding tokens reaches the burst.
func (k *KeyedRate) fullAt(now time.Time, tokens float64) time.Time {
	missing := float64(k.policy.Burst) - tokens
	if missing <= 0 {
		return now
	}
	return now.Add(refillDelay(missing, k.policy.PerSecond))
}

// evictFull removes every bucket that has refilled to the full burst. It
// examines only buckets due by now, so its work is proportional to the number
// it evicts. A due bucket that rounding left just short of full is rescheduled.
func (k *KeyedRate) evictFull(now time.Time) {
	for len(k.byFull) > 0 && !k.byFull[0].fullAt.After(now) {
		k.inspected++
		bucket := k.byFull[0]
		tokens := bucket.limiter.TokensAt(now)
		if float64(k.policy.Burst)-tokens > 0 {
			bucket.fullAt = k.fullAt(now, tokens)
			heap.Fix(&k.byFull, 0)
			continue
		}
		heap.Pop(&k.byFull)
		delete(k.buckets, bucket.key)
	}
}
