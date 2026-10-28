------------------------ MODULE NestedFailure ------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES innerFailure, phase, outcome
vars == <<innerFailure, phase, outcome>>
Init == /\ innerFailure \in {"mismatch", "unexpected"}
        /\ phase = "input"
        /\ outcome = "unset"
Expected == IF innerFailure = "mismatch" THEN "other branch" ELSE "error"
Decode == /\ phase = "input"
          /\ outcome' = IF Mode = "unclassified" THEN "error" ELSE Expected
          /\ phase' = "decoded"
          /\ UNCHANGED innerFailure
Next == Decode
Spec == Init /\ [][Next]_vars
Agreement == phase = "decoded" => outcome = Expected
=====================================================================
