---------------------------- MODULE CLIInput ----------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES input, required, selected, physical, phase, result
vars == <<input, required, selected, physical, phase, result>>
Init == /\ input \in {"absent", "zero", "empty", "null", "value"}
        /\ required \in BOOLEAN
        /\ selected \in BOOLEAN
        /\ physical = "unset"
        /\ phase = "text"
        /\ result = "unset"
Decode == /\ phase = "text"
          /\ phase' = "decoded"
          /\ physical' = IF input = "absent" /\ Mode \in {"encoder", "old-zero"}
                            THEN "zero"
                         ELSE IF input = "absent" /\ Mode = "seed" /\ selected
                            THEN "default"
                         ELSE input
          /\ UNCHANGED <<input, required, selected, result>>
Admit == /\ phase = "decoded"
         /\ phase' = "done"
         /\ result' = IF required /\ physical = "absent" THEN "reject"
                      ELSE IF selected /\ (physical = "absent" \/ (Mode = "old-zero" /\ physical = "zero"))
                           THEN "default" ELSE physical
         /\ UNCHANGED <<input, required, selected, physical>>
Next == Decode \/ Admit
Spec == Init /\ [][Next]_vars
MissingRequired == phase = "done" /\ required /\ input = "absent" => result = "reject"
DefaultAbsent == phase = "done" /\ ~required /\ input = "absent" /\ selected => result = "default"
PreserveSupplied == phase = "done" /\ input # "absent" => result = input
=============================================================================
