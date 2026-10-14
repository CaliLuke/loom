--------------------------- MODULE ReadShutdown ---------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES phase, closing, joined, blackhole
vars == <<phase, closing, joined, blackhole>>
Init == /\ phase = "idle"
        /\ closing = FALSE
        /\ joined = FALSE
        /\ blackhole \in BOOLEAN
Arm == /\ phase = "idle"
       /\ ~closing
       /\ phase' = "read"
       /\ UNCHANGED <<closing, joined, blackhole>>
Stop == /\ ~closing
        /\ closing' = TRUE
        /\ UNCHANGED <<phase, joined, blackhole>>
Return == /\ phase = "read"
          /\ ((Mode = "socket-owner" /\ closing)
              \/ Mode = "bounded-io"
              \/ (Mode = "server-bound" /\ ~blackhole))
          /\ phase' = "returned"
          /\ UNCHANGED <<closing, joined, blackhole>>
Join == /\ closing /\ phase # "read" /\ ~joined
        /\ joined' = TRUE
        /\ UNCHANGED <<phase, closing, blackhole>>
Next == Arm \/ Stop \/ Return \/ Join
Spec == Init /\ [][Next]_vars /\ WF_vars(Stop) /\ WF_vars(Return) /\ WF_vars(Join)
TypeOK == /\ phase \in {"idle", "read", "returned"}
          /\ closing \in BOOLEAN /\ joined \in BOOLEAN /\ blackhole \in BOOLEAN
JoinedOwnsNoRead == joined => phase # "read"
Terminates == closing ~> joined
=============================================================================
