------------------------- MODULE TaggedFormProjection ------------------------
EXTENDS Naturals, FiniteSets
CONSTANTS Flatten, ShiftBounds
VARIABLES payload, closed, required, minimum, maximum, phase
vars == <<payload, closed, required, minimum, maximum, phase>>
Fields == {"code", "extra"}
Wire == payload \cup {"tag"}
Init == /\ payload \in SUBSET Fields
        /\ closed \in BOOLEAN
        /\ required \in BOOLEAN
        /\ minimum \in 0..2
        /\ maximum \in minimum..2
        /\ phase = "ready"
Next == /\ phase = "ready" /\ phase' = "checked"
        /\ UNCHANGED <<payload, closed, required, minimum, maximum>>
Accepted == /\ (~closed \/ payload \subseteq {"code"})
            /\ (~required \/ "code" \in payload)
            /\ Cardinality(payload) >= minimum
            /\ Cardinality(payload) <= maximum
Exported == IF Flatten
  THEN /\ (~closed \/ Wire \subseteq {"code", "tag"})
       /\ (~required \/ "code" \in Wire)
       /\ "tag" \in Wire
       /\ Cardinality(Wire) >= minimum + (IF ShiftBounds THEN 1 ELSE 0)
       /\ Cardinality(Wire) <= maximum + (IF ShiftBounds THEN 1 ELSE 0)
  ELSE Wire = {"tag", "value"}
Agreement == phase = "checked" => Accepted = Exported
Spec == Init /\ [][Next]_vars
=============================================================================
