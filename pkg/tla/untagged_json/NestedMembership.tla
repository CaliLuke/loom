---------------------- MODULE NestedMembership ----------------------
EXTENDS Naturals, FiniteSets, Sequences
CONSTANT Mode
VARIABLES rows, phase, matched
vars == <<rows, phase, matched>>
Candidates == {1, 2}
Init == /\ rows \in UNION { [1..n -> SUBSET Candidates] : n \in 0..2 }
        /\ phase = "input"
        /\ matched = FALSE
Expected == \A i \in DOMAIN rows : Cardinality(rows[i]) = 1
RowMatch(set) == IF Mode = "any" THEN Cardinality(set) > 0
                 ELSE Cardinality(set) = 1
Match == /\ phase = "input"
         /\ matched' = \A i \in DOMAIN rows : RowMatch(rows[i])
         /\ phase' = "matched"
         /\ UNCHANGED rows
Next == Match
Spec == Init /\ [][Next]_vars
Agreement == phase = "matched" => matched = Expected
=====================================================================
