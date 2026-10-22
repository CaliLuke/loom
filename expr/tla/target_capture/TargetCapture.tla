-------------------------- MODULE TargetCapture --------------------------
EXTENDS Naturals
CONSTANTS ReReadTarget, IgnoreContext, SharePolicy
VARIABLES raw, captured, ready, plans, cache
vars == <<raw, captured, ready, plans, cache>>
Policies == {"schema", "runtime"}
Contexts == {"owner", "foreign"}
Requests == Policies \X Contexts
Unset == "unset"

Init == /\ raw = 0
        /\ captured = 0
        /\ ready = FALSE
        /\ plans = [r \in Requests |-> [status |-> Unset, version |-> 0, use |-> Unset]]
        /\ cache = Unset

Capture == /\ ~ready
           /\ ready' = TRUE
           /\ captured' = raw
           /\ UNCHANGED <<raw, plans, cache>>

Mutate == /\ raw = 0
          /\ raw' = 1
          /\ UNCHANGED <<captured, ready, plans, cache>>

Build(r) ==
    /\ ready
    /\ plans[r].status = Unset
    /\ LET rejected == r[2] = "foreign" /\ ~IgnoreContext
           policy == IF SharePolicy /\ cache # Unset THEN cache ELSE r[1]
           value == IF ReReadTarget THEN raw ELSE captured
       IN /\ plans' = [plans EXCEPT ![r] =
                [status |-> IF rejected THEN "rejected" ELSE "accepted",
                 version |-> value, use |-> policy]]
          /\ cache' = IF rejected THEN cache ELSE policy
    /\ UNCHANGED <<raw, captured, ready>>

Next == Capture \/ Mutate \/ (\E r \in Requests : Build(r))
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
Accepted(r) == plans[r].status = "accepted"
SnapshotStable == \A r \in Requests : Accepted(r) => plans[r].version = captured
ContextIsolated == \A r \in Requests : Accepted(r) => r[2] = "owner"
PolicyIndependent == \A r \in Requests : Accepted(r) => plans[r].use = r[1]
Completes == <> (\A r \in Requests : plans[r].status # Unset)
=============================================================================
