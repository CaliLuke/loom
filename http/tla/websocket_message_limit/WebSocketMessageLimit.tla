--------------------- MODULE WebSocketMessageLimit ---------------------
EXTENDS Naturals
CONSTANT Design
VARIABLES phase, wireBytes, expandedBytes, accepted, released, code
vars == <<phase, wireBytes, expandedBytes, accepted, released, code>>
Limit == 2
Init == /\ phase = "read" /\ wireBytes \in 1..3
        /\ expandedBytes \in wireBytes..4
        /\ accepted = FALSE /\ released = FALSE /\ code = 0
Reject == wireBytes > Limit \/ (Design = "complete" /\ expandedBytes > Limit)
Read == /\ phase = "read" /\ phase' = "done"
        /\ accepted' = IF Design = "old" THEN TRUE ELSE ~Reject
        /\ released' = (Design = "complete" /\ Reject)
        /\ code' = IF Design # "old" /\ Reject THEN 1009 ELSE 0
        /\ UNCHANGED <<wireBytes, expandedBytes>>
Spec == Init /\ [][Read]_vars
ExpandedBound == accepted => expandedBytes <= Limit
OverflowReleased == phase = "done" /\ ~accepted => released /\ code = 1009
ValidAccepted == phase = "done" /\ expandedBytes <= Limit => accepted
=============================================================================
