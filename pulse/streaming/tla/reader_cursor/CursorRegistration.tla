----------------------- MODULE CursorRegistration -----------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
Attempts == {"first", "second"}
VARIABLES phase, snapshot, registered, cursor, last, closing, rewound
vars == <<phase, snapshot, registered, cursor, last, closing, rewound>>
Init == /\ phase = [a \in Attempts |-> "idle"]
        /\ snapshot = [a \in Attempts |-> 0]
        /\ registered = FALSE /\ cursor = 0 /\ last = 0
        /\ closing = FALSE /\ rewound = FALSE
Prepare(a) == /\ phase[a] = "idle" /\ ~registered /\ ~closing
              /\ phase' = [phase EXCEPT ![a] = "prepared"]
              /\ snapshot' = [snapshot EXCEPT ![a] = last]
              /\ UNCHANGED <<registered, cursor, last, closing, rewound>>
Commit(a) == /\ phase[a] = "prepared" /\ ~closing
             /\ phase' = [phase EXCEPT ![a] = "done"]
             /\ IF ~registered \/ Mode = "unchecked-duplicate"
                THEN /\ registered' = TRUE /\ cursor' = snapshot[a]
                     /\ rewound' = (rewound \/ (registered /\ cursor > snapshot[a]))
                ELSE UNCHANGED <<registered, cursor, rewound>>
             /\ UNCHANGED <<snapshot, last, closing>>
Fail(a) == /\ phase[a] = "prepared"
           /\ phase' = [phase EXCEPT ![a] = "done"]
           /\ UNCHANGED <<snapshot, registered, cursor, last, closing, rewound>>
Produce == /\ last < 2 /\ last' = last + 1
           /\ UNCHANGED <<phase, snapshot, registered, cursor, closing, rewound>>
Poll == /\ registered /\ ~closing /\ cursor' = last
        /\ UNCHANGED <<phase, snapshot, registered, last, closing, rewound>>
Close == /\ ~closing /\ closing' = TRUE
         /\ UNCHANGED <<phase, snapshot, registered, cursor, last, rewound>>
Next == Produce \/ Poll \/ Close \/ \E a \in Attempts: Prepare(a) \/ Commit(a) \/ Fail(a)
Spec == Init /\ [][Next]_vars
TypeOK == /\ phase \in [Attempts -> {"idle", "prepared", "done"}]
          /\ snapshot \in [Attempts -> 0..2]
          /\ cursor \in 0..2 /\ last \in 0..2
          /\ registered \in BOOLEAN /\ closing \in BOOLEAN /\ rewound \in BOOLEAN
NoDuplicateRewind == ~rewound
=============================================================================
