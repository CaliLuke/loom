------------------------ MODULE NullableEnum ------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES nullable, inherited, required, wire, phase, matched
vars == <<nullable, inherited, required, wire, phase, matched>>
Init == /\ nullable \in BOOLEAN
        /\ inherited \in BOOLEAN
        /\ required \in BOOLEAN
        /\ wire \in {"missing", "null", "member", "other"}
        /\ phase = "input"
        /\ matched = FALSE
Expected == CASE wire = "missing" -> ~required
              [] wire = "null" -> nullable \/ inherited
              [] OTHER -> wire = "member"
BodyAdmits == wire \in {"member", "other"} \/
              (wire = "null" /\ (nullable \/ inherited))
NullAdmits == CASE Mode = "legacy" -> FALSE
               [] Mode = "local-only" -> nullable
               [] OTHER -> nullable \/ inherited
ValueGate == (wire = "null" /\ NullAdmits) \/ wire = "member"
Match == /\ phase = "input"
         /\ matched' = IF wire = "missing" THEN ~required
                       ELSE BodyAdmits /\ ValueGate
         /\ phase' = "matched"
         /\ UNCHANGED <<nullable, inherited, required, wire>>
Next == Match
Spec == Init /\ [][Next]_vars
Agreement == phase = "matched" => matched = Expected
=====================================================================
