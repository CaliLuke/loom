-------------------------- MODULE WorkflowAdmission --------------------------
\* Admission identity for #1024: caller-chosen IDs, one encoding per Submit
\* attempt, uncertain native responses, duplicate reconciliation, typed lookup,
\* and worker completion across a crash and restart.
\*
\* Native assumptions (Selvedge v0.7.0 behavior, not proved here): an insert is
\* atomic per instance ID; ErrWorkflowStartRejected means that call inserted
\* nothing; stored input bytes round-trip; completion is fenced by the lease, so
\* a result is recorded once (upstream docs/formal/Selvedge.tla).
EXTENDS Naturals, FiniteSets, Sequences

\* rq[r].outcome is the caller's conclusion for request r, derived from the
\* per-call results Loom returns (Created, Created=false, conflict, rejection,
\* uncertain) and the encoded identities of r's earlier uncertain attempts.
CONSTANT Mode
ASSUME Mode \in {"checked", "reencode", "newIdOnUncertain", "rejectFinal", "conflictFinal",
                 "decodeFirst"}

Requests == {"r1", "r2"}
IDs == {"j1", "j2"}
None == "none"
\* A stateful MarshalJSON may produce different bytes per call; the two values'
\* possible encodings overlap in "x2".
EncodingsOf(v) == IF v = "v" THEN {"x1", "x2"} ELSE {"x2", "x3"}
MaxAttempts == 2
MaxCrashes == 1

VARIABLES wf, val, store, rq, workerUp, workerEverUp, crashes, decodedOther
vars == <<wf, val, store, rq, workerUp, workerEverUp, crashes, decodedOther>>

Instance(name, bytes, parent, origin) ==
    [present |-> TRUE, name |-> name, bytes |-> bytes, parent |-> parent, origin |-> origin,
     status |-> "running", results |-> 0]
Absent == [present |-> FALSE, name |-> None, bytes |-> None, parent |-> None, origin |-> None,
           status |-> None, results |-> 0]

Init ==
    /\ wf \in {[r \in Requests |-> IF r = "r1" THEN "A" ELSE n] : n \in {"A", "B"}}
    /\ val \in {[r \in Requests |-> IF r = "r1" THEN "v" ELSE w] : w \in {"v", "w"}}
    \* j1 may already hold a child instance started by a parent workflow.
    /\ store \in {[j \in IDs |-> Absent],
                  [j \in IDs |-> IF j = "j1" THEN Instance("A", "x1", "p", "parent") ELSE Absent]}
    /\ rq = [r \in Requests |-> [phase |-> "idle", id |-> "j1", bytes |-> None,
                                 attempts |-> 0, uncertainSeen |-> FALSE, prior |-> {},
                                 outcome |-> None, created |-> FALSE]]
    /\ workerUp = FALSE /\ workerEverUp = FALSE /\ crashes = 0 /\ decodedOther = FALSE

\* A Submit call encodes its input exactly once. Both the insert and any
\* reconciliation use those bytes.
Encode(r) ==
    /\ rq[r].phase = "idle" /\ rq[r].attempts < MaxAttempts
    /\ \E b \in EncodingsOf(val[r]) :
          rq' = [rq EXCEPT ![r].phase = "encoded", ![r].bytes = b,
                           ![r].attempts = @ + 1, ![r].outcome = None]
    /\ UNCHANGED <<wf, val, store, workerUp, workerEverUp, crashes, decodedOther>>

\* An uncertain call may have committed its own bytes; the caller keeps them.
Uncertain(r) == [rq EXCEPT ![r].phase = "idle", ![r].uncertainSeen = TRUE, ![r].outcome = "uncertain",
                           ![r].prior = @ \cup {rq[r].bytes},
                           ![r].id = IF Mode = "newIdOnUncertain" THEN "j2" ELSE @]

