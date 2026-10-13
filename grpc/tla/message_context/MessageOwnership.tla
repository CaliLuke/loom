-------------------------- MODULE MessageOwnership --------------------------
EXTENDS Integers, TLC
CONSTANT Mode
VARIABLES input, snapshot, canceled, wire
vars == <<input, snapshot, canceled, wire>>
Init == /\ input = 0 /\ snapshot = 0 /\ canceled = FALSE /\ wire = -1
Cancel == /\ ~canceled /\ canceled' = TRUE /\ UNCHANGED <<input, snapshot, wire>>
ReuseInput == /\ canceled /\ input = 0 /\ input' = 1
              /\ UNCHANGED <<snapshot, canceled, wire>>
NativeSend == /\ wire = -1
              /\ wire' = IF Mode = "detached" THEN snapshot ELSE input
              /\ UNCHANGED <<input, snapshot, canceled>>
Next == Cancel \/ ReuseInput \/ NativeSend
Spec == Init /\ [][Next]_vars
SnapshotPreserved == wire \in {-1, 0}
=============================================================================
