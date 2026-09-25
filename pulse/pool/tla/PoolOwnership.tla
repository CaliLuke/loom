---------------------------- MODULE PoolOwnership ----------------------------
(***************************************************************************)
(* Job ownership in pulse/pool, as of main 967f4fbe, plus the owner-record *)
(* redesign described in roadmap/pulse-pool-ownership.md.                  *)
(*                                                                         *)
(* Redis is modeled as a set of authoritative variables. Every node keeps  *)
(* a per-map replica (pulse/rmap) that is refreshed only by an explicit    *)
(* Replicate action, so a node may act on any past version of each map.   *)
(* Each rmap uses its own pub/sub subscription, so maps replicate          *)
(* independently of each other.                                            *)
(*                                                                         *)
(* Time is abstract: "stale" keep-alives, cleanup locks, pending guards    *)
(* and orphan grace periods become true through explicit actions that may  *)
(* fire at any moment. See README.md for the mapping to the Go code.       *)
(***************************************************************************)
EXTENDS Naturals, FiniteSets, Sequences

CONSTANTS
    Nodes,            \* pool nodes
    Workers,          \* workers (each hosted by exactly one node)
    Keys,             \* job keys
    Host,             \* Host[w]: node that runs worker w
    WOrder,           \* sequence of Workers: deterministic hash order
    NoW,              \* "no worker" sentinel
    MaxEv,            \* bound on concurrently referenced event ids (ids are recycled)
    MaxTok,           \* bound on concurrently referenced cleanup-lock tokens (recycled)
    MaxStream,        \* bound on events buffered in one worker stream
    MaxEpoch,         \* bound on owner epochs / handler starts per key (epoch configs only)
    RepMaps,          \* replicated maps that Replicate may refresh
    \* Environment toggles
    FALSE_DEATH,      \* a live worker's keep-alive may look stale (GC pause, skew, Redis latency)
    LEASE_BOUND,      \* with FIX_SELF_FENCE: a live worker looks stale to others only after it fenced itself
    LEASE_EXPIRE_HELD,\* a cleanup lock may go stale while its holder still works
    ENABLE_STOP,      \* model StopJob
    ENABLE_CLOSE,     \* model graceful Node.Close
    ENABLE_CRASH,     \* model node process crash
    ENABLE_REBALANCE, \* model Worker.rebalance
    ENABLE_ORPHAN,    \* model requeueOrphanedPayloads
    ENABLE_WRITES,    \* model handler side-effect writes (for NoStaleWrite)
    ALLOW_PARTIAL,    \* cleanupWorker's poolStream.Add may fail (early return)
    REDELIVER_INFLIGHT, \* sink may redeliver while a routed copy is still queued/unacked
    ORPHAN_GRACE_COVERS_LAG, \* replicas converge within the orphan grace period
    PRE_BA2AF97C,     \* cleanup lock as before ba2af97c: unconditional Delete, then SetIfNotExists
    \* Pool stream detail (issue #385). With TRACK_STREAM = FALSE the model is
    \* exactly the one before #385: no trimming, atomic routing, no TTL.
    TRACK_STREAM,     \* pool stream entries have a delivery state, MAXLEN may trim them,
                      \* and routing is two steps (sink delivery, then worker XADD)
    REDIS7,           \* XAUTOCLAIM purges the pending entry of a trimmed id (Redis 7+);
                      \* the same state change as the sink acking deleted ids (#411)
    ENABLE_TTL,       \* pendingEventTTL passes for an event no worker stream or router holds
    GUARD_DESIGN,     \* how the dispatch guard tells a live start event from a gone one:
                      \* "asis", "marker", "marker_check", "minid" or "ackclear" (README)
    \* Lost requeued starts (issue #416).
    REQUEUE_DESIGN,   \* "asis": main; "acktrim": trimming never removes an unacked entry;
                      \* "release": rebalance removes the key from jobMap[w];
                      \* "guard": release, and rebalance and orphan requeues write the
                      \* dispatch guard, which the orphan sweep respects; a rebalance
                      \* refused by an in-flight guard keeps the job and does not retry;
                      \* "guard_retry": guard, and a refused rebalance retries (README)
    SPLIT_ACK,        \* the router's XACK of a start event (luaAckStart, which deletes the
                      \* guard) is a later step than the worker's startJob (AckStart)
    \* Fixes and redesign steps (FALSE = main as written)
    FIX_LOCK_RELEASE, \* removeWorkerFromMaps only deletes the cleanup lock it owns (token-checked)
    FIX_DEDUP,        \* startJob returns early when key already in w.jobs
    FIX_DISPATCH,     \* pending guard kept while the dispatch start event is unacked
    FIX_EVICT_STOP,   \* handleWorkerMapUpdate calls handler.Stop for evicted local worker jobs
    FIX_OWNER,        \* authoritative owner record: claim CAS in startJob, CAD on release,
                      \* atomic token-checked cleanup script reading owners from Redis
    FIX_RECHECK,      \* cleanup script aborts if the worker's keep-alive is fresh in Redis
    FIX_SELF_FENCE,   \* worker stops its handlers when it cannot refresh its keep-alive,
                      \* and revalidates ownership (owner + epoch) before resuming
    FIX_EPOCH         \* handler writes carry the owner epoch; stale-epoch writes are rejected

ASSUME Host \in [Workers -> Nodes]
ASSUME NoW \notin Workers

\* The keep-alive map is not replicated separately: its lag only matters as
\* "a live worker looks dead", which KeepAliveExpire with FALSE_DEATH covers.
\* RepMaps (a subset of these four) selects which replicas ever update. The
\* self-fencing configs replicate only wmap and cl: with FIX_OWNER and the
\* orphan sweep off, jm/pl replicas only gate cleanup candidates.
ASSUME RepMaps \subseteq {"wmap", "cl", "jm", "pl"}
MapNames == RepMaps

\* Epochs and start generations are tracked only when a config needs them.
TrackEp == FIX_OWNER /\ (FIX_SELF_FENCE \/ FIX_EPOCH \/ ENABLE_WRITES)

VARIABLES
    \* ---- Redis (authoritative) ----
    wmap,      \* workerMap: [w -> {"absent","live","dash"}] ("dash" = "-" inactive marker)
    ka,        \* workerKeepAliveMap presence: [w -> BOOLEAN]
    kaStale,   \* last keep-alive older than workerTTL (time abstraction): [w -> BOOLEAN]
    cl,        \* workerCleanupMap: [w -> 0..MaxTok], 0 = no lock
    staleTok,  \* cleanup-lock timestamps older than workerTTL
    jm,        \* jobMap worker -> set of keys
    pl,        \* jobPayloadMap presence: [k -> BOOLEAN]
    pend,      \* jobPendingMap guard: [k -> 0..MaxEv] (dispatch event id), 0 = none
    owner,     \* FIX_OWNER: owner record [k -> Workers \cup {NoW}]
    ownerEp,   \* FIX_OWNER: per-key epoch counter, never decreases
    pool,      \* unacked pool stream events: set of [id, kind, key]
    wstream,   \* worker streams: [w -> Seq(event)]
    wsExists,  \* worker stream exists: [w -> BOOLEAN]
    routed,    \* GUARD_DESIGN marker*: ids of routed start events not yet acked
    \* ---- node-local state ----
    rep,       \* rep[n][m]: node n's replica of map m
    nodeUp,    \* node process running
    stopped,   \* worker.stop() called
    fenced,    \* FIX_SELF_FENCE: worker suspended its handlers (lease lost locally)
    jobs,      \* Worker.jobs (local sync.Map)
    running,   \* handler-level: #Start - #Stop per (w,k), capped at 2
    runEp,     \* epoch a worker holds for its run of k
    cstate,    \* cleanup progress per node: [ph, w, tok, seen]
    orphanSeen,\* node.orphanedPayloads first-seen marks
    inbox,     \* TRACK_STREAM: event a node's router read from the sink and has not
               \* added to a worker stream yet (NoEv when none)
    acking,    \* SPLIT_ACK: start event ids a worker handled whose XACK has not run yet
    rebal,     \* guard designs: rebalance of worker w is due (a worker map or keep-alive
               \* change triggered handleWorkerMapUpdate); always TRUE otherwise
    \* ---- ghost (history) ----
    gen,       \* number of handler.Start calls for k
    runGen,    \* gen value of the run w holds for k
    staleWrite \* a write from a superseded run was accepted

redisVars == <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
               pool, wstream, wsExists, routed, acking>>
localVars == <<rep, nodeUp, stopped, fenced, jobs, running, runEp, cstate, orphanSeen,
               inbox, rebal>>
ghostVars == <<gen, runGen, staleWrite>>
vars == <<redisVars, localVars, ghostVars>>

-----------------------------------------------------------------------------
(* Helpers *)

Redis == [wmap |-> wmap, cl |-> cl, jm |-> jm, pl |-> pl]

\* Rebalance runs only when triggered, and a refused rebalance clears the
\* trigger, in the guard designs (issue #416 review).
Triggered == REQUEUE_DESIGN \in {"guard", "guard_retry"}

MinOf(S) == CHOOSE i \in S : \A j \in S : i <= j

\* Deterministic hash over an active-worker set: first worker of WOrder in S.
Owner(k, S) == WOrder[CHOOSE i \in 1..Len(WOrder) :
                        /\ WOrder[i] \in S
                        /\ \A j \in 1..(i-1) : WOrder[j] \notin S]

LockFresh(tok) == tok # 0 /\ tok \notin staleTok

\* node_membership.go:127 activeWorkers, evaluated on node n's replicas.
Active(n) == {w \in Workers :
                /\ rep[n].wmap[w] = "live"
                /\ ~LockFresh(rep[n].cl[w])
                /\ ka[w]
                /\ ~kaStale[w]}

\* A pool stream record. st is its state in Redis (TRACK_STREAM only, else
\* always "new"):
\*   "new"    in the stream, not yet delivered to the sink group
\*   "pend"   in the stream, delivered and unacked (in the pending list)
\*   "trim"   trimmed by MAXLEN while unacked; still in the pending list
\*   "purged" trimmed, then its pending entry purged by XAUTOCLAIM (REDIS7)
\* An acked record, or one trimmed before delivery, is dropped from pool.
\* disp marks a start event added by Dispatch (TRACK_STREAM only), which is
\* the only kind of event a dispatch guard names.
Rec(id, kind, k, disp) == [id |-> id, kind |-> kind, key |-> k, disp |-> disp, st |-> "new"]
\* The copy that routing adds to a worker stream.
Copy(e) == [id |-> e.id, kind |-> e.kind, key |-> e.key, disp |-> e.disp]
NoEv == [id |-> 0, kind |-> "none", key |-> "none", disp |-> FALSE]

\* Event ids and lock tokens are recycled once nothing references them.
UsedIds == {e.id : e \in pool}
           \cup UNION {{wstream[w][i].id : i \in 1..Len(wstream[w])} : w \in Workers}
           \cup {pend[k] : k \in Keys}
           \cup {inbox[n].id : n \in {m \in Nodes : inbox[m] # NoEv}}
           \cup routed \cup acking
FreeIds == (1..MaxEv) \ UsedIds
UsedToks == {cl[w] : w \in Workers} \cup {cstate[n].tok : n \in Nodes}
            \cup {cstate[n].seen : n \in Nodes}
            \cup {rep[n].cl[w] : n \in Nodes, w \in Workers}
FreeToks == (1..MaxTok) \ UsedToks


AddStarts(R) ==
    /\ Cardinality(FreeIds) >= Cardinality(R)
    /\ LET f == CHOOSE f \in [R -> FreeIds] : \A a, b \in R : a # b => f[a] # f[b]
       IN pool' = pool \cup {Rec(f[k], "start", k, FALSE) : k \in R}

\* Redis view of an event id, as in_flight reads it.
RecOf(id) == {e \in pool : e.id = id}
InPEL(id) == \E e \in RecOf(id) : e.st \in {"pend", "trim"}
Undelivered(id) == \E e \in RecOf(id) : e.st = "new"
Purged(id) == \E e \in RecOf(id) : e.st = "purged"

\* in_flight(id) of the dispatch guard, per design. Without TRACK_STREAM every
\* record is "new", so each design reduces to FIX_DISPATCH as before #385:
\* the guard stays while its event is unacked.
\*   asis          pending, or in the stream and undelivered (main)
\*   marker*       asis, or routed and not yet acked by a worker
\*   minid         asis; trimming never removes a pending entry
\*   ackclear      asis, or delivered and gone from both stream and pending
\*                 list: it may have been purged unacked. A worker ack clears
\*                 the guard in the XACK script, so an acked event never
\*                 reaches this check.
InFlight(id) ==
    \/ InPEL(id) \/ Undelivered(id)
    \/ GUARD_DESIGN \in {"marker", "marker_check"} /\ id \in routed
    \/ GUARD_DESIGN = "ackclear" /\ Purged(id)

\* Copies of an event held by a router or queued on a worker stream.
Held(id) == \/ \E n \in Nodes : inbox[n] # NoEv /\ inbox[n].id = id
            \/ \E w \in Workers : \E i \in 1..Len(wstream[w]) : wstream[w][i].id = id
            \/ id \in acking

Alive(w) == nodeUp[Host[w]]
Fenced(w) == FIX_SELF_FENCE /\ fenced[w]

\* node_cleanup.go:40 removeWorkerFromMaps + stream.Destroy (deleteWorker).
\* As coded, the cleanup lock is deleted unconditionally (node_cleanup.go:47).
DeleteWorkerRedis(w, myTok) ==
    /\ wmap' = [wmap EXCEPT ![w] = "absent"]
    /\ ka' = [ka EXCEPT ![w] = FALSE]
    /\ cl' = IF ~FIX_LOCK_RELEASE \/ (myTok # 0 /\ cl[w] = myTok)
             THEN [cl EXCEPT ![w] = 0] ELSE cl
    /\ jm' = [jm EXCEPT ![w] = {}]
    /\ wsExists' = [wsExists EXCEPT ![w] = FALSE]
    /\ wstream' = [wstream EXCEPT ![w] = <<>>]

\* Release ownership of k by w (compare-and-delete); the epoch counter stays.
ReleaseOwner(k, w) ==
    owner' = IF FIX_OWNER /\ owner[k] = w THEN [owner EXCEPT ![k] = NoW] ELSE owner

ZeroRun == [k \in Keys |-> 0]
IdleC == [ph |-> "idle", w |-> NoW, tok |-> 0, seen |-> 0]

-----------------------------------------------------------------------------
Init ==
    /\ wmap = [w \in Workers |-> "live"]
    /\ ka = [w \in Workers |-> TRUE]
    /\ kaStale = [w \in Workers |-> FALSE]
    /\ cl = [w \in Workers |-> 0]
    /\ staleTok = {}
    /\ jm = [w \in Workers |-> {}]
    /\ pl = [k \in Keys |-> FALSE]
    /\ pend = [k \in Keys |-> 0]
    /\ owner = [k \in Keys |-> NoW]
    /\ ownerEp = [k \in Keys |-> 0]
    /\ pool = {}
    /\ wstream = [w \in Workers |-> <<>>]
    /\ wsExists = [w \in Workers |-> TRUE]
    /\ routed = {}
    /\ rep = [n \in Nodes |-> Redis]
    /\ nodeUp = [n \in Nodes |-> TRUE]
    /\ stopped = [w \in Workers |-> FALSE]
    /\ fenced = [w \in Workers |-> FALSE]
    /\ jobs = [w \in Workers |-> {}]
    /\ running = [w \in Workers |-> ZeroRun]
    /\ runEp = [w \in Workers |-> ZeroRun]
    /\ cstate = [n \in Nodes |-> IdleC]
    /\ orphanSeen = [n \in Nodes |-> {}]
    /\ inbox = [n \in Nodes |-> NoEv]
    /\ acking = {}
    /\ rebal = [w \in Workers |-> ~Triggered]
    /\ gen = [k \in Keys |-> 0]
    /\ runGen = [w \in Workers |-> ZeroRun]
    /\ staleWrite = FALSE

-----------------------------------------------------------------------------
(* Replication: rmap pub/sub delivery, per node, per map. *)
Replicate(n, m) ==
    /\ nodeUp[n]
    /\ rep[n][m] # Redis[m]
    /\ rep' = [rep EXCEPT ![n][m] = Redis[m]]
    \* node_events.go watchWorkers: a worker map update runs
    \* handleWorkerMapUpdate, which rebalances the node's workers.
    /\ rebal' = IF Triggered /\ m = "wmap"
                THEN [w \in Workers |-> IF Host[w] = n THEN TRUE ELSE rebal[w]]
                ELSE rebal
    /\ UNCHANGED <<redisVars, nodeUp, stopped, fenced, jobs, running, runEp,
                   cstate, orphanSeen, inbox, ghostVars>>

(* Time: node_membership.go:98 isWithinTTL becomes false. With self-fencing
   and a bounded pause/skew (LEASE_BOUND), others can see a live worker as
   stale only after the worker's own shorter local lease expired. *)
KeepAliveExpire(w) ==
    /\ ~kaStale[w]
    /\ \/ ~Alive(w) \/ stopped[w]
       \/ FALSE_DEATH /\ (~(FIX_SELF_FENCE /\ LEASE_BOUND) \/ fenced[w])
    /\ kaStale' = [kaStale EXCEPT ![w] = TRUE]
    /\ UNCHANGED <<wmap, ka, cl, staleTok, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, routed, localVars, ghostVars, acking>>

LockExpire(w) ==
    /\ LockFresh(cl[w])
    /\ LEASE_EXPIRE_HELD \/
       ~\E n \in Nodes : nodeUp[n] /\ cstate[n].ph = "holding" /\ cstate[n].tok = cl[w]
    /\ staleTok' = staleTok \cup {cl[w]}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, routed, localVars, ghostVars, acking>>

(* FIX_SELF_FENCE: the worker failed to refresh its keep-alive for
   workerTTL - margin, so it stops its handlers but remembers its jobs. *)
SelfFence(w) ==
    /\ FIX_SELF_FENCE /\ FALSE_DEATH
    /\ Alive(w) /\ ~stopped[w] /\ ~fenced[w] /\ ~kaStale[w]
    /\ fenced' = [fenced EXCEPT ![w] = TRUE]
    /\ running' = [running EXCEPT ![w] = ZeroRun]
    /\ UNCHANGED <<redisVars, rep, nodeUp, stopped, jobs, runEp, cstate,
                   orphanSeen, inbox, ghostVars, rebal>>

(* worker.go:331 keepAlive: Set recreates the entry even after deletion.
   With FIX_SELF_FENCE a fenced worker then resumes only the jobs whose owner
   record still names it with the same epoch. *)
KeepAlive(w) ==
    /\ Alive(w) /\ ~stopped[w]
    /\ (~ka[w] \/ kaStale[w] \/ Fenced(w))
    /\ ka' = [ka EXCEPT ![w] = TRUE]
    /\ kaStale' = [kaStale EXCEPT ![w] = FALSE]
    /\ IF Fenced(w)
       THEN LET keep == {k \in jobs[w] : owner[k] = w /\ ownerEp[k] = runEp[w][k]} IN
            /\ fenced' = [fenced EXCEPT ![w] = FALSE]
            /\ jobs' = [jobs EXCEPT ![w] = keep]
            /\ running' = [running EXCEPT ![w] = [k \in Keys |-> IF k \in keep THEN 1 ELSE 0]]
       ELSE UNCHANGED <<fenced, jobs, running>>
    \* A worker that returns stands for a worker that joins: its worker map
    \* update makes every node rebalance its workers.
    /\ rebal' = IF Triggered /\ (~ka[w] \/ kaStale[w])
                THEN [v \in Workers |-> IF Alive(v) THEN TRUE ELSE rebal[v]]
                ELSE rebal
    /\ UNCHANGED <<wmap, cl, staleTok, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, routed, rep, nodeUp, stopped, runEp, cstate,
                   orphanSeen, inbox, ghostVars, acking>>

-----------------------------------------------------------------------------
(* Client API *)

(* node_jobs.go:89 dispatchJob + scripts.go:21 luaClaimDispatch. *)
Dispatch(k) ==
    /\ FreeIds # {}
    /\ ~pl[k] /\ pend[k] = 0
    /\ FIX_OWNER => owner[k] = NoW
    /\ LET id == MinOf(FreeIds) IN
       /\ pend' = [pend EXCEPT ![k] = id]
       /\ pool' = pool \cup {Rec(id, "start", k, TRACK_STREAM)}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, owner, ownerEp,
                   wstream, wsExists, routed, localVars, ghostVars, acking>>

(* node_jobs.go:112-123 releaseDispatchPending after ack or timeout, plus the
   stale-guard paths (scripts.go:41-51, node_membership.go:29). With
   FIX_DISPATCH every path runs luaReleaseDispatch, which clears the guard
   only when in_flight is false. A purged record whose guard is released is
   of no further use and is dropped. *)
DispatchRelease(k) ==
    /\ pend[k] # 0
    /\ FIX_DISPATCH => ~InFlight(pend[k])
    /\ pend' = [pend EXCEPT ![k] = 0]
    /\ pool' = {e \in pool : ~(e.id = pend[k] /\ e.st = "purged")}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, owner, ownerEp,
                   wstream, wsExists, routed, localVars, ghostVars, acking>>

(* node_jobs.go:207 StopJob *)
StopJob(k) ==
    /\ ENABLE_STOP
    /\ FreeIds # {}
    /\ pool' = pool \cup {Rec(MinOf(FreeIds), "stop", k, FALSE)}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   wstream, wsExists, routed, localVars, ghostVars, acking>>

-----------------------------------------------------------------------------
(* Routing: node_events.go:59 routeWorkerEvent; redelivery by XAUTOCLAIM
   (streaming/sink_consumers.go:308). *)
Route(n, e) ==
    /\ ~TRACK_STREAM
    /\ nodeUp[n]
    /\ e \in pool
    /\ LET A == Active(n) IN
       /\ A # {}
       /\ LET w == Owner(e.key, A) IN
          /\ wsExists[w]                       \* WithOnlyIfStreamExists
          /\ Len(wstream[w]) < MaxStream
          /\ REDELIVER_INFLIGHT \/ ~Held(e.id)
          /\ wstream' = [wstream EXCEPT ![w] = Append(@, Copy(e))]
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   pool, wsExists, routed, localVars, ghostVars, acking>>

(* TRACK_STREAM routing, in two steps. Deliver: the sink hands an entry that
   is in the stream to node n's router, by XREADGROUP (first delivery) or by
   the sink idle check (redelivery of a pending entry, after ackGracePeriod).
   A trimmed entry is never delivered again: the idle check acks its pending
   entry on every version (#411), as Redis 7 XAUTOCLAIM purges it
   (AutoClaim). *)
Deliver(n, e) ==
    /\ TRACK_STREAM
    /\ nodeUp[n] /\ inbox[n] = NoEv
    /\ e \in pool /\ e.st \in {"new", "pend"}
    /\ REDELIVER_INFLIGHT \/ ~Held(e.id)
    /\ pool' = (pool \ {e}) \cup {[e EXCEPT !.st = "pend"]}
    /\ inbox' = [inbox EXCEPT ![n] = Copy(e)]
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   wstream, wsExists, routed, rep, nodeUp, stopped, fenced, jobs,
                   running, runEp, cstate, orphanSeen, ghostVars, acking, rebal>>

(* RouteIn: routeWorkerEvent adds the delivered event to the worker stream
   (node_events.go:91). The router can stall between Deliver and RouteIn for
   longer than ackGracePeriod, so the pending entry can be claimed, trimmed or
   purged meanwhile. The marker designs set the routed marker in the same
   script as the worker XADD. marker_check refuses to route an event whose
   pending entry is gone. *)
RouteIn(n) ==
    /\ TRACK_STREAM
    /\ nodeUp[n] /\ inbox[n] # NoEv
    /\ LET e == inbox[n]
           A == Active(n)
       IN
       /\ inbox' = [inbox EXCEPT ![n] = NoEv]
       /\ IF GUARD_DESIGN = "marker_check" /\ ~InPEL(e.id)
          THEN UNCHANGED <<wstream, routed>>
          ELSE /\ A # {}
               /\ LET w == Owner(e.key, A) IN
                  /\ wsExists[w]
                  /\ Len(wstream[w]) < MaxStream
                  /\ wstream' = [wstream EXCEPT ![w] = Append(@, e)]
               /\ routed' = IF GUARD_DESIGN \in {"marker", "marker_check"} /\ e.kind = "start"
                             THEN routed \cup {e.id} ELSE routed
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   pool, wsExists, rep, nodeUp, stopped, fenced, jobs, running,
                   runEp, cstate, orphanSeen, ghostVars, acking, rebal>>

(* The router fails to route (no active worker, missing worker stream, a
   client error): the event stays pending for redelivery. *)
RouteDrop(n) ==
    /\ TRACK_STREAM
    /\ nodeUp[n] /\ inbox[n] # NoEv
    /\ inbox' = [inbox EXCEPT ![n] = NoEv]
    /\ UNCHANGED <<redisVars, rep, nodeUp, stopped, fenced, jobs, running, runEp,
                   cstate, orphanSeen, ghostVars, rebal>>

(* MAXLEN ~ maxQueuedJobs trims an entry once enough events were added after
   it; any pool stream XADD can trim. The model lets any entry go, in any
   order. With minid, a trim never passes the oldest pending entry, and
   undelivered entries are newer than every pending one, so only undelivered
   entries go, and only while nothing is pending. An entry trimmed before
   delivery is lost; one trimmed while pending stays in the pending list. *)
Trim(e) ==
    /\ TRACK_STREAM
    /\ e \in pool /\ e.st \in {"new", "pend"}
    /\ GUARD_DESIGN = "minid" => (e.st = "new" /\ \A f \in pool : ~InPEL(f.id))
    /\ REQUEUE_DESIGN # "acktrim"    \* every record in pool is unacked
    /\ pool' = IF e.st = "new" THEN pool \ {e}
               ELSE (pool \ {e}) \cup {[e EXCEPT !.st = "trim"]}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   wstream, wsExists, routed, localVars, ghostVars, acking>>

(* XAUTOCLAIM reaches a trimmed pending entry after ackGracePeriod. Redis 7
   and later purge it from the pending list, and the sink acks it on every
   version since #411 (REDIS7); Redis 6.2 before #411 kept it, so the model
   has no step for that case. Only a purged dispatch start whose guard
   still names it is kept, because only in_flight reads it. *)
AutoClaim(e) ==
    /\ TRACK_STREAM /\ REDIS7
    /\ e \in pool /\ e.st = "trim"
    /\ pool' = IF e.disp /\ pend[e.key] = e.id
               THEN (pool \ {e}) \cup {[e EXCEPT !.st = "purged"]}
               ELSE pool \ {e}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   wstream, wsExists, routed, localVars, ghostVars, acking>>

(* pendingEventTTL passes. Assumption: it exceeds the time an event waits on
   a worker stream or in a router, so no copy is held. After it, routing acks
   the event as stale, in_flight is false by age, and any routed marker has
   expired. The guard is then released by DispatchRelease. *)
EventExpire(e) ==
    /\ TRACK_STREAM /\ ENABLE_TTL
    /\ e \in pool /\ ~Held(e.id)
    /\ pool' = pool \ {e}
    /\ routed' = routed \ {e.id}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   wstream, wsExists, localVars, ghostVars, acking>>

(* worker.go:186 handleEvents -> startJob (worker.go:248) / stopJob
   (worker.go:277), then the ack path. With FIX_OWNER, startJob's claim is one
   Lua script: take the key if unowned (new epoch), re-enter if owned by w,
   otherwise drop the start. *)
StartEffects(w, k) ==
    LET fresh == owner[k] = NoW
        ep == IF TrackEp /\ fresh THEN ownerEp[k] + 1 ELSE ownerEp[k]
    IN
    /\ TrackEp => (ep <= MaxEpoch /\ gen[k] < MaxEpoch)
    /\ jm' = [jm EXCEPT ![w] = @ \cup {k}]
    /\ pl' = [pl EXCEPT ![k] = TRUE]
    /\ running' = [running EXCEPT ![w][k] = IF @ < 2 THEN @ + 1 ELSE 2]
    /\ jobs' = [jobs EXCEPT ![w] = @ \cup {k}]
    /\ owner' = IF FIX_OWNER THEN [owner EXCEPT ![k] = w] ELSE owner
    /\ ownerEp' = [ownerEp EXCEPT ![k] = ep]
    /\ runEp' = [runEp EXCEPT ![w][k] = ep]
    /\ IF TrackEp
       THEN /\ gen' = [gen EXCEPT ![k] = @ + 1]
            /\ runGen' = [runGen EXCEPT ![w][k] = gen[k] + 1]
       ELSE UNCHANGED <<gen, runGen>>

WorkerHandle(w) ==
    /\ Alive(w) /\ ~stopped[w] /\ wsExists[w] /\ ~Fenced(w)
    /\ wstream[w] # <<>>
    /\ LET e == Head(wstream[w])
           k == e.key IN
       /\ wstream' = [wstream EXCEPT ![w] = Tail(@)]
       \* The worker ack: XACK, which also clears the routed marker (marker
       \* designs) or the guard that names this event (ackclear). With
       \* SPLIT_ACK the XACK of a start is a later step (AckStart).
       /\ IF SPLIT_ACK /\ e.kind = "start"
          THEN /\ acking' = acking \cup {e.id}
               /\ UNCHANGED <<pool, routed, pend>>
          ELSE /\ pool' = {p \in pool : p.id # e.id}
               /\ routed' = routed \ {e.id}
               /\ pend' = IF GUARD_DESIGN = "ackclear" /\ e.kind = "start" /\ pend[k] = e.id
                          THEN [pend EXCEPT ![k] = 0] ELSE pend
               /\ UNCHANGED acking
       /\ IF e.kind = "start"
          THEN IF \/ (FIX_OWNER /\ owner[k] \notin {NoW, w})
                  \/ ((FIX_DEDUP \/ FIX_OWNER) /\ k \in jobs[w])
               THEN UNCHANGED <<jm, pl, running, jobs, owner, ownerEp, runEp, gen, runGen>>
               ELSE StartEffects(w, k)
          ELSE /\ UNCHANGED <<ownerEp, runEp, gen, runGen>>
               /\ IF k \in jobs[w]
                  THEN /\ running' = [running EXCEPT ![w][k] = IF @ > 0 THEN @ - 1 ELSE 0]
                       /\ jobs' = [jobs EXCEPT ![w] = @ \ {k}]
                       /\ jm' = [jm EXCEPT ![w] = @ \ {k}]
                       /\ pl' = [pl EXCEPT ![k] = FALSE]
                       /\ ReleaseOwner(k, w)
                  ELSE UNCHANGED <<jm, pl, running, jobs, owner>>
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, wsExists, rep, nodeUp,
                   stopped, fenced, cstate, orphanSeen, inbox, staleWrite, rebal>>

(* SPLIT_ACK: the worker's ack reaches the router node, which runs
   luaAckStart (node_events.go ackRoutedEvent): XACK, and delete the guard
   that names the event. *)
AckStart(id) ==
    /\ SPLIT_ACK /\ id \in acking
    /\ acking' = acking \ {id}
    /\ pool' = {p \in pool : p.id # id}
    /\ routed' = routed \ {id}
    /\ pend' = [k \in Keys |-> IF GUARD_DESIGN = "ackclear" /\ pend[k] = id THEN 0 ELSE pend[k]]
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, owner, ownerEp,
                   wstream, wsExists, localVars, ghostVars>>

(* A handler side-effect write (ENABLE_WRITES). With FIX_EPOCH the write
   carries runEp and Redis rejects it unless owner/epoch still match; on
   rejection the worker stops that handler. Only writes that would be flagged
   or rejected are modeled; other writes do not change state. *)
JobWrite(w, k) ==
    /\ ENABLE_WRITES /\ TrackEp
    /\ Alive(w) /\ running[w][k] > 0
    /\ LET ok == ~FIX_EPOCH \/ (owner[k] = w /\ ownerEp[k] = runEp[w][k])
           stale == runGen[w][k] < gen[k]
       IN
       /\ (ok /\ stale) \/ ~ok
       /\ IF ok
          THEN /\ staleWrite' = TRUE
               /\ UNCHANGED <<running, jobs>>
          ELSE /\ running' = [running EXCEPT ![w][k] = 0]
               /\ jobs' = [jobs EXCEPT ![w] = @ \ {k}]
               /\ UNCHANGED staleWrite
    /\ UNCHANGED <<redisVars, rep, nodeUp, stopped, fenced, runEp, cstate,
                   orphanSeen, inbox, gen, runGen, rebal>>

-----------------------------------------------------------------------------
(* Worker map watcher: node_events.go:226 handleWorkerMapUpdate. *)

(* First loop (node_events.go:231-245): deleteWorker + worker.stop, which
   never calls handler.Stop (worker.go:230-245). *)
Evict(n, w) ==
    /\ nodeUp[n] /\ Host[w] = n /\ ~stopped[w]
    /\ rep[n].wmap[w] = "absent"
    /\ DeleteWorkerRedis(w, 0)
    /\ stopped' = [stopped EXCEPT ![w] = TRUE]
    /\ IF FIX_EVICT_STOP
       THEN /\ running' = [running EXCEPT ![w] = ZeroRun]
            /\ jobs' = [jobs EXCEPT ![w] = {}]
       ELSE UNCHANGED <<running, jobs>>
    /\ UNCHANGED <<kaStale, staleTok, pl, pend, owner, ownerEp, pool, routed,
                   rep, nodeUp, fenced, runEp, cstate, orphanSeen, inbox, ghostVars, acking, rebal>>

(* REQUEUE_DESIGN "guard": a requeue adds its start event and writes the
   dispatch guard that names it in one script, as Dispatch does. It is
   refused while another guard of the key is held. The script replaces a
   guard that is no longer in flight; the model releases it first
   (DispatchRelease). *)
GuardedStart(k) ==
    /\ FreeIds # {} /\ pend[k] = 0
    /\ LET id == MinOf(FreeIds) IN
       /\ pend' = [pend EXCEPT ![k] = id]
       /\ pool' = pool \cup {Rec(id, "start", k, TRACK_STREAM)}

\* A requeue of k by rebalance or the orphan sweep, per REQUEUE_DESIGN.
Requeue(k) ==
    IF Triggered THEN GuardedStart(k)
    ELSE AddStarts({k}) /\ UNCHANGED pend

(* Second loop: worker.go:354 rebalance. As coded it leaves jobMap[w] and the
   payload untouched. With FIX_OWNER it releases ownership (CAD) and jobMap.
   REQUEUE_DESIGN "release" and the guard designs remove the key from
   jobMap[w] too, before the start event is added (issue #416). *)
Rebalance(n, w, k) ==
    /\ ENABLE_REBALANCE
    /\ nodeUp[n] /\ Host[w] = n /\ ~stopped[w] /\ ~Fenced(w)
    /\ rep[n].wmap[w] # "absent"
    /\ k \in jobs[w]
    /\ LET A == Active(n) IN
       /\ A # {}
       /\ Owner(k, A) # w
    \* A pass with nothing to move keeps the trigger: the model leaves out
    \* the join race in which a pass runs before a start lands on w (README).
    /\ rebal[w]
    /\ IF Triggered /\ pend[k] # 0 /\ InFlight(pend[k])
       THEN \* Refused: a start of the key may still start it, so the job
            \* keeps running here. "guard" clears the trigger, so nothing
            \* rebalances the job until the next trigger. "guard_retry"
            \* retries after ackGracePeriod: the pass stays due.
            /\ REQUEUE_DESIGN = "guard"
            /\ rebal' = [rebal EXCEPT ![w] = FALSE]
            /\ UNCHANGED <<pool, pend, running, jobs, owner, jm>>
       ELSE /\ Requeue(k)
            /\ running' = [running EXCEPT ![w][k] = IF @ > 0 THEN @ - 1 ELSE 0]
            /\ jobs' = [jobs EXCEPT ![w] = @ \ {k}]
            /\ ReleaseOwner(k, w)
            /\ jm' = IF FIX_OWNER \/ REQUEUE_DESIGN \in {"release", "guard", "guard_retry"}
                     THEN [jm EXCEPT ![w] = @ \ {k}] ELSE jm
            /\ rebal' = IF Triggered THEN [rebal EXCEPT ![w] = FALSE] ELSE rebal
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, pl, ownerEp, wstream,
                   wsExists, routed, rep, nodeUp, stopped, fenced, runEp, cstate,
                   orphanSeen, inbox, ghostVars, acking>>

-----------------------------------------------------------------------------
(* Cleanup: node_recovery.go:36 cleanupInactiveWorkers -> cleanupWorker
   (node_recovery.go:138) -> acquireCleanupLock / claimCleanupLock
   (node_membership.go:52-93). *)

\* Step 1: candidate selection and the lock read on the LOCAL replica
\* (node_membership.go:53). The observed value is kept as `seen`. Before
\* ba2af97c a stale lock was then deleted unconditionally.
CleanupBegin(n, w) ==
    /\ nodeUp[n] /\ cstate[n].ph = "idle"
    /\ rep[n].wmap[w] # "absent" \/ rep[n].jm[w] # {}
    /\ w \notin Active(n)
    /\ ~LockFresh(rep[n].cl[w])            \* node_recovery.go:60, node_membership.go:78
    /\ LET seen == rep[n].cl[w] IN
       /\ cl' = IF PRE_BA2AF97C /\ seen # 0 THEN [cl EXCEPT ![w] = 0] ELSE cl
       /\ cstate' = [cstate EXCEPT ![n] = [ph |-> "try", w |-> w, tok |-> 0, seen |-> seen]]
    /\ UNCHANGED <<wmap, ka, kaStale, staleTok, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, routed, rep, nodeUp, stopped, fenced, jobs,
                   running, runEp, orphanSeen, inbox, ghostVars, acking, rebal>>

\* Step 2: SetIfNotExists when nothing was seen (node_membership.go:65), else
\* TestAndSetEx(seen, now) (node_membership.go:82). Pre-ba2af97c: always
\* SetIfNotExists after the delete.
CleanupAcquire(n) ==
    /\ nodeUp[n] /\ cstate[n].ph = "try"
    /\ LET w == cstate[n].w
           seen == cstate[n].seen
           expect == IF PRE_BA2AF97C THEN 0 ELSE seen
       IN
       IF cl[w] = expect
       THEN /\ FreeToks # {}
            /\ LET t == MinOf(FreeToks) IN
               /\ cl' = [cl EXCEPT ![w] = t]
               /\ staleTok' = (staleTok \cap UsedToks) \ {t}
               /\ cstate' = [cstate EXCEPT ![n] = [ph |-> "holding", w |-> w, tok |-> t, seen |-> 0]]
       ELSE /\ cstate' = [cstate EXCEPT ![n] = IdleC]
            /\ UNCHANGED <<cl, staleTok>>
    /\ UNCHANGED <<wmap, ka, kaStale, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, routed, rep, nodeUp, stopped, fenced, jobs, running,
                   runEp, orphanSeen, inbox, ghostVars, acking, rebal>>

\* Step 3, as coded: requeue keys read from the LOCAL jobMap / payload replicas
\* (node_recovery.go:145, 161), then deleteWorker (193). partial models a
\* failed poolStream.Add: early return at 186 leaving the lock in place.
\* With FIX_OWNER this is one Lua script: check the lock token, optionally
\* re-check keep-alive, requeue and release the keys whose owner is w (read
\* from Redis), delete the worker.
CleanupFinishAsCoded(n, w) ==
    LET R == {k \in rep[n].jm[w] : rep[n].pl[k]}
        Gone == rep[n].jm[w] \ R
    IN
    \E partial \in (IF R = {} \/ ~ALLOW_PARTIAL THEN {FALSE} ELSE BOOLEAN) :
      IF partial
      THEN /\ jm' = [jm EXCEPT ![w] = @ \ Gone]
           /\ UNCHANGED <<wmap, ka, cl, pool, wstream, wsExists, owner>>
      ELSE /\ AddStarts(R)
           /\ UNCHANGED owner
           /\ DeleteWorkerRedis(w, cstate[n].tok)

CleanupFinishScript(n, w) ==
    LET R == {k \in Keys : owner[k] = w} IN
    IF \/ cl[w] # cstate[n].tok
       \/ (FIX_RECHECK /\ ka[w] /\ ~kaStale[w])
    THEN \* abort: lock lost, or the worker is alive after all
         /\ cl' = IF cl[w] = cstate[n].tok THEN [cl EXCEPT ![w] = 0] ELSE cl
         /\ UNCHANGED <<wmap, ka, jm, pool, wstream, wsExists, owner>>
    ELSE /\ AddStarts(R)
         /\ owner' = [k \in Keys |-> IF k \in R THEN NoW ELSE owner[k]]
         /\ DeleteWorkerRedis(w, cstate[n].tok)

CleanupFinish(n) ==
    /\ nodeUp[n] /\ cstate[n].ph = "holding"
    /\ IF FIX_OWNER THEN CleanupFinishScript(n, cstate[n].w)
                    ELSE CleanupFinishAsCoded(n, cstate[n].w)
    /\ cstate' = [cstate EXCEPT ![n] = IdleC]
    /\ UNCHANGED <<kaStale, staleTok, pl, pend, ownerEp, routed, rep, nodeUp, stopped,
                   fenced, jobs, running, runEp, orphanSeen, inbox, ghostVars, acking, rebal>>

(* node_recovery.go:80 requeueOrphanedPayloads: per-node grace, no lock,
   local replicas only. With FIX_OWNER only unowned keys are requeued. With
   REQUEUE_DESIGN "guard" a key whose guard is held is not requeued, and the
   requeue writes a guard (GuardedStart). *)
Orphan(n, k) ==
    /\ ENABLE_ORPHAN
    /\ nodeUp[n]
    /\ rep[n].pl[k]
    /\ \A w \in Workers : k \notin rep[n].jm[w]
    /\ IF k \notin orphanSeen[n]
       THEN /\ orphanSeen' = [orphanSeen EXCEPT ![n] = @ \cup {k}]
            /\ UNCHANGED <<pool, pend>>
       ELSE /\ ORPHAN_GRACE_COVERS_LAG => (rep[n].jm = jm /\ rep[n].pl = pl)
            /\ FIX_OWNER => owner[k] = NoW
            /\ Requeue(k)
            /\ orphanSeen' = [orphanSeen EXCEPT ![n] = @ \ {k}]
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, owner, ownerEp,
                   wstream, wsExists, routed, rep, nodeUp, stopped, fenced, jobs, running,
                   runEp, cstate, inbox, ghostVars, acking, rebal>>

OrphanClear(n, k) ==
    /\ nodeUp[n]
    /\ k \in orphanSeen[n]
    /\ ~rep[n].pl[k] \/ \E w \in Workers : k \in rep[n].jm[w]
    /\ orphanSeen' = [orphanSeen EXCEPT ![n] = @ \ {k}]
    /\ UNCHANGED <<redisVars, rep, nodeUp, stopped, fenced, jobs, running, runEp,
                   cstate, inbox, ghostVars, rebal>>

-----------------------------------------------------------------------------
(* Process lifecycle *)

NodeCrash(n) ==
    /\ ENABLE_CRASH
    /\ nodeUp[n]
    /\ Cardinality({m \in Nodes : nodeUp[m]}) > 1
    /\ nodeUp' = [nodeUp EXCEPT ![n] = FALSE]
    /\ running' = [w \in Workers |-> IF Host[w] = n THEN ZeroRun ELSE running[w]]
    /\ jobs' = [w \in Workers |-> IF Host[w] = n THEN {} ELSE jobs[w]]
    /\ cstate' = [cstate EXCEPT ![n] = IdleC]
    /\ inbox' = [inbox EXCEPT ![n] = NoEv]
    /\ UNCHANGED <<redisVars, rep, stopped, fenced, runEp, orphanSeen, ghostVars, rebal>>

(* node_jobs.go:311 close(shutdown=false): requeueAllJobs then removeWorker.
   Atomic. With FIX_OWNER, ownership of the requeued keys is released. *)
NodeClose(n) ==
    /\ ENABLE_CLOSE
    /\ nodeUp[n] /\ cstate[n].ph = "idle"
    /\ Cardinality({m \in Nodes : nodeUp[m]}) > 1
    /\ LET L == {w \in Workers : Host[w] = n /\ ~stopped[w]}
           R == UNION {jobs[w] : w \in L}
       IN
       /\ AddStarts(R)
       /\ wmap' = [w \in Workers |-> IF w \in L THEN "absent" ELSE wmap[w]]
       /\ ka' = [w \in Workers |-> IF w \in L THEN FALSE ELSE ka[w]]
       /\ cl' = [w \in Workers |-> IF w \in L /\ ~FIX_LOCK_RELEASE THEN 0 ELSE cl[w]]
       /\ jm' = [w \in Workers |-> IF w \in L THEN {} ELSE jm[w]]
       /\ wsExists' = [w \in Workers |-> IF w \in L THEN FALSE ELSE wsExists[w]]
       /\ wstream' = [w \in Workers |-> IF w \in L THEN <<>> ELSE wstream[w]]
       /\ owner' = IF FIX_OWNER
                   THEN [k \in Keys |-> IF owner[k] \in L THEN NoW ELSE owner[k]]
                   ELSE owner
       /\ stopped' = [w \in Workers |-> IF w \in L THEN TRUE ELSE stopped[w]]
       /\ running' = [w \in Workers |-> IF Host[w] = n THEN ZeroRun ELSE running[w]]
       /\ jobs' = [w \in Workers |-> IF Host[w] = n THEN {} ELSE jobs[w]]
    /\ nodeUp' = [nodeUp EXCEPT ![n] = FALSE]
    /\ inbox' = [inbox EXCEPT ![n] = NoEv]
    /\ UNCHANGED <<kaStale, staleTok, pl, pend, ownerEp, routed, rep, fenced, runEp,
                   cstate, orphanSeen, ghostVars, acking, rebal>>

-----------------------------------------------------------------------------
\* Named wrappers so TLC traces show the action name.
DoReplicate      == \E n \in Nodes, m \in MapNames : Replicate(n, m)
DoKeepAliveExp   == \E w \in Workers : KeepAliveExpire(w)
DoLockExpire     == \E w \in Workers : LockExpire(w)
DoKeepAlive      == \E w \in Workers : KeepAlive(w)
DoSelfFence      == \E w \in Workers : SelfFence(w)
DoWorkerHandle   == \E w \in Workers : WorkerHandle(w)
DoJobWrite       == \E w \in Workers, k \in Keys : JobWrite(w, k)
DoDispatch       == \E k \in Keys : Dispatch(k)
DoDispatchRel    == \E k \in Keys : DispatchRelease(k)
DoStopJob        == \E k \in Keys : StopJob(k)
DoRoute          == \E n \in Nodes, e \in pool : Route(n, e)
DoEvict          == \E n \in Nodes, w \in Workers : Evict(n, w)
DoCleanupBegin   == \E n \in Nodes, w \in Workers : CleanupBegin(n, w)
DoCleanupAcquire == \E n \in Nodes : CleanupAcquire(n)
DoCleanupFinish  == \E n \in Nodes : CleanupFinish(n)
DoRebalance      == \E n \in Nodes, w \in Workers, k \in Keys : Rebalance(n, w, k)
DoNodeCrash      == \E n \in Nodes : NodeCrash(n)
DoNodeClose      == \E n \in Nodes : NodeClose(n)
DoOrphan         == \E n \in Nodes, k \in Keys : Orphan(n, k)
DoOrphanClear    == \E n \in Nodes, k \in Keys : OrphanClear(n, k)
DoDeliver        == \E n \in Nodes, e \in pool : Deliver(n, e)
DoRouteIn        == \E n \in Nodes : RouteIn(n)
DoRouteDrop      == \E n \in Nodes : RouteDrop(n)
DoTrim           == \E e \in pool : Trim(e)
DoAutoClaim      == \E e \in pool : AutoClaim(e)
DoEventExpire    == \E e \in pool : EventExpire(e)
DoAckStart       == \E i \in acking : AckStart(i)

Next ==
    \/ DoReplicate \/ DoKeepAliveExp \/ DoLockExpire \/ DoKeepAlive \/ DoSelfFence
    \/ DoWorkerHandle \/ DoJobWrite \/ DoDispatch \/ DoDispatchRel \/ DoStopJob
    \/ DoRoute \/ DoEvict \/ DoCleanupBegin \/ DoCleanupAcquire \/ DoCleanupFinish
    \/ DoRebalance \/ DoNodeCrash \/ DoNodeClose \/ DoOrphan \/ DoOrphanClear
    \/ DoDeliver \/ DoRouteIn \/ DoRouteDrop \/ DoTrim \/ DoAutoClaim \/ DoEventExpire
    \/ DoAckStart

Spec == Init /\ [][Next]_vars

-----------------------------------------------------------------------------
(* Properties *)

TypeOK ==
    /\ wmap \in [Workers -> {"absent", "live", "dash"}]
    /\ cl \in [Workers -> 0..MaxTok]
    /\ jm \in [Workers -> SUBSET Keys]
    /\ pend \in [Keys -> 0..MaxEv]
    /\ running \in [Workers -> [Keys -> 0..2]]
    /\ owner \in [Keys -> Workers \cup {NoW}]

\* (1) each key runs in at most one live worker's handler
AtMostOneRunner ==
    \A k \in Keys : Cardinality({w \in Workers : Alive(w) /\ running[w][k] > 0}) <= 1

\* (2) handler.Start is never called for a key already running on that worker
NoDoubleStartSameWorker ==
    \A w \in Workers, k \in Keys : running[w][k] <= 1

\* (3) two nodes never both believe they hold the cleanup lock for one worker
CleanupLockMutualExclusion ==
    \A n1, n2 \in Nodes :
        (n1 # n2 /\ nodeUp[n1] /\ nodeUp[n2]
         /\ cstate[n1].ph = "holding" /\ cstate[n2].ph = "holding")
        => cstate[n1].w # cstate[n2].w

\* (5) no write from a superseded run (an earlier handler.Start of k) is accepted
NoStaleWrite == ~staleWrite

\* (6) TRACK_STREAM: the dispatch guard stays while its start event can still
\* start the job: while a router holds it, a live worker's stream queues it,
\* or the sink can still deliver it. This is the promise B2 breaks. A router
\* copy that marker_check will refuse to route does not count.
LiveDispatchCopies ==
    {Copy(e) : e \in {f \in pool : f.disp /\ f.st \in {"new", "pend"}}}
    \cup {inbox[n] : n \in {m \in Nodes : nodeUp[m] /\ inbox[m] # NoEv /\ inbox[m].disp
                          /\ (GUARD_DESIGN = "marker_check" => InPEL(inbox[m].id))}}
    \cup UNION {{wstream[w][i] : i \in {j \in 1..Len(wstream[w]) : wstream[w][j].disp}} :
                w \in {v \in Workers : Alive(v) /\ ~stopped[v] /\ wsExists[v]}}
GuardHeld == \A e \in LiveDispatchCopies : pend[e.key] = e.id

\* (4) liveness: a queued start event eventually results in the key running
\* on a live worker. Fairness covers the system's own progress, not faults or
\* client calls.
FairSpec == /\ Spec
            /\ WF_vars(DoReplicate) /\ WF_vars(DoRoute) /\ WF_vars(DoWorkerHandle)
            /\ WF_vars(DoKeepAliveExp) /\ WF_vars(DoLockExpire) /\ WF_vars(DoEvict)
            /\ WF_vars(DoCleanupBegin) /\ WF_vars(DoCleanupAcquire)
            /\ WF_vars(DoCleanupFinish) /\ WF_vars(DoOrphan) /\ WF_vars(DoOrphanClear)
            /\ WF_vars(DoDispatchRel)  \* DispatchJob always returns (ack or timeout)
\* (7) liveness, TRACK_STREAM: a dispatch guard is eventually released, so no
\* guard leaks. The TTL is part of the fairness, as it is in Go.
\* The TTL is fair per event: expiring other events does not discharge it.
ExpireId(i) == \E e \in pool : e.id = i /\ EventExpire(e)
FairGuardSpec == /\ FairSpec
                 /\ WF_vars(DoDeliver) /\ WF_vars(DoRouteIn) /\ WF_vars(DoAutoClaim)
                 /\ \A i \in 1..MaxEv : SF_vars(ExpireId(i))
GuardReleased == \A k \in Keys : pend[k] # 0 ~> pend[k] = 0
EventuallyRuns ==
    \A k \in Keys :
        (\E e \in pool : e.key = k /\ e.kind = "start")
            ~> (\E w \in Workers : Alive(w) /\ running[w][k] > 0)

\* (8) liveness, under FairJobSpec: a job that has started and was not
\* stopped (its payload exists) runs again on a live worker, even when a
\* start event of it is lost, once the faults stop (issue #416).
\* EventuallyRuns follows a queued start event, so it cannot see a start
\* that trimming or the TTL removed. The faults are trims, failed routings,
\* TTL expiries and false deaths. The property needs them to stop
\* eventually: a pool that loses every event can run nothing, and a worker
\* that looks dead again and again can move a job forever.
\*
\* FairJobSpec differs from FairGuardSpec in three ways:
\*   - FairGuardSpec is weakly fair to DoReplicate as a whole, so one map can
\*     replicate forever while another never does. Each rmap has its own
\*     subscription, so FairJobSpec is fair to each replica, and to the
\*     orphan sweep of each node and key.
\*   - With FALSE_DEATH, WF_vars(DoKeepAliveExp) forces false deaths forever,
\*     which makes FaultsCease false and JobRecovered vacuous. FairJobSpec is
\*     fair only to the expiry of a dead or stopped worker's keep-alive.
\*   - The keep-alive loop of a live worker is fair (KeepAlive).
DeadExpire == \E w \in Workers : (~Alive(w) \/ stopped[w]) /\ KeepAliveExpire(w)
FalseDeath == \E w \in Workers : Alive(w) /\ ~stopped[w] /\ KeepAliveExpire(w)
FairJobSpec == /\ Spec
               /\ WF_vars(DoRoute) /\ WF_vars(DoWorkerHandle)
               /\ WF_vars(DeadExpire) /\ WF_vars(DoLockExpire) /\ WF_vars(DoEvict)
               /\ WF_vars(DoCleanupBegin) /\ WF_vars(DoCleanupAcquire)
               /\ WF_vars(DoCleanupFinish) /\ WF_vars(DoOrphanClear)
               /\ WF_vars(DoDispatchRel) /\ WF_vars(DoKeepAlive)
               /\ WF_vars(DoDeliver) /\ WF_vars(DoRouteIn) /\ WF_vars(DoAutoClaim)
               /\ \A i \in 1..MaxEv : SF_vars(ExpireId(i))
               /\ \A n \in Nodes, m \in MapNames : WF_vars(Replicate(n, m))
               /\ \A n \in Nodes, k \in Keys : WF_vars(Orphan(n, k))
               /\ WF_vars(DoRebalance) /\ WF_vars(DoAckStart)
FaultsCease == <>[][~(DoTrim \/ DoRouteDrop \/ DoEventExpire \/ FalseDeath)]_vars
JobRuns ==
    \A k \in Keys :
        pl[k] ~> (~pl[k] \/ \E w \in Workers : Alive(w) /\ running[w][k] > 0)
JobRecovered == FaultsCease => JobRuns

\* (9) liveness, under FairJobSpec: once the faults stop, a job that runs on
\* a worker that is not its owner among the workers really active stops
\* running there (rebalance moves it). StopJob and NotifyWorker route to
\* the owner, so a job left on another worker cannot be stopped or
\* notified (issue #416 review).
TrueActive == {w \in Workers : /\ wmap[w] = "live" /\ ka[w] /\ ~kaStale[w]
                               /\ Alive(w) /\ ~stopped[w] /\ ~LockFresh(cl[w])}
OffOwner(k, w) == /\ Alive(w) /\ running[w][k] > 0 /\ TrueActive # {}
                  /\ w # Owner(k, TrueActive)
JobMoves == \A k \in Keys, w \in Workers : OffOwner(k, w) ~> ~OffOwner(k, w)
OwnerReached == FaultsCease => JobMoves

-----------------------------------------------------------------------------
(* Model-checking helpers *)
DefaultHost == [w \in Workers |-> IF w = "w1" THEN "n1" ELSE "n2"]
DefaultOrder == <<"w1", "w2">>
\* State constraint for the cleanup-lock configs: only w1 is ever cleaned up.
LockScope == ~kaStale["w2"]
\* State constraint for the requeue_runner_* configs: cleanup never takes a
\* worker that is still running, so the known false-death double runner (B6)
\* is left out and rebalance can be checked on its own.
NoLiveCleanup == \A n \in Nodes : cstate[n].w = NoW \/ ~Alive(cstate[n].w) \/ stopped[cstate[n].w]

=============================================================================