\* Native start by name with an explicit instance ID. A rejection (including
\* Loom's ErrNotRunning) happens before the store is consulted and proves only
\* that this call inserted nothing.
Insert(r) ==
    /\ rq[r].phase = "encoded"
    /\ LET id == rq[r].id
           inst == Instance(wf[r], rq[r].bytes, None, r)
       IN \/ /\ UNCHANGED store
             \* After an uncertain attempt the job's state is still unknown.
             \* A rejected call inserted nothing, so its bytes are not a candidate.
             /\ rq' = IF rq[r].uncertainSeen /\ Mode # "rejectFinal"
                      THEN [Uncertain(r) EXCEPT ![r].prior = rq[r].prior]
                      ELSE [rq EXCEPT ![r].phase = "done", ![r].outcome = "rejected"]
          \/ IF ~store[id].present
             THEN \/ /\ store' = [store EXCEPT ![id] = inst]
                     /\ rq' = [rq EXCEPT ![r].phase = "done", ![r].outcome = "created", ![r].created = TRUE]
                  \/ /\ store' = [store EXCEPT ![id] = inst] /\ rq' = Uncertain(r)
                  \/ /\ UNCHANGED store /\ rq' = Uncertain(r)
             ELSE \/ /\ rq' = [rq EXCEPT ![r].phase = "reconcile"] /\ UNCHANGED store
                  \/ /\ rq' = Uncertain(r) /\ UNCHANGED store
    /\ UNCHANGED <<wf, val, workerUp, workerEverUp, crashes, decodedOther>>

\* Duplicate ID: Loom reads the stored instance and compares name, parent and
\* this call's bytes (Created=false or a conflict). reencode is the rejected
\* alternative that compares a fresh encoding. A conflict proves only a mismatch
\* with this call: the caller compares the conflict's stored identity with its
\* earlier uncertain attempts, whose commit it may be. conflictFinal is the
\* rejected blanket reading of a conflict as "not admitted".
Reconcile(r) ==
    /\ rq[r].phase = "reconcile"
    /\ LET inst == store[rq[r].id]
           Ident(b) == inst.name = wf[r] /\ inst.parent = None /\ inst.bytes = b
       IN \/ rq' = Uncertain(r)
          \/ \E fresh \in EncodingsOf(val[r]) :
               LET cmp == IF Mode = "reencode" THEN fresh ELSE rq[r].bytes
                   conclusion == CASE Ident(cmp) -> "existing"
                                   [] Mode # "conflictFinal" /\ \E b \in rq[r].prior : Ident(b) -> "earlier"
                                   [] OTHER -> "conflict"
               IN rq' = [rq EXCEPT ![r].phase = "done", ![r].outcome = conclusion]
    /\ UNCHANGED <<wf, val, store, workerUp, workerEverUp, crashes, decodedOther>>

WorkerStart ==
    /\ ~workerUp
    /\ workerUp' = TRUE /\ workerEverUp' = TRUE
    /\ UNCHANGED <<wf, val, store, rq, crashes, decodedOther>>

WorkerCrash ==
    /\ workerUp /\ crashes < MaxCrashes
    /\ workerUp' = FALSE /\ crashes' = crashes + 1
    /\ UNCHANGED <<wf, val, store, rq, workerEverUp, decodedOther>>

\* Only a worker completes a job; the lease fences a second completion.
Complete(id) ==
    /\ workerUp /\ store[id].present /\ store[id].status = "running"
    /\ store' = [store EXCEPT ![id].status = "completed", ![id].results = @ + 1]
    /\ UNCHANGED <<wf, val, rq, workerUp, workerEverUp, crashes, decodedOther>>

\* Lookup reads the instance and rejects a different workflow name before it
\* decodes output; decodeFirst decodes as the caller's type first.
Lookup(r) ==
    /\ store[rq[r].id].present
    /\ LET inst == store[rq[r].id]
       IN decodedOther' = (decodedOther \/ (Mode = "decodeFirst" /\ inst.name # wf[r] /\ inst.status = "completed"))
    /\ UNCHANGED <<wf, val, store, rq, workerUp, workerEverUp, crashes>>

Next ==
    \/ \E r \in Requests : Encode(r) \/ Insert(r) \/ Reconcile(r) \/ Lookup(r)
    \/ WorkerStart \/ WorkerCrash \/ \E id \in IDs : Complete(id)
Spec == Init /\ [][Next]_vars /\ WF_vars(WorkerStart) /\ \A id \in IDs : WF_vars(Complete(id))

Admitted(r) == rq[r].outcome \in {"created", "existing"}
\* Created and Created=false name a job whose stored name and bytes are this
\* call's; "earlier" names a job holding an earlier uncertain attempt's bytes.
AcceptedIdentity ==
    \A r \in Requests :
        LET inst == store[rq[r].id]
        IN /\ Admitted(r) => inst.present /\ inst.name = wf[r] /\ inst.parent = None
                             /\ inst.bytes = rq[r].bytes
           /\ rq[r].outcome = "earlier" => inst.present /\ inst.name = wf[r] /\ inst.parent = None
                                           /\ inst.bytes \in rq[r].prior
\* A "not admitted" conflict conclusion differs from every identity r submitted.
ConflictSound ==
    \A r \in Requests : rq[r].outcome = "conflict" =>
        LET inst == store[rq[r].id]
        IN inst.name # wf[r] \/ inst.parent # None \/ inst.bytes \notin rq[r].prior \cup {rq[r].bytes}
\* A "not admitted" conflict conclusion means the request has no job.
ConflictMeansAbsent ==
    \A r \in Requests : rq[r].outcome = "conflict" => \A id \in IDs : ~store[id].present \/ store[id].origin # r
\* Created is reported at most once per ID, by the call that inserted it.
CreatedOnce ==
    \A r \in Requests : rq[r].created => store[rq[r].id].origin = r
\* Retrying an uncertain admission never creates a second job.
OneJobPerRequest == \A r \in Requests : Cardinality({id \in IDs : store[id].present /\ store[id].origin = r}) <= 1
\* A final rejection means the request has no job.
RejectedMeansAbsent ==
    \A r \in Requests : rq[r].outcome = "rejected" => \A id \in IDs : ~store[id].present \/ store[id].origin # r
\* Producers never complete work: before any worker starts, every job is pending.
PendingUntilWorker == ~workerEverUp => \A id \in IDs : ~store[id].present \/ store[id].status = "running"
CompletedOnce == \A id \in IDs : store[id].results <= 1
LookupNeverDecodesOther == ~decodedOther
\* Durability: with the worker eventually up, every admitted job completes.
Durability == \A id \in IDs : store[id].present ~> (store[id].present /\ store[id].status = "completed")
=============================================================================
