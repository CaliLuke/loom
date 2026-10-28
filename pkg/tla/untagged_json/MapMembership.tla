------------------------ MODULE MapMembership ------------------------
EXTENDS Naturals, FiniteSets
CONSTANT Mode
VARIABLES present, nullable, required, wire, entries, minimum, phase, matched
vars == <<present, nullable, required, wire, entries, minimum, phase, matched>>
Init == /\ present \in BOOLEAN
        /\ nullable \in BOOLEAN
        /\ required \in BOOLEAN
        /\ wire \in {"null", "map", "array"}
        /\ entries \in SUBSET {"valid", "invalid"}
        /\ minimum \in 0..1
        /\ phase = "input"
        /\ matched = FALSE
EveryValue == "invalid" \notin entries
LengthOK == Cardinality(entries) >= minimum
Expected == IF ~present THEN ~required
            ELSE IF wire = "null" THEN nullable
            ELSE wire = "map" /\ LengthOK /\ EveryValue
ValueGate == IF Mode = "existential" THEN "valid" \in entries
             ELSE EveryValue
Match == /\ phase = "input"
         /\ matched' = IF ~present THEN ~required
                       ELSE IF wire = "null" THEN nullable
                       ELSE wire = "map" /\ LengthOK /\ ValueGate
         /\ phase' = "matched"
         /\ UNCHANGED <<present, nullable, required, wire, entries, minimum>>
Next == Match
Spec == Init /\ [][Next]_vars
Agreement == phase = "matched" => matched = Expected
=====================================================================
