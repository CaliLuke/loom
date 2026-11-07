------------------------ MODULE MismatchProjection ------------------------
EXTENDS FiniteSets
CONSTANT Policy
VARIABLES raw, closed, schemaValue, typedValue, retained, decoded, phase
vars == <<raw, closed, schemaValue, typedValue, retained, decoded, phase>>
Claims == {"x"}
Init == /\ raw \in SUBSET {"x", "unknown", ""}
        /\ closed \in BOOLEAN
        /\ schemaValue \in BOOLEAN
        /\ typedValue \in BOOLEAN
        /\ retained = {} /\ decoded = FALSE /\ phase = "project"
Project == /\ phase = "project"
           /\ retained' = IF schemaValue \/ Policy = "retain" THEN raw ELSE {}
           /\ phase' = "decode"
           /\ UNCHANGED <<raw, closed, schemaValue, typedValue, decoded>>
Decode == /\ phase = "decode"
          /\ decoded' = (typedValue /\ (~closed \/ retained \ Claims = {}))
          /\ phase' = "done"
          /\ UNCHANGED <<raw, closed, schemaValue, typedValue, retained>>
Next == Project \/ Decode \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
IndependentTypedClosure == phase = "done" =>
    decoded = (typedValue /\ (~closed \/ raw \ Claims = {}))
=============================================================================
