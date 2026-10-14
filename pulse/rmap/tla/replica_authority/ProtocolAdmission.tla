------------------------- MODULE ProtocolAdmission -------------------------
EXTENDS Naturals, TLC
CONSTANT Policy
VARIABLES marker, oldActive, newActive
vars == <<marker, oldActive, newActive>>
Init == /\ marker \in {"absent", "3", "4", "2", "future", "malformed"}
        /\ oldActive = FALSE /\ newActive = FALSE
OldJoin == /\ ~oldActive /\ marker \in {"absent", "3"}
           /\ marker' = "3" /\ oldActive' = TRUE /\ UNCHANGED newActive
NewJoin == /\ ~newActive /\ ~oldActive
           /\ marker \in {"absent", "3", "4"}
           /\ marker' = IF Policy = "SharedMarker" THEN "3" ELSE "4"
           /\ newActive' = TRUE /\ UNCHANGED oldActive
Next == OldJoin \/ NewJoin
Spec == Init /\ [][Next]_vars
ExclusiveAdmission == ~(oldActive /\ newActive)
UnknownUntouched == marker \in {"2", "future", "malformed"} => ~oldActive /\ ~newActive
=============================================================================
