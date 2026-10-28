# Unary concurrency admission

`github.com/CaliLuke/loom/admission` bounds active unary endpoint invocations
within one process. A shared `Concurrency` instance supplies one budget across
the endpoints using it. Separate instances and separate replicas have independent
budgets. Applications choose capacity and which methods share it.

```go
budget, err := admission.NewConcurrency(32)
if err != nil {
    return err
}
endpoints := work.NewEndpoints(service, authorizer)
endpoints.Use(budget.Middleware)
```

Use `Endpoints.Use` only when all affected methods are unary and should share
the budget. For a mixed service, pass `budget.Middleware` to the generated
protected per-method constructor for each selected unary method. Do not wrap or
replace a protected endpoint field from outside its access boundary.

Protected endpoints validate, authenticate and authorize before admission.
Access checks still run on saturated requests, and repeat before service
execution if admission succeeds. Admission preserves the context, payload,
deadline and cancellation. See [authorization ordering](authorization.md#invocation-boundary).
Unprotected methods retain their existing validation and middleware ordering;
the limiter does not add authorization or move validation to a new boundary.

The budget protects work inside the wrapped endpoint. Protect transport decoding
and authentication with separately positioned transport controls. Request-rate
and burst quotas are separate from the count of active calls; see
[rate admission](rate-limiting.md) for early and authenticated quotas.

## Saturation and completion

When all permits are occupied, middleware returns immediately without invoking
the next endpoint. The returned `*loom.ServiceError` is named `overloaded`, has
`Temporary` set, and contains no request identity. Declare and map it explicitly:

```go
Error("overloaded", func() {
    Temporary()
})
HTTP(func() {
    Response("overloaded", StatusServiceUnavailable)
})
GRPC(func() {
    Response("overloaded", CodeUnavailable)
})
```

HTTP 503 describes service saturation. Caller-quota exhaustion uses a separate
error and HTTP 429. Other transports use the mappings declared in the design;
the middleware does not select transport statuses. It adds no retry or
`Retry-After` header because it cannot predict when an active call will finish.

An already canceled or expired context is rejected with `ctx.Err()` before
capacity is considered. Once admitted, a call retains its permit until the
wrapped endpoint exits, including service errors and panic unwinding. Panics
propagate to the caller's recovery boundary. Cancellation does **not** free
capacity while an invocation continues running. A cancellation racing admission
may occur after its context check; the endpoint still receives the original
context and remains responsible for honoring it.

Unary endpoint completion is the lifetime boundary. Do not apply this helper to
streams, subscriptions, or background work that outlives endpoint return. The
helper creates no queue, background goroutine, distributed lease or adaptive
capacity policy.

## Updating and observing capacity

```go
if err := budget.SetCapacity(16); err != nil {
    return err
}
snapshot := budget.Snapshot()
```

Capacity must be positive at construction and on every update. An invalid
update preserves the prior capacity and all accounting. Updates and admissions
are synchronized, so each decision sees one effective capacity. Reapplying a
capacity does not reset counters or permits.

Reducing capacity leaves active invocations running. `InFlight` may temporarily
exceed the new capacity; no new call enters until it is below that limit.
Increasing capacity permits more admissions immediately. The framework does not
provide an operator endpoint, configuration store, authorization for updates or
coordination across replicas.

`Snapshot` returns capacity, in-flight calls and cumulative admitted, saturated
(`Rejected`) and pre-admission canceled (`Canceled`) counts in one coherent read.
Export those fields through your metrics system with application-chosen bounded
labels. Requests rejected by earlier validation or access checks are outside
these counters. An admitted call that later fails or is canceled remains counted
as admitted. There are no caller identities or metrics-backend dependencies.

## Runnable generated service

The [design](examples/concurrency/design.go.txt) and
[application](examples/concurrency/main.go.txt) demonstrate a shared local budget
of two calls with API-key authentication, a protected unary method and explicit
HTTP/gRPC overload mappings. The application mounts HTTP on loopback. This guide
tracks `main`; select a Loom release or exact commit containing `admission`.
Use Go 1.27.3 and the normal gRPC generation prerequisites described in the
[gRPC guide](grpc-guide.md).

In a new directory, save the design as `design/design.go`, then run:

```sh
go mod init example.com/admission
go get -tool github.com/CaliLuke/loom/cmd/loom@"$LOOM_VERSION"
go tool loom gen example.com/admission/design
```

Set `LOOM_VERSION` to the selected release or commit before running the commands.
Save the application as `main.go` after generation, then run:

```sh
go mod tidy
EXAMPLE_API_KEY=local-demo-key go run .
```

From another terminal, start two long calls:

```sh
curl -s -H 'X-API-Key: local-demo-key' -H 'Content-Type: application/json' \
  -d '{"seconds":30}' http://127.0.0.1:8080/work &
curl -s -H 'X-API-Key: local-demo-key' -H 'Content-Type: application/json' \
  -d '{"seconds":30}' http://127.0.0.1:8080/work &
```

Wait until the server terminal has printed `work started; holding an admission
permit` twice. Those messages confirm that both calls acquired permits. Then,
before the 30-second calls finish, send the third request. If either call has
already finished, run `wait` in the request terminal to let both calls finish,
then start the two long calls again and wait for two new messages.

```sh
curl -i -H 'X-API-Key: local-demo-key' -H 'Content-Type: application/json' \
  -d '{"seconds":1}' http://127.0.0.1:8080/work
wait
```

The saturated request receives 503 without service execution. An invalid payload
still receives 400, and an invalid key receives 401, even while capacity is full.
Use application-owned authentication and authorization policies in deployed
services; this example grants work permission to its one configured demo key.
