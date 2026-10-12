-------------------------- MODULE CarrierLocation --------------------------
EXTENDS Naturals
CONSTANT Mode
Carriers == {"root", "child", "recursive"}
VARIABLES phase, canonical, projected, snapshot, emitted
vars == <<phase, canonical, projected, snapshot, emitted>>
Init == /\ phase = 0
        /\ canonical = [c \in Carriers |-> "custom"]
        /\ projected = canonical
        /\ snapshot = [c \in Carriers |-> "unset"]
        /\ emitted = [c \in Carriers |-> "unset"]
First == /\ phase = 0 /\ phase' = 1
         /\ projected' = IF Mode = "checked" THEN [c \in Carriers |-> "views"] ELSE projected
         /\ snapshot' = IF Mode = "legacy" THEN projected ELSE snapshot
         /\ UNCHANGED <<canonical, emitted>>
Second == /\ phase = 1 /\ phase' = 2
          /\ projected' = [c \in Carriers |-> "views"]
          /\ snapshot' = IF Mode = "checked" THEN projected ELSE snapshot
          /\ UNCHANGED <<canonical, emitted>>
Emit == /\ phase = 2 /\ phase' = 3 /\ emitted' = snapshot
        /\ UNCHANGED <<canonical, projected, snapshot>>
Next == First \/ Second \/ Emit
        \/ (phase = 3 /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars /\ WF_vars(Next)
CanonicalLocationPreserved == \A c \in Carriers: canonical[c] = "custom"
ReferencesNameDeclaration == phase = 3 => \A c \in Carriers: emitted[c] = projected[c]
Terminates == <>(phase = 3)
=============================================================================
