------------------------- MODULE JSONFieldSelection -------------------------
EXTENDS Naturals
CONSTANT CapturedNullability
VARIABLES tag, nullable, input, phase, accepted
vars == <<tag, nullable, input, phase, accepted>>
Init == /\ tag \in BOOLEAN /\ nullable \in BOOLEAN
        /\ input \in {"missing", "null", "value"}
        /\ phase = "ready" /\ accepted = FALSE
Next == /\ phase = "ready" /\ phase' = "done"
        /\ accepted' = (tag /\ input # "missing" /\
             (input # "null" \/ (CapturedNullability /\ nullable)))
        /\ UNCHANGED <<tag, nullable, input>>
Expected == tag /\ input # "missing" /\ (input # "null" \/ nullable)
Agreement == phase = "done" => accepted = Expected
Spec == Init /\ [][Next]_vars
=============================================================================
