------------------------- MODULE SnapshotOrdering -------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC
CONSTANT Policy
VARIABLES authoritative, pending, published, captured, regressed
vars == <<authoritative,pending,published,captured,regressed>>
Init == /\ authoritative = 1 /\ pending = <<>> /\ published = 0
        /\ captured = {} /\ regressed = FALSE
Write == /\ authoritative = 1 /\ authoritative' = 2
         /\ UNCHANGED <<pending,published,captured,regressed>>
Capture == /\ authoritative \notin captured
           /\ (Policy = "Parallel" \/ Len(pending) = 0)
           /\ pending' = Append(pending, authoritative)
           /\ captured' = captured \cup {authoritative}
           /\ UNCHANGED <<authoritative,published,regressed>>
Publish == /\ Len(pending) > 0
           /\ \E i \in 1..Len(pending):
                /\ published' = pending[i]
                /\ regressed' = (regressed \/ pending[i] < published)
                /\ pending' = SubSeq(pending,1,i-1) \o SubSeq(pending,i+1,Len(pending))
           /\ UNCHANGED <<authoritative,captured>>
Next == Write \/ Capture \/ Publish
Spec == Init /\ [][Next]_vars
NoSnapshotRegression == ~regressed
=============================================================================
