# Consumer constructor numeric admission (#772)

Status: implementation and affected checks complete against `5eea445b9`; exact final review pending.
Original admission evidence uses exact archived `92e232503`; relevant owners are unchanged.

- [x] Read the original zero-grace report and current option consumers.
- [x] Reproduce invalid options reaching Redis before admission; preserve controls.
- [x] Independent readiness review of the bounded domain/owner.
- [x] Implement constructor admission, then run direct and affected runtime gates.
- [ ] Independent exact-final review, atomic delivery and cleanup.

## Contract and owner

Parsed options must be validated before constructing a live consumer or touching
Redis. Two demonstrated runtime domains belong here: Reader/Sink subscriber
buffer sizes must be nonnegative (`make(chan, n)` panics for negative n); Sink
acknowledgement grace must be positive (`time.NewTicker` requires it, and zero or
negative grace also invalidates stale-consumer and claim timing). Rejection must
return a descriptive error from NewReader/NewSink, before commands, membership,
background workers or subscriber creation. Options are checked after all setters,
so the final supplied configuration is authoritative.

Keep this small: share one private buffer-domain check beside the constructors,
used by both newReader and newSink; validate the sink-only grace domain once at
newSink's same pre-publication boundary. Do not change Parse*Options signatures,
add a generic configuration framework, or scatter checks into Subscribe, ticker,
rotation or claim loops. Those consumers may rely on admitted values. Existing
regex admission still owns the effective filter; filter membership is #768.

## Preserved behavior and limits

Buffer zero is a valid unbuffered channel and existing Reader tests use it.
BlockDuration zero is documented indefinite blocking and used by pool tests.
Any positive grace remains in this correction's admitted domain; no arbitrary
minimum, maximum, millisecond rounding or disabling mode is introduced.

Count and blocking options require different evidence: pinned go-redis omits
COUNT for nonpositive MaxPolled, and omits BLOCK for negative BlockDuration.
The reported channel/ticker failure does not establish a new bound for either.
Leave those behaviors unchanged in this ticket rather than silently define a new
policy. A focused native-owner probe also confirmed doubled-grace overflow at
`time.Duration(math.MaxInt64)`: both rotation and cleanup treat a fresh consumer
as stale (parent race0.680s). Both call sites derive from one private
`consumerKeepAliveExpired(elapsed, grace)` comparison: elapsed must exceed grace
and its remaining difference must exceed grace. Positive grace makes the
subtraction safe; strict equality remains live, negative elapsed remains live,
and no overflowing multiplication or arbitrary upper bound is needed. This is
not an exhaustive numeric configuration audit or a Sink lifetime/claim protocol
change.

## Evidence and bounded acceptance

`pulse/streaming/consumer_options_test.go` uses public constructors with a Redis command hook
that fails the first attempted command. The exact parent fails four expectations:
Sink grace zero/negative, Sink buffer negative, Reader buffer negative. All reach
native setup instead of reporting their invalid option (0.833s under race).
The hook makes this deterministic without starting the known crashing ticker or
spawning live consumers. It also proves no native side effect when corrected.
Four controls keep default/unbuffered buffers, documented block zero, and a
positive grace accepted up to the same native setup boundary.

Six final-option-wins cases and two invalid effective-regex cases pass. Eleven
arithmetic boundaries cover strict equality, future timestamps, ordinary expiry,
and durations near the maximum. Two native-owner controls retain a fresh
consumer through both rotation and cleanup with maximum positive grace. The full
streaming/options race suite passes (6.973s/1.381s), including existing successful
constructor/subscription/keepalive/rotation/pending-entry controls. Scoped lint
reports zero issues. No new Redis harness, model, generator or
full Pulse suite is needed: the change is stateless admission before existing
ownership/lifetime protocols. Existing #768/#800 witnesses remain unchanged.


## Review packet and limits

`/tmp/loom-772-final-manifest.json` freezes exact source, original/candidate
receipts, inputs and owner hashes. No Redis command shape, Lua script, filter
membership protocol, generator or checked-in generated artifact changes. The
native-owner regressions run against the existing miniredis tier; the arithmetic
is independent of Redis behavior. No new real-Redis or all-Pulse sweep is needed.
The constructors enforce the positive-grace precondition for the shared expiry
predicate; manually assembled private Sink test fixtures must retain it.
