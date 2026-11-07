--------------------------- MODULE FormBranch ---------------------------
EXTENDS FiniteSets
CONSTANT Policy
VARIABLES raw, selected, remaining, phase
vars == <<raw, selected, remaining, phase>>
UnionClaims == {"x", "y"}
Init == /\ raw \in SUBSET {"x", "y", "other", ""}
        /\ selected \in UnionClaims
        /\ selected \in raw
        /\ remaining = {}
        /\ phase = "project"
Project == /\ phase = "project"
           /\ remaining' = raw \ (IF Policy = "selected" THEN {selected} ELSE UnionClaims)
           /\ phase' = "done"
           /\ UNCHANGED <<raw, selected>>
Next == Project \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
ExactUnknownMembers == phase = "done" => remaining = raw \ UnionClaims
=============================================================================
