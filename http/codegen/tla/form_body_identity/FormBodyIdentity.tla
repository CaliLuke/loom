------------------------- MODULE FormBodyIdentity -------------------------
EXTENDS Naturals
CONSTANT Policy
VARIABLES body, fields, tagged, mode, phase
vars == <<body, fields, tagged, mode, phase>>
Init == /\ body \in {"object", "union"}
        /\ fields \in {1, 2}
        /\ tagged \in BOOLEAN
        /\ mode = "unresolved" /\ phase = "prepare"
Prepare == /\ phase = "prepare"
           /\ mode' = IF body = "union" \/ (Policy = "shape" /\ fields = 1)
                       THEN "root-union" ELSE "named-fields"
           /\ phase' = "done"
           /\ UNCHANGED <<body, fields, tagged>>
Next == Prepare \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
AuthoredShape == phase = "done" =>
  ((body = "object") = (mode = "named-fields"))
=============================================================================
