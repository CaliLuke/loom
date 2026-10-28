------------------------- MODULE Concurrency -------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Requests, MaxCapacity, Mode
VARIABLES capacity, running, owned, admitted, finished, rejected, canceled,
          safeAdmission
vars == <<capacity, running, owned, admitted, finished, rejected, canceled,
          safeAdmission>>
Init == /\ capacity = 1
        /\ running = {}
        /\ owned = {}
        /\ admitted = {}
        /\ finished = {}
        /\ rejected = {}
        /\ canceled = {}
        /\ safeAdmission = TRUE
Pending == Requests \ (admitted \cup rejected)
Admit(r) == /\ r \in Pending
            /\ r \notin canceled
            /\ Cardinality(owned) < capacity
            /\ safeAdmission' = (Cardinality(running) < capacity)
            /\ running' = running \cup {r}
            /\ owned' = owned \cup {r}
            /\ admitted' = admitted \cup {r}
            /\ UNCHANGED <<capacity, finished, rejected, canceled>>
Reject(r) == /\ r \in Pending
             /\ r \in canceled \/ Cardinality(owned) >= capacity
             /\ rejected' = rejected \cup {r}
             /\ UNCHANGED <<capacity, running, owned, admitted, finished,
                            canceled, safeAdmission>>
Cancel(r) == /\ r \notin canceled
             /\ canceled' = canceled \cup {r}
             /\ owned' = IF Mode = "release-on-cancel" THEN owned \ {r}
                         ELSE owned
             /\ UNCHANGED <<capacity, running, admitted, finished, rejected,
                            safeAdmission>>
Finish(r) == /\ r \in running
             /\ running' = running \ {r}
             /\ owned' = owned \ {r}
             /\ finished' = finished \cup {r}
             /\ UNCHANGED <<capacity, admitted, rejected, canceled, safeAdmission>>
Resize(n) == /\ capacity' = IF n = 0 THEN capacity ELSE n
             /\ owned' = IF Mode = "reset-on-resize" /\ n # 0 THEN {} ELSE owned
             /\ UNCHANGED <<running, admitted, finished, rejected, canceled,
                            safeAdmission>>
Next == (\E r \in Requests : Admit(r) \/ Reject(r) \/ Cancel(r) \/ Finish(r))
        \/ (\E n \in 0..MaxCapacity : Resize(n))
Spec == Init /\ [][Next]_vars
PermitLifetime == owned = running
Accounting == /\ admitted = running \cup finished
              /\ running \intersect finished = {}
              /\ admitted \intersect rejected = {}
AdmissionBound == safeAdmission
ValidCapacity == capacity \in 1..MaxCapacity
======================================================================
