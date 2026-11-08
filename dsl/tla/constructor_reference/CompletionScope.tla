---------------------------- MODULE CompletionScope ----------------------------
EXTENDS Naturals
CONSTANT TraverseOrdinary
VARIABLES ownerDone, pendingDone, dependentDone, backEdge, phase
vars == <<ownerDone, pendingDone, dependentDone, backEdge, phase>>
Init == /\ ownerDone = FALSE /\ pendingDone = FALSE /\ dependentDone = FALSE
        /\ backEdge = "owner" /\ phase = "running"
CompletePending == /\ phase = "running" /\ ~pendingDone /\ pendingDone' = TRUE
                   /\ UNCHANGED <<ownerDone, dependentDone, backEdge, phase>>
EvaluateOrdinary == /\ ~dependentDone /\ dependentDone' = TRUE
                    /\ (TraverseOrdinary \/ ownerDone)
                    /\ backEdge' = IF ownerDone THEN "owner" ELSE "snapshot"
                    /\ UNCHANGED <<ownerDone, pendingDone, phase>>
CompleteOwner == /\ phase = "running" /\ pendingDone
                 /\ ownerDone' = TRUE /\ phase' = "complete"
                 /\ UNCHANGED <<pendingDone, dependentDone, backEdge>>
Next == CompletePending \/ EvaluateOrdinary \/ CompleteOwner
OwnedPendingComplete == ownerDone => pendingDone
StructuralOwnerRetained == ownerDone /\ dependentDone => backEdge = "owner"
Spec == Init /\ [][Next]_vars
=============================================================================
