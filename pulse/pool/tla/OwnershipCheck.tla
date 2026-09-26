-------------------------- MODULE OwnershipCheck --------------------------
EXTENDS Naturals
CONSTANT SyncStop
VARIABLES work, callback, requested
vars == <<work, callback, requested>>
Init == /\ work = "checking" /\ callback = "idle" /\ requested = FALSE
Reject == /\ work = "checking" /\ callback = "idle"
          /\ requested' = TRUE
          /\ IF SyncStop
                THEN /\ callback' = "joining" /\ UNCHANGED work
                ELSE /\ work' = "returning" /\ UNCHANGED callback
ExitWork == /\ work = "returning" /\ work' = "exited"
            /\ UNCHANGED <<callback, requested>>
StartStop == /\ ~SyncStop /\ requested /\ callback = "idle"
             /\ callback' = "joining" /\ UNCHANGED <<work, requested>>
FinishStop == /\ callback = "joining" /\ work = "exited"
              /\ callback' = "done" /\ UNCHANGED <<work, requested>>
Next == Reject \/ ExitWork \/ StartStop \/ FinishStop
Spec == Init /\ [][Next]_vars /\ WF_vars(Reject) /\ WF_vars(ExitWork)
        /\ WF_vars(StartStop) /\ WF_vars(FinishStop)
Stopped == <>(callback = "done")
TypeOK == /\ work \in {"checking", "returning", "exited"}
          /\ callback \in {"idle", "joining", "done"}
          /\ requested \in BOOLEAN
=============================================================================
