------------------------ MODULE UnionPresence ------------------------
EXTENDS TLC
CONSTANT Policy
VARIABLES optional, outerNullable, nullBranches, input, result, phase
vars == <<optional, outerNullable, nullBranches, input, result, phase>>
Expected == CASE input = "absent" -> IF optional THEN "absent" ELSE "reject"
             [] input = "null" -> IF outerNullable THEN "outer-null"
                                  ELSE IF nullBranches = 1 THEN "selected-null" ELSE "reject"
             [] OTHER -> "selected-value"
Init == /\ optional \in BOOLEAN
        /\ outerNullable \in BOOLEAN
        /\ nullBranches \in {0, 1, 2}
        /\ input \in {"absent", "null", "value"}
        /\ result = "pending" /\ phase = "decode"
Decode == /\ phase = "decode"
          /\ result' = CASE Policy = "wrapper" /\ ~outerNullable /\ nullBranches = 1 /\ input = "null" -> "outer-null"
                           [] Policy = "pointer" /\ ~outerNullable /\ optional /\ input = "null" -> "absent"
                           [] Policy = "optional" /\ ~outerNullable /\ optional /\ input = "null" -> "reject"
                           [] OTHER -> Expected
          /\ phase' = "done"
          /\ UNCHANGED <<optional, outerNullable, nullBranches, input>>
Next == Decode \/ (phase = "done" /\ UNCHANGED vars)
Spec == Init /\ [][Next]_vars
Contract == phase = "done" => result = Expected
SelectedNull == phase = "done" /\ ~outerNullable /\ nullBranches = 1 /\ input = "null" => result = "selected-null"
=============================================================================
