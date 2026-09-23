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
    \* ---- ghost (history) ----
    gen,       \* number of handler.Start calls for k
    runGen,    \* gen value of the run w holds for k
    staleWrite \* a write from a superseded run was accepted

redisVars == <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
               pool, wstream, wsExists>>
localVars == <<rep, nodeUp, stopped, fenced, jobs, running, runEp, cstate, orphanSeen>>
ghostVars == <<gen, runGen, staleWrite>>
vars == <<redisVars, localVars, ghostVars>>

-----------------------------------------------------------------------------
(* Helpers *)

Redis == [wmap |-> wmap, cl |-> cl, jm |-> jm, pl |-> pl]

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

\* Event ids and lock tokens are recycled once nothing references them.
UsedIds == {e.id : e \in pool}
           \cup UNION {{wstream[w][i].id : i \in 1..Len(wstream[w])} : w \in Workers}
           \cup {pend[k] : k \in Keys}
FreeIds == (1..MaxEv) \ UsedIds
UsedToks == {cl[w] : w \in Workers} \cup {cstate[n].tok : n \in Nodes}
            \cup {cstate[n].seen : n \in Nodes}
            \cup {rep[n].cl[w] : n \in Nodes, w \in Workers}
FreeToks == (1..MaxTok) \ UsedToks

AddStarts(R) ==
    /\ Cardinality(FreeIds) >= Cardinality(R)
    /\ LET f == CHOOSE f \in [R -> FreeIds] : \A a, b \in R : a # b => f[a] # f[b]
       IN pool' = pool \cup {[id |-> f[k], kind |-> "start", key |-> k] : k \in R}

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
    /\ rep = [n \in Nodes |-> Redis]
    /\ nodeUp = [n \in Nodes |-> TRUE]
    /\ stopped = [w \in Workers |-> FALSE]
    /\ fenced = [w \in Workers |-> FALSE]
    /\ jobs = [w \in Workers |-> {}]
    /\ running = [w \in Workers |-> ZeroRun]
    /\ runEp = [w \in Workers |-> ZeroRun]
    /\ cstate = [n \in Nodes |-> IdleC]
    /\ orphanSeen = [n \in Nodes |-> {}]
    /\ gen = [k \in Keys |-> 0]
    /\ runGen = [w \in Workers |-> ZeroRun]
    /\ staleWrite = FALSE

-----------------------------------------------------------------------------
(* Replication: rmap pub/sub delivery, per node, per map. *)
Replicate(n, m) ==
    /\ nodeUp[n]
    /\ rep[n][m] # Redis[m]
    /\ rep' = [rep EXCEPT ![n][m] = Redis[m]]
    /\ UNCHANGED <<redisVars, nodeUp, stopped, fenced, jobs, running, runEp,
                   cstate, orphanSeen, ghostVars>>

