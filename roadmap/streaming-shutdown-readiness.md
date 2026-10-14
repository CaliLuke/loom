# Bounded streaming operation and Close readiness — #800

Status: **reviewed implementation being integrated with the published exact
hook-preserving module fork**. Original-module-only waiting is superseded by
the reproducible dependency decision in
[the dependency record](../pulse/streaming/tla/read_shutdown/dependency_fetchability.md).
Historical implementation parent: `ebc06ec94`. Current refresh parent: `7da1b00ee`
(includes #765 registration recovery, #901 activation, #768 filter membership,
and #771/#772/#773 notification, option and Reader lookup corrections).
Historical failure probes ran at `81192e7a93cfafee03e081a4be45184b3458b831`.

## Sequence and evidence

- [x] Preserve old #687/#704 worktree; copy only durable #800 investigation inputs.
- [x] Reproduce all eight Reader/Sink blocking reads on this parent using actual
  Redis 8.10.2 CLIENT LIST admission evidence, plus eight concurrent Close calls
  which incorrectly return before the first joins. Releasing one event joins all
  goroutines and the caller client remains usable.
- [x] Reproduce actual NewSink rollback after maps/membership acquisition: fail
  group creation while Redis is frozen, cancel setup, observe cleanup still
  blocked; unfreeze, join and retain the original error. The 200ms observation
  interval diagnoses the parent, not a proposed shutdown deadline.
- [x] Establish a conditional native-command operation mechanism: bounded pool,
  retries, dial/handshake and actual TCP writes; native positive BLOCK sets its
  socket read deadline. Preserve hooks, caller options and pool ownership.
- [x] Discover and isolate go-redis v9.23.0 WithTimeout dropping inherited hooks.
  Standalone regression fails; minimal local hooksMixin clone proposal passes.
  Upstream publication is complete; the separately reviewed root-module fork
  makes the correction fetchable without downstream replacements.
- [x] Model current unbounded operations, incomplete read-only correction, early
  second Close and shared bounded operation/admission/join ownership.
- [x] Independently review readiness, exact upstream patch and outgoing draft;
  resolve the timeout-wording finding. User authorized publication: issue4094,
  PR4095 now retain the exact reviewed stable-plus-fix correction.
- [x] Publish the independently reviewed stable-plus-fix module fork and record
  its exact pin, application type migration and return-to-upstream condition.
- [x] Implement the shared operation and lifecycle owner, then migrate
  Reader/Sink/rmap and delete superseded stop/close state. Use the exact local
  reviewed dependency source; verify the published pin before delivery.
- [x] Run direct red-to-green admission, cleanup and concurrent Close checks,
  align the model, then run the bounded acceptance matrix.
- [ ] Obtain independent review of the exact integration diff and verify ordinary
  downstream module consumption before delivery.

## Owner and public behavior

Introduce one internal Pulse operation policy shared by streaming and rmap. It
borrows a timeout facade for each logical operation from the caller's client;
never closes that facade or changes caller Options. With the proposed driver
repair, borrowing snapshots installed hooks, and future operations see hooks
added since earlier operations. Native XREAD/XREADGROUP remain native, preserving
ProcessHook dispatch, key metadata and group-lag instrumentation.

Derive the shared operation ceiling from the existing five-second framework
poll/default cleanup budget. Use stricter positive caller read/write settings
when choosing the facade's ordinary socket timeout and operation context budget;
disabled/infinite settings do not disable owned-operation bounds. Preserve the
caller options unchanged. This does not promise that every command completes
within the caller's ReadTimeout: native XREAD/XREADGROUP replace that socket-read
limit with BLOCK + 10 seconds when ContextTimeoutEnabled is false.
Attach a deadline to the borrowed operation, covering pool acquisition, retry
sleep and cooperative dial/handshake. Finite facade options bound ordinary socket
reads and writes; native blocking reads use the separate driver deadline below.
Cleanup detached with WithoutCancel receives a fresh operation budget. Keep unrelated options, hooks,
authentication, TLS, retry policy and caller pools intact.

Cap internal blocking polls at that ceiling. A configured zero or longer block
still means continuous listening, implemented by successive finite native polls.
Preserve negative/nonblocking options. The driver adds ten seconds to native
positive BLOCK for its socket deadline; an active read need not stop at context
cancellation or at the shorter operation deadline. No new subsecond or fixed
five-second Close SLA is promised. Retries must not restart after the operation
deadline, and finite work already admitted must finish or time out before Close
returns. Multiple cleanup commands can take multiple finite budgets.

Use the shared atomic stopping state and once-only cancellation, separate from locks held during Redis I/O,
to seal admission and cancel the runtime before joining. Keep internal stop
signaling separate from external Close: an internal read-loop error must not wait
for itself. Every external Close waits for the same completion signal, which is
published only after workers, admitted resource acquisitions and owned map
cleanup finish. Do not introduce an asynchronous abandoned-cleanup path.

Sink AddStream acquires replicated maps, so its acquisition must be admitted and
joined or rolled back before completion. Reader AddStream's unpublished newest
lookup owns no resource and retains #761's established Close-during-lookup
contract: Close does not wait for that lookup; publication rechecks the owner.
Use the existing #802 registration/cursor owner for reads and publication. Do not
create a second registration registry.

rmap owns its PubSub handles. Bound initial/reconnect subscription setup with the
same operation policy and explicit ReceiveTimeout; close an owned PubSub directly
on shutdown instead of an unnecessary unsubscribe network round trip before
Close. Preserve error reporting and waiter/subscriber completion. This is local
handle ownership, not a guarantee of joining private caller-driver goroutines.

## Required consumers and limits

| Owner | Owned operation paths |
| --- | --- |
| Reader | Newest snapshots, admitted XREAD and background retries |
| Sink construction and AddStream | Script loads, map joins, membership, groups, consumers, failure cleanup |
| Sink runtime | XREADGROUP, keepalive, lease, stale cleanup, claims, malformed-entry acknowledgement, consumer rotation and deregistration |
| rmap | Script loads/execution, initial content, subscription initialization/reconnect and owned handle Close |
| Close | Stop admission before I/O locks, join every caller, preserve pending-message recovery metadata |

Stream producer Add/Destroy and external Event.Ack are caller operations, not
background-owned shutdown work; do not silently impose a new timeout there.
rmap Reset/Destroy's documented post-Close behavior remains supported. Arbitrary
caller hooks, loggers or dialers which never return are outside the guarantee;
context cancellation cannot forcibly terminate them while preserving behavior.
The operation model assumes finite admitted work and cooperative scheduling; it
does not prove wall-clock limits, Redis delivery guarantees or custom callbacks.

#683 Subscribe-after-Close was delivered independently at `ebc06ec94`; retain
its shared admission owner and tests. #773 equivalent-handle removal remains intact. #704 worker admission/publication
is already delivered and starts only admitted Readers. #687 node-construction
rollback remains a separate repair; this patch supplies bounded subordinate
cleanup without changing pool publication.

## Bounded implementation acceptance

1. Promote current read/concurrent-close and constructor-cleanup regressions.
   Test Reader, Sink and Map concurrent joins plus internal stop without self-wait.
2. Exercise maintenance holding the Sink state mutex and an AddStream acquisition
   racing Close; stop admission first and join or roll back owned maps. Preserve
   Reader pending-lookup behavior and #802 stale-registration regressions.
3. Repeat finite native read, saturated pool, retry, handshake and actual blocked
   socket-write probes with the adopted dependency. Add already-established
   PubSub Receive/Close/reconnect socket controls; current subscription evidence
   reaches dial/handshake failure, not every PubSub phase.
4. Verify normal and empty polls, explicit/newest/multistream registrations,
   event-between-polls behavior, negative/nonblocking options, filters and
   subscriber unsubscribe. For groups, cover ack/no-ack and lost replies: no new
   exactly-once guarantee; preserve pending-entry recovery and lag instrumentation.
5. Verify caller options/client usability, inherited and late hooks, stricter
   ordinary socket/context budgets and disabled limits. Explicitly assert the
   native blocking-read override; do not interpret unchanged caller ReadTimeout
   as its effective deadline. Verify errors.Is identities and joined cleanup
   errors. No cloned-client Close or mutable Options workaround.
6. Run focused streaming/rmap tests and race tests with miniredis and isolated
   Redis 8.10.2, plus affected pool constructor/ownership controls. Recheck models
   against final implementation seams. No generator or exhaustive corpus gates.

Root owns integration, shared guides/master trackers and commits. Durable probe
sources, model assumptions, results and driver proposal are in
[pulse/streaming/tla/read_shutdown](../pulse/streaming/tla/read_shutdown/README.md).

## Historical implementation evidence (conditional on the reviewed local dependency)

- The common lifecycle owns atomic stopping, cancellation once and synchronous
  cleanup once. Reader internal fatal-stop never joins itself. Sink cancellation
  precedes its I/O mutex; that mutex joins an admitted AddStream acquisition.
  Map Close uses the same join owner and synchronizes subscriber cleanup with
  admission that began before stopping. External map writes are bounded commands,
  not new runtime workers; Reset/Destroy after Close remain permitted.
- Concurrent Reader/Sink and Map Close regressions failed on the parent, then
  pass under the shared owner. Maintenance lock cancellation, successful and
  failed AddStream acquisition, concurrent map subscriber admission and reconnect
  snapshot cleanup have explicit synchronized tests.
- Streaming/rmap package race gates pass; pool shutdown, ownership and event-loop
  controls pass. Eight real native read configurations pass with disabled caller
  socket options. Frozen socket probes call the actual redisop owner; constructor
  rollback and established/reconnecting PubSub ownership now have actual socket
  coverage. See the owning model README for exact source, commands and receipts.
- The model now explicitly includes replica receive/reconnect/cleanup, plus the
  reader, lock-held maintenance and acquisition owners. Candidate passes 44,160
  distinct states; negative controls still fail as intended. No fixed Close SLA,
  arbitrary callback termination, exactly-once delivery or private driver-worker
  joining guarantee is added.
- Final exact-diff review and a reproducibly fetchable dependency remain open.
  This work must not be called delivered or #800 closed while either is missing.

## Review correction: healthy empty native polls

Independent review reproduced a healthy empty poll returning a socket timeout
when ContextTimeoutEnabled is true: the five-second operation context expired
before Redis's five-second nil reply. The shared native-read plan must include
the capped poll duration plus the existing ordinary I/O budget as response
headroom, then retain any stricter positive caller read/write ceiling. Both
Reader and Sink consume that plan. Ordinary operations and facade socket options
retain their existing budget. This remains finite; no caller deadline is extended.
Add red-first Reader/Sink empty-poll tests for zero and long authored blocks,
then repeat affected empty-poll and finite-native shutdown checks. The model's
eventual-return assumption is unchanged, but successful empty polls must be
checked as protocol outcomes outside the liveness abstraction.

Correction evidence: Reader/Sink real Redis red10.934s; healthy-empty plus all
native shutdown controls green72.612s; stricter caller context/client controls
green1.800s. EmptyPoll model reproduces current protocol failure (exit12), then
passes response-headroom and stricter-caller controls (8/4states). Other reviewed
operation, rollback, PubSub and pool evidence is reused where unchanged.

# Conditional #800 current-owner refresh

1. Verify and archive the exact historical 53-source packet and its receipts; preserve original repro/model evidence. Confirm no unrelated work or active consumers before inverse/fast-forward.
2. Remove only that verified patch, fast-forward to delivered #768 main, then compose the old bounded-operation/lifetime owner with current registration, activation, typed membership, notification and numeric-admission owners. Dependency files remain unchanged.
3. Preserve first-subscription activation under the lifecycle mutex. Route newly introduced recovery/membership/ACK commands through redisop; stopping must seal admission before acquiring I/O-held locks. Close releases typed membership replicas without changing filter identity or PEL disposition.
4. Update active-read probes to Subscribe explicitly. Add bounded blocked-recovery/membership-versus-Close controls and retain lazy activation, concurrent Close, equivalent-reader removal, strongest notification, numeric-domain and membership recovery safeguards.
5. Run scoped native/runtime races with a disposable modfile pointing only at exact reviewed stable-plus-fix go-redis source; rerun affected existing models. Record limitations, no fixed Close SLA or arbitrary callback termination.
6. Freeze the exact refreshed diff for a non-implementer review. Original-module fetchability remains an independent delivery gate; no production dependency edits or commit are authorized by conditional green tests.

## Current-owner refresh correspondence

The refresh retains one runtime lifetime and one operation policy. Sink's first
successful Subscribe still atomically starts read/claim workers under the existing
lock; Close seals that admission before waiting for the lock. Recovery, rotation
and keepalive still use current registration identity and authored group selectors.
The typed filter/membership record stays persistent and authoritative; Close keeps
PEL recovery metadata. Newly introduced membership reads, native inventory,
group-consumer creation and rejected-event ACK borrow the same bounded client.

`changeSinkMembership` additionally owns one context across its CAS retry loop.
Per-command timeouts alone restart after every conflicting revision and cannot
bound the logical operation, including detached rollback. This is not a new
membership policy: timeout still preserves uncertain publication and original
error identities; no forced disposal or independent membership registry is added.
The command-only overlay reproduces a missing logical deadline; the extended
OperationShutdown model reproduces retry nontermination even with bounded commands.
Existing caller deadlines/positive limits remain stricter than the ordinary budget.

Current focused runtime controls cover blocked recovery PTTL and CREATECONSUMER
under the state lock, cancellation before that lock, joined cleanup, whole CAS
budget, first activation, filter admission/disposition/recovery, equivalent Reader
removal and notification precedence. Historical native failure inputs remain
unchanged; active candidate Sink tests explicitly Subscribe before observing reads.
No producer, scheduler or public ACK operation is moved into this lifetime.

### Refreshed conditional acceptance

- [x] Archive the exact historical 53-file candidate and safely fast-forward the
  managed worktree to `7da1b00ee`; preserve all newer ownership contracts.
- [x] Add current recovery/Close and logical CAS deadline controls. The latter
  also forces two successful conflicting CAS replies and checks one unchanged
  deadline across all three attempts; the command-only overlay fails both cases.
- [x] Extend the existing operation model with independent `BoundRetry`.
  Command-only/current/read-only fail liveness, early-Close fails ownership, owned
  passes44,160 states. Existing empty-poll negative/control models retain results.
- [x] Native read, empty-poll, Close and lost-reply checks pass63.444s. Native
  socket/constructor phases pass streaming28.008s/rmap7.279s. Current membership
  and pool admission/shutdown composition pass streaming1.860s/pool9.952s.
- [x] Focused current consumers pass streaming5.038s/rmap2.187s/redisop1.242s.
  Final membership/control race after extracting the unchanged unknown-group
  identity check passes5.947s. Scoped pinned lint reports0 issues.
- [ ] Fresh independent review of this exact refreshed conditional source.
- [ ] Fetchable approved original-module dependency and separate delivery review.

The linter's x/tools standard-library release-tag probe explicitly disables
modules, so `GOFLAGS=-modfile` is invalid for that one probe. A disposable traced
Go launcher removes that flag only for the exact `GO111MODULE=off` `unsafe`
release-tag query. Every real package load retains the approved modfile. No
repository dependency, toolchain, shared cache or caller configuration was changed.

## Published-module integration acceptance

The exact fork pin resolves and downloads through ordinary Go module resolution.
A separate consumer with a fresh module cache and no replace directives passes
the inherited/independent hook race regression. The actual Loom-MCP registry and
Pulse client packages compile with their Redis imports migrated consistently;
that check uses the unpublished local framework source, and its final adoption
requires a pushed containing framework version. The original consumer checkout
is untouched.

The affected Pulse race run passes pool (108.817s), rmap (2.277s), internal
packages and options. Its only failure was the new healthy native poll probe
constructing a fresh client without miniredis's Lua shim. That socket-timing
probe now explicitly requires actual Redis, consistently with the existing
active-read probe. The entire streaming race package then passes (9.923s).
Actual Redis checks pass all four healthy empty-poll cases and both stricter
deadline cases (21.762s), plus active zero-block Reader/Sink shutdown (11.684s),
with no native skips. Focused Pulse lint reports zero issues.

The shared owner and model source are otherwise identical to the previously
independently approved implementation, after consistent Redis import-path
migration. The existing model results remain applicable: the module rename and
explicit native-test requirement do not change modeled state transitions.
Final integration review covers the dependency/public-type boundary, updated
guidance, exact diff composition, and this test admission correction.
