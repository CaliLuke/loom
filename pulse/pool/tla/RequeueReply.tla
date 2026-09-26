--------------------------- MODULE RequeueReply ---------------------------
(* Issue #436: a Redis reply does not prove whether the requeue ran.      *)
(* Scope: one successful release, two workers, one guarded start. Redis   *)
(* recovers and the orphan sweep eventually runs. Ownership races beyond *)
(* this handoff remain the scope of PoolOwnership.tla.                    *)
EXTENDS Naturals
CONSTANTS RestartOnFailure, ConcurrentRequeue
VARIABLES phase, local, remote, queued
vars == <<phase, local, remote, queued>>

Init == /\ phase = "running"
        /\ local = TRUE /\ remote = FALSE /\ queued = FALSE

Release == /\ phase = "running"
           /\ phase' = "calling" /\ local' = FALSE
           /\ UNCHANGED <<remote, queued>>

Apply == /\ phase = "calling"
         /\ phase' = "applied" /\ queued' = TRUE
         /\ UNCHANGED <<local, remote>>

FailBefore == /\ phase = "calling"
              /\ phase' = "failed"
              /\ UNCHANGED <<local, remote, queued>>

Reply == /\ phase = "applied"
         /\ phase' \in {"ok", "failed"}
         /\ UNCHANGED <<local, remote, queued>>

(* A concurrent guarded requeue wins between the precheck and the call. *)
Refused == /\ phase = "calling"
           /\ ConcurrentRequeue
           /\ phase' = "failed" /\ queued' = TRUE
           /\ UNCHANGED <<local, remote>>

Recover == /\ phase = "failed"
           /\ phase' = "ok"
           /\ local' = RestartOnFailure
           /\ UNCHANGED <<remote, queued>>

(* The sweep waits for the grace period and sees no owner or live guard. *)
Sweep == /\ phase = "ok" /\ ~local /\ ~remote /\ ~queued
         /\ queued' = TRUE
         /\ UNCHANGED <<phase, local, remote>>

Deliver == /\ queued /\ ~remote
           /\ remote' = TRUE /\ queued' = FALSE
           /\ UNCHANGED <<phase, local>>

Next == Release \/ Apply \/ FailBefore \/ Reply \/ Refused \/ Recover \/ Sweep \/ Deliver
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
TypeOK == /\ phase \in {"running", "calling", "applied", "failed", "ok"}
          /\ local \in BOOLEAN /\ remote \in BOOLEAN /\ queued \in BOOLEAN
AtMostOneRunner == ~(local /\ remote)
Recovered == <>remote
=============================================================================