(* Time: node_membership.go:98 isWithinTTL becomes false. With self-fencing
   and a bounded pause/skew (LEASE_BOUND), others can see a live worker as
   stale only after the worker's own shorter local lease expired. *)
KeepAliveExpire(w) ==
    /\ ~kaStale[w]
    /\ \/ ~Alive(w) \/ stopped[w]
       \/ FALSE_DEATH /\ (~(FIX_SELF_FENCE /\ LEASE_BOUND) \/ fenced[w])
    /\ kaStale' = [kaStale EXCEPT ![w] = TRUE]
    /\ UNCHANGED <<wmap, ka, cl, staleTok, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, localVars, ghostVars>>

LockExpire(w) ==
    /\ LockFresh(cl[w])
    /\ LEASE_EXPIRE_HELD \/
       ~\E n \in Nodes : nodeUp[n] /\ cstate[n].ph = "holding" /\ cstate[n].tok = cl[w]
    /\ staleTok' = staleTok \cup {cl[w]}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, localVars, ghostVars>>

(* FIX_SELF_FENCE: the worker failed to refresh its keep-alive for
   workerTTL - margin, so it stops its handlers but remembers its jobs. *)
SelfFence(w) ==
    /\ FIX_SELF_FENCE /\ FALSE_DEATH
    /\ Alive(w) /\ ~stopped[w] /\ ~fenced[w] /\ ~kaStale[w]
    /\ fenced' = [fenced EXCEPT ![w] = TRUE]
    /\ running' = [running EXCEPT ![w] = ZeroRun]
    /\ UNCHANGED <<redisVars, rep, nodeUp, stopped, jobs, runEp, cstate,
                   orphanSeen, ghostVars>>

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
    /\ UNCHANGED <<wmap, cl, staleTok, jm, pl, pend, owner, ownerEp, pool,
                   wstream, wsExists, rep, nodeUp, stopped, runEp, cstate,
                   orphanSeen, ghostVars>>

-----------------------------------------------------------------------------
(* Client API *)

(* node_jobs.go:89 dispatchJob + scripts.go:21 luaClaimDispatch. *)
Dispatch(k) ==
    /\ FreeIds # {}
    /\ ~pl[k] /\ pend[k] = 0
    /\ FIX_OWNER => owner[k] = NoW
    /\ LET id == MinOf(FreeIds) IN
       /\ pend' = [pend EXCEPT ![k] = id]
       /\ pool' = pool \cup {[id |-> id, kind |-> "start", key |-> k]}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, owner, ownerEp,
                   wstream, wsExists, localVars, ghostVars>>

(* node_jobs.go:112-123 releaseDispatchPending after ack or timeout, plus the
   stale-guard paths (scripts.go:41-51, node_membership.go:29). *)
DispatchRelease(k) ==
    /\ pend[k] # 0
    /\ FIX_DISPATCH => \A e \in pool : e.id # pend[k]
    /\ pend' = [pend EXCEPT ![k] = 0]
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, owner, ownerEp, pool,
                   wstream, wsExists, localVars, ghostVars>>

(* node_jobs.go:207 StopJob *)
StopJob(k) ==
    /\ ENABLE_STOP
    /\ FreeIds # {}
    /\ pool' = pool \cup {[id |-> MinOf(FreeIds), kind |-> "stop", key |-> k]}
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   wstream, wsExists, localVars, ghostVars>>

-----------------------------------------------------------------------------
(* Routing: node_events.go:59 routeWorkerEvent; redelivery by XAUTOCLAIM
   (streaming/sink_consumers.go:308). *)
Route(n, e) ==
    /\ nodeUp[n]
    /\ e \in pool
    /\ LET A == Active(n) IN
       /\ A # {}
       /\ LET w == Owner(e.key, A) IN
          /\ wsExists[w]                       \* WithOnlyIfStreamExists
          /\ Len(wstream[w]) < MaxStream
          /\ REDELIVER_INFLIGHT \/ \A v \in Workers : \A i \in 1..Len(wstream[v]) : wstream[v][i] # e
          /\ wstream' = [wstream EXCEPT ![w] = Append(@, e)]
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   pool, wsExists, localVars, ghostVars>>

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
       /\ pool' = pool \ {e}
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
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, pend, wsExists, rep, nodeUp,
                   stopped, fenced, cstate, orphanSeen, staleWrite>>

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
                   orphanSeen, gen, runGen>>

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
    /\ UNCHANGED <<kaStale, staleTok, pl, pend, owner, ownerEp, pool,
                   rep, nodeUp, fenced, runEp, cstate, orphanSeen, ghostVars>>

(* Second loop: worker.go:354 rebalance. As coded it leaves jobMap[w] and the
   payload untouched. With FIX_OWNER it releases ownership (CAD) and jobMap. *)
Rebalance(n, w, k) ==
    /\ ENABLE_REBALANCE
    /\ nodeUp[n] /\ Host[w] = n /\ ~stopped[w] /\ ~Fenced(w)
    /\ rep[n].wmap[w] # "absent"
    /\ k \in jobs[w]
    /\ LET A == Active(n) IN
       /\ A # {}
       /\ Owner(k, A) # w
    /\ AddStarts({k})
    /\ running' = [running EXCEPT ![w][k] = IF @ > 0 THEN @ - 1 ELSE 0]
    /\ jobs' = [jobs EXCEPT ![w] = @ \ {k}]
    /\ ReleaseOwner(k, w)
    /\ jm' = IF FIX_OWNER THEN [jm EXCEPT ![w] = @ \ {k}] ELSE jm
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, pl, pend, ownerEp, wstream,
                   wsExists, rep, nodeUp, stopped, fenced, runEp, cstate,
                   orphanSeen, ghostVars>>

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
                   wstream, wsExists, rep, nodeUp, stopped, fenced, jobs,
                   running, runEp, orphanSeen, ghostVars>>

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
                   wstream, wsExists, rep, nodeUp, stopped, fenced, jobs, running,
                   runEp, orphanSeen, ghostVars>>

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
    /\ UNCHANGED <<kaStale, staleTok, pl, pend, ownerEp, rep, nodeUp, stopped,
                   fenced, jobs, running, runEp, orphanSeen, ghostVars>>

(* node_recovery.go:80 requeueOrphanedPayloads: per-node grace, no lock,
   local replicas only. With FIX_OWNER only unowned keys are requeued. *)
Orphan(n, k) ==
    /\ ENABLE_ORPHAN
    /\ nodeUp[n]
    /\ rep[n].pl[k]
    /\ \A w \in Workers : k \notin rep[n].jm[w]
    /\ IF k \notin orphanSeen[n]
       THEN /\ orphanSeen' = [orphanSeen EXCEPT ![n] = @ \cup {k}]
            /\ UNCHANGED pool
       ELSE /\ ORPHAN_GRACE_COVERS_LAG => (rep[n].jm = jm /\ rep[n].pl = pl)
            /\ FIX_OWNER => owner[k] = NoW
            /\ AddStarts({k})
            /\ orphanSeen' = [orphanSeen EXCEPT ![n] = @ \ {k}]
    /\ UNCHANGED <<wmap, ka, kaStale, cl, staleTok, jm, pl, pend, owner, ownerEp,
                   wstream, wsExists, rep, nodeUp, stopped, fenced, jobs, running,
                   runEp, cstate, ghostVars>>

OrphanClear(n, k) ==
    /\ nodeUp[n]
    /\ k \in orphanSeen[n]
    /\ ~rep[n].pl[k] \/ \E w \in Workers : k \in rep[n].jm[w]
    /\ orphanSeen' = [orphanSeen EXCEPT ![n] = @ \ {k}]
    /\ UNCHANGED <<redisVars, rep, nodeUp, stopped, fenced, jobs, running, runEp,
                   cstate, ghostVars>>

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
    /\ UNCHANGED <<redisVars, rep, stopped, fenced, runEp, orphanSeen, ghostVars>>

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
    /\ UNCHANGED <<kaStale, staleTok, pl, pend, ownerEp, rep, fenced, runEp,
                   cstate, orphanSeen, ghostVars>>

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

Next ==
    \/ DoReplicate \/ DoKeepAliveExp \/ DoLockExpire \/ DoKeepAlive \/ DoSelfFence
    \/ DoWorkerHandle \/ DoJobWrite \/ DoDispatch \/ DoDispatchRel \/ DoStopJob
    \/ DoRoute \/ DoEvict \/ DoCleanupBegin \/ DoCleanupAcquire \/ DoCleanupFinish
    \/ DoRebalance \/ DoNodeCrash \/ DoNodeClose \/ DoOrphan \/ DoOrphanClear

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

\* (4) liveness: a queued start event eventually results in the key running
\* on a live worker. Fairness covers the system's own progress, not faults or
\* client calls.
FairSpec == /\ Spec
            /\ WF_vars(DoReplicate) /\ WF_vars(DoRoute) /\ WF_vars(DoWorkerHandle)
            /\ WF_vars(DoKeepAliveExp) /\ WF_vars(DoLockExpire) /\ WF_vars(DoEvict)
            /\ WF_vars(DoCleanupBegin) /\ WF_vars(DoCleanupAcquire)
            /\ WF_vars(DoCleanupFinish) /\ WF_vars(DoOrphan) /\ WF_vars(DoOrphanClear)
            /\ WF_vars(DoDispatchRel)  \* DispatchJob always returns (ack or timeout)
EventuallyRuns ==
    \A k \in Keys :
        (\E e \in pool : e.key = k /\ e.kind = "start")
            ~> (\E w \in Workers : Alive(w) /\ running[w][k] > 0)

-----------------------------------------------------------------------------
(* Model-checking helpers *)
DefaultHost == [w \in Workers |-> IF w = "w1" THEN "n1" ELSE "n2"]
DefaultOrder == <<"w1", "w2">>
\* State constraint for the cleanup-lock configs: only w1 is ever cleaned up.
LockScope == ~kaStale["w2"]

=============================================================================
