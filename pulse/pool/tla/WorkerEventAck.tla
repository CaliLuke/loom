--------------------------- MODULE WorkerEventAck ---------------------------
(***************************************************************************)
(* How a pool node matches a worker's ack to the pool event it routed      *)
(* (issue #381).                                                           *)
(*                                                                         *)
(* One node routes one pool start event. routeWorkerEvent XADDs a copy to  *)
(* the worker stream and records it in pendingEvents under the worker      *)
(* event id that the XADD returns. The worker acks that id on the node     *)
(* stream, and ackWorkerEvent looks the id up, XACKs the pool event and    *)
(* sends the dispatch return. XAUTOCLAIM redelivers the pool event while   *)
(* it is unacked, so the router can route it again under a new worker      *)
(* event id.                                                               *)
(*                                                                         *)
(* DESIGN selects how routing registers the pending event:                 *)
(*   "asis"   main as written: register after Add returns.                 *)
(*   "prereg" register before Add, unregister when Add fails. Go cannot    *)
(*            do this: XADD assigns the id. The design is here to compare. *)
(*   "lock"   one lock held across Add and the registration; the ack       *)
(*            lookup takes the same lock.                                  *)
(*   "park"   an ack that finds no registration is parked; registration    *)
(*            consumes a parked ack. Parked acks are pruned by age.        *)
(* See README.md for the mapping to the Go code.                           *)
(***************************************************************************)
EXTENDS Naturals, FiniteSets

CONSTANTS
    DESIGN,          \* "asis", "prereg", "lock" or "park"
    MaxRoutes,       \* bound on routings of the pool event (worker event ids)
    AMBIGUOUS_FAIL,  \* Add can return an error after Redis applied the XADD
    DUP_ACK,         \* the node stream can carry a second ack for one id
    SHUTDOWN,        \* the node can stop
    PRUNE_ANYTIME    \* park: a parked ack can be pruned while its Add runs

ASSUME DESIGN \in {"asis", "prereg", "lock", "park"}

Ids == 1..MaxRoutes

VARIABLES
    up,          \* node process runs
    sinkQ,       \* the sink holds a delivery of the pool event for the router
    poolAcked,   \* the pool event is XACKed
    rpc,         \* router: "idle", "adding", "ok" (Add returned ok), "fail"
    cur,         \* worker event id of the routing in progress
    nextId,      \* next worker event id XADD assigns
    lockHeld,    \* "lock": the router holds the registration lock
    wstream,     \* worker event ids in the worker stream, not yet handled
    nodeQ,       \* acks on the node stream, not yet processed: <<id, copy>>
    pending,     \* pendingEvents keys
    parked,      \* "park": acks that arrived before their registration
    \* History
    addedOk,     \* ids whose Add applied and returns ok
    workerAcked, \* ids the worker acked
    dropped,     \* ids whose ack was dropped (or pruned) before any ack matched
    matches      \* matches[id]: acks matched to the registration of id

vars == <<up, sinkQ, poolAcked, rpc, cur, nextId, lockHeld, wstream, nodeQ,
          pending, parked, addedOk, workerAcked, dropped, matches>>

NoId == 0

Init ==
    /\ up = TRUE
    /\ sinkQ = TRUE
    /\ poolAcked = FALSE
    /\ rpc = "idle"
    /\ cur = NoId
    /\ nextId = 1
    /\ lockHeld = FALSE
    /\ wstream = {}
    /\ nodeQ = {}
    /\ pending = {}
    /\ parked = {}
    /\ addedOk = {}
    /\ workerAcked = {}
    /\ dropped = {}
    /\ matches = [i \in Ids |-> 0]

TypeOK ==
    /\ up \in BOOLEAN /\ sinkQ \in BOOLEAN /\ poolAcked \in BOOLEAN
    /\ rpc \in {"idle", "adding", "ok", "fail"}
    /\ cur \in Ids \cup {NoId}
    /\ nextId \in 1..(MaxRoutes + 1)
    /\ lockHeld \in BOOLEAN
    /\ wstream \subseteq Ids
    /\ nodeQ \subseteq (Ids \X {1, 2})
    /\ pending \subseteq Ids /\ parked \subseteq Ids
    /\ addedOk \subseteq Ids /\ workerAcked \subseteq Ids /\ dropped \subseteq Ids
    /\ matches \in [Ids -> 0..2]

(* A matched ack XACKs the pool event and sends the dispatch return. *)
Match(id) ==
    /\ poolAcked' = TRUE
    /\ matches' = [matches EXCEPT ![id] = @ + 1]

(* Dropping an ack loses it only when no ack of that id matched. A late
   duplicate of a matched ack carries nothing new. *)
Lost(id) == IF matches[id] = 0 THEN {id} ELSE {}

-----------------------------------------------------------------------------
(* routeWorkerEvent: the sink delivers the pool event, the router picks the
   worker and calls stream.Add. The XADD assigns the next id. *)
RouteBegin ==
    /\ up /\ sinkQ /\ rpc = "idle" /\ nextId <= MaxRoutes
    /\ DESIGN = "lock" => ~lockHeld
    /\ sinkQ' = FALSE
    /\ rpc' = "adding"
    /\ cur' = nextId
    /\ nextId' = nextId + 1
    /\ lockHeld' = (DESIGN = "lock")
    /\ pending' = IF DESIGN = "prereg" THEN pending \cup {nextId} ELSE pending
    /\ UNCHANGED <<up, poolAcked, wstream, nodeQ, parked, addedOk, workerAcked,
                   dropped, matches>>

(* Redis applies the XADD. Add then returns ok, or with AMBIGUOUS_FAIL an
   error (for example a client timeout after the write). *)
AddApply ==
    /\ rpc = "adding"
    /\ wstream' = wstream \cup {cur}
    /\ \/ /\ rpc' = "ok"
          /\ addedOk' = addedOk \cup {cur}
       \/ /\ AMBIGUOUS_FAIL
          /\ rpc' = "fail"
          /\ UNCHANGED addedOk
    /\ UNCHANGED <<up, sinkQ, poolAcked, cur, nextId, lockHeld, nodeQ, pending,
                   parked, workerAcked, dropped, matches>>

(* The XADD fails without effect (worker stream gone, Redis down). *)
AddFail ==
    /\ rpc = "adding"
    /\ rpc' = "fail"
    /\ UNCHANGED <<up, sinkQ, poolAcked, cur, nextId, lockHeld, wstream, nodeQ,
                   pending, parked, addedOk, workerAcked, dropped, matches>>

(* Add returned ok: register the pending event ("prereg" registered it
   before the Add). *)
RouteRegister ==
    /\ up /\ rpc = "ok"
    /\ rpc' = "idle"
    /\ lockHeld' = FALSE
    /\ IF DESIGN = "park" /\ cur \in parked
       THEN /\ parked' = parked \ {cur}
            /\ Match(cur)
            /\ UNCHANGED pending
       ELSE /\ pending' = IF DESIGN = "prereg" THEN pending ELSE pending \cup {cur}
            /\ UNCHANGED <<parked, poolAcked, matches>>
    /\ UNCHANGED <<up, sinkQ, cur, nextId, wstream, nodeQ, addedOk, workerAcked,
                   dropped>>

(* Add returned an error: nothing stays registered. The pool event stays
   unacked, so XAUTOCLAIM redelivers it. *)
RouteFailed ==
    /\ up /\ rpc = "fail"
    /\ rpc' = "idle"
    /\ lockHeld' = FALSE
    /\ pending' = pending \ {cur}
    /\ UNCHANGED <<up, sinkQ, poolAcked, cur, nextId, wstream, nodeQ, parked,
                   addedOk, workerAcked, dropped, matches>>

(* The worker handles its event and acks it on the node stream. *)
WorkerAck(id) ==
    /\ id \in wstream
    /\ wstream' = wstream \ {id}
    /\ nodeQ' = nodeQ \cup {<<id, 1>>}
    /\ workerAcked' = workerAcked \cup {id}
    /\ UNCHANGED <<up, sinkQ, poolAcked, rpc, cur, nextId, lockHeld, pending,
                   parked, addedOk, dropped, matches>>

(* A second ack for the same id reaches the node stream. *)
DuplicateAck(id) ==
    /\ DUP_ACK
    /\ id \in workerAcked
    /\ <<id, 1>> \notin nodeQ
    /\ nodeQ' = nodeQ \cup {<<id, 2>>}
    /\ UNCHANGED <<up, sinkQ, poolAcked, rpc, cur, nextId, lockHeld, wstream,
                   pending, parked, addedOk, workerAcked, dropped, matches>>

(* ackWorkerEvent. "lock" waits for the router to release the lock. *)
ProcessAck(m) ==
    /\ up /\ m \in nodeQ
    /\ DESIGN = "lock" => ~lockHeld
    /\ LET id == m[1] IN
       IF id \in pending
       THEN /\ pending' = pending \ {id}
            /\ Match(id)
            /\ UNCHANGED <<parked, dropped>>
       ELSE /\ IF DESIGN = "park"
               THEN /\ parked' = parked \cup {id}
                    /\ UNCHANGED dropped
               ELSE /\ dropped' = dropped \cup Lost(id)
                    /\ UNCHANGED parked
            /\ UNCHANGED <<pending, poolAcked, matches>>
    /\ nodeQ' = nodeQ \ {m}
    /\ UNCHANGED <<up, sinkQ, rpc, cur, nextId, lockHeld, wstream, addedOk,
                   workerAcked>>

(* A parked ack older than the prune age is dropped. The age bound exceeds
   any Add call, so the router is not still adding that id, unless
   PRUNE_ANYTIME. *)
Prune(id) ==
    /\ id \in parked
    /\ PRUNE_ANYTIME \/ ~(rpc # "idle" /\ cur = id)
    /\ parked' = parked \ {id}
    /\ dropped' = dropped \cup Lost(id)
    /\ UNCHANGED <<up, sinkQ, poolAcked, rpc, cur, nextId, lockHeld, wstream,
                   nodeQ, pending, addedOk, workerAcked, matches>>

(* XAUTOCLAIM redelivers the unacked pool event after ackGracePeriod. *)
Redeliver ==
    /\ up /\ ~poolAcked /\ ~sinkQ
    /\ sinkQ' = TRUE
    /\ UNCHANGED <<up, poolAcked, rpc, cur, nextId, lockHeld, wstream, nodeQ,
                   pending, parked, addedOk, workerAcked, dropped, matches>>

(* The node stops. Its in-memory maps are gone; an XADD in flight may still
   apply. Another node's XAUTOCLAIM takes the pool event over. *)
Shutdown ==
    /\ SHUTDOWN /\ up
    /\ up' = FALSE
    /\ pending' = {}
    /\ parked' = {}
    /\ lockHeld' = FALSE
    /\ UNCHANGED <<sinkQ, poolAcked, rpc, cur, nextId, wstream, nodeQ, addedOk,
                   workerAcked, dropped, matches>>

Next ==
    \/ RouteBegin \/ AddApply \/ AddFail \/ RouteRegister \/ RouteFailed
    \/ \E id \in Ids : WorkerAck(id) \/ DuplicateAck(id) \/ Prune(id)
    \/ \E m \in nodeQ : ProcessAck(m)
    \/ Redeliver \/ Shutdown

Spec == Init /\ [][Next]_vars

(* The system's own progress is fair. Redelivery, faults and shutdown are
   not. *)
FairSpec ==
    /\ Spec
    /\ WF_vars(AddApply)
    /\ WF_vars(RouteRegister) /\ WF_vars(RouteFailed)
    /\ \A id \in Ids : WF_vars(WorkerAck(id)) /\ WF_vars(Prune(id))
    /\ \A id \in Ids, c \in {1, 2} : WF_vars(ProcessAck(<<id, c>>))

-----------------------------------------------------------------------------
(* Properties *)

(* An ack of an event whose Add returned ok is never dropped as unknown. *)
NoUnknownAckForRoutedEvent == dropped \cap addedOk = {}

(* A registration exists only for an event whose Add returned ok. *)
NoRegistrationWithoutAdd == rpc = "idle" => pending \subseteq addedOk

(* The pool event is acked only after the worker acked a routed copy. *)
AckedOnlyAfterWorker == poolAcked => workerAcked # {}

(* Each registration consumes at most one ack: one XACK, one dispatch
   return. *)
MatchedAtMostOnce == \A id \in Ids : matches[id] <= 1

(* Once everything in flight has settled, and before any redelivery: the pool
   event is acked if the worker acked a copy whose Add returned ok, and no
   registration or parked ack remains for an acked id. *)
Settled == up /\ rpc = "idle" /\ wstream = {} /\ nodeQ = {}

AckedWhenWorkerAcked ==
    Settled /\ (workerAcked \cap addedOk # {}) => poolAcked

NoLeakWhenSettled ==
    Settled => (pending \cap workerAcked = {})

(* Parked acks are bounded by the acks the worker sent. *)
ParkedBounded == parked \subseteq workerAcked

(* Liveness, under FairSpec: every parked ack is eventually consumed or
   pruned. *)
ParkedDrains == \A id \in Ids : [](id \in parked => <>(id \notin parked))

(* Liveness, under FairSpec: once the worker acked a copy whose Add returned
   ok, the pool event is acked without waiting for a redelivery, unless the
   node stops. *)
PromptAck == (workerAcked \cap addedOk # {}) ~> (poolAcked \/ ~up)

=============================================================================
