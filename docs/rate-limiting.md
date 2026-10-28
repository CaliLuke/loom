# Rate and burst admission

Rate admission counts attempts over time. It is separate from
[concurrency admission](admission.md), which bounds active unary calls.
Loom uses existing HTTP and protected endpoint middleware for both controls.
Applications choose budgets, trusted grouping keys, storage and quota scope.

## Local budgets and live policy

`admission.Rate` wraps `golang.org/x/time/rate` with synchronized policy updates.
One instance is one process-local token bucket. It starts full; each allowed
attempt consumes one token. It does not queue, reserve future tokens or refund
tokens after a later service failure, access recheck failure or cancellation.

```go
budget, err := admission.NewRate(admission.RatePolicy{PerSecond: 5, Burst: 10})
if err != nil {
    return err
}
decision, err := budget.Allow(ctx)
```

`Allow` reports `Allowed`, an advisory `RetryAfter`, and the policy `Version` used
for that decision. Cancellation observed before admission consumes nothing.
A concurrent cancellation can happen after that decision; the request still
retains its original context and spent quota.

Use `SetPolicy` to update rate and burst together without rebuilding middleware:

```go
if err := budget.SetPolicy(admission.RatePolicy{PerSecond: 2, Burst: 4}); err != nil {
    return err
}
policy, version := budget.Policy()
```

Both settings must be positive and the rate must be finite and below
`math.MaxFloat64`, which is reserved for unlimited admission. Invalid updates
preserve the previous settings, version and accounting. Each decision and policy
read sees a coherent version. Reapplying unchanged settings does not reset quota
or advance the version.

An update first accrues tokens under the **old** rate up to the update time, capped
by the old burst. It retains that allowance, capped by the **new** burst, then
uses the new rate for future refill. Increasing burst adds room for future refill;
it does not immediately replenish allowance. For example, an exhausted 2-token
bucket raised to a burst of 10 remains exhausted until refill. Decreasing burst
from 10 to 2 with 6 tokens available leaves 2 tokens available. This is a local
atomic update, not a fleet-wide policy activation protocol.

## Backend admission boundary

`admission.NewRateAdmission(check, timeout)` calls an application-supplied
`RateCheck` with a grouping key and a bounded child context. A check can select
one of several local `Rate` budgets or use an application-owned shared backend.
The backend owns atomic accounting and policy changes for shared quotas.

The check must be safe for concurrent use and return promptly on cancellation.
Configure its network/storage operations to honor the supplied deadline. Loom
does not launch a goroutine to abandon an uncooperative backend. The configured
timeout bounds cooperative checks; it cannot forcibly terminate arbitrary Go
code. No requests proceed after backend errors, timeout, malformed decisions or
late successful answers observed after the deadline. Caller cancellation returns
the caller's context error. Other failures become the safe named error
`rate_limit_unavailable`; original backend text is not exposed.

The child context is used only for the check. An admitted service receives the
original request context, preserving values, cancellation and its own deadline.
There is no automatic retry.

## Placement and error mappings

Use `http/middleware.RateLimit(control, key)` outside the generated HTTP handler
for an early limit protecting decoding and authentication. Quota denial returns
HTTP 429. A known delay produces a positive, rounded-up whole-second `Retry-After`;
an unknown delay omits the header. Backend or grouping failure returns HTTP 503
with no backend details.

Use `control.Middleware(key, denied)` through generated `Endpoints.Use` or a
protected unary per-method constructor for an authenticated quota. Generated
validation and access checks run before the quota, including on rejected calls,
and repeat before service execution. Grouping must read identity established by
authentication. A grouping failure or nil denial error fails closed. The denial
callback returns the application's declared error, allowing a typed response
with the known delay mapped to a `Retry-After` header. Declare
`rate_limit_unavailable` too and map it explicitly, such as HTTP 503. Other
transports retain their authored error mappings.

An early quota may be spent on malformed or unauthorized requests because it
protects that work. An authenticated quota is not spent when earlier protected
validation/access fails. A later access recheck failure does not refund an
already consumed attempt. Place observation at these boundaries if you need
separate counters; use bounded labels, never credentials or user identifiers.

## Grouping and storage ownership

Local budgets are independent across replicas. Applications that need a shared
quota must supply a shared backend; a local version does not imply fleet-wide
consistency. The framework provides no tenant policy store, operator endpoint,
audit history, distribution protocol or streaming/message quota.

`RateAdmission` stores no grouping keys. The runnable example uses an immutable
map of two known groups, so key storage is bounded. Production applications must
bound their own group registry or backend. Unknown groups should be rejected;
do not turn backend failures or capacity pressure into permission to proceed.
Evicting a bucket and recreating it full can reset spent quota, so an eviction
policy must preserve accounting. Never create unlimited buckets from arbitrary
request input.

For an early network-based key, `RemoteAddr` identifies the connected peer. Behind
a proxy that may be the proxy itself. Only use forwarded IP headers under an
explicit trusted proxy policy that strips untrusted input; Loom does not infer
that trust. Authenticated quotas should use a trusted stable group from context,
not an arbitrary tenant header, raw credential or request body field.

## Runnable generated service

Save the [design](examples/rate/design.go.txt) as `design/design.go` in a new
directory. With Go 1.27.3 and `LOOM_VERSION` set to a release or exact commit
containing rate admission:

```sh
go mod init example.com/admission
go get -tool github.com/CaliLuke/loom/cmd/loom@"$LOOM_VERSION"
go tool loom gen example.com/admission/design
```

Save the [application](examples/rate/main.go.txt) as `main.go` after generation:

```sh
go mod tidy
EXAMPLE_API_KEY=local-demo-key go run .
```

The loopback HTTP server has a shared early budget of 10 requests/second with a
burst of 20. Its two authenticated groups each have 1 request/second and a burst
of 2. Send three quick requests using `local-demo-key-alpha` to observe a 429;
`local-demo-key-beta` has separate allowance:

```sh
for attempt in 1 2 3; do
  curl -i -H 'X-API-Key: local-demo-key-alpha' -H 'Content-Type: application/json' \
    -d '{"message":"hello"}' http://127.0.0.1:8080/work
done
```

The demo grants work permission to either configured key. Replace that demo
policy with application authentication and authorization. Policy persistence,
operator authorization and activation across replicas remain application-owned.
