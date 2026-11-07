-------------------- MODULE ClientPresence --------------------
EXTENDS Naturals
CONSTANT CheckFirst, CheckKind
VARIABLE carrier, value, required, stage, outcome
vars == <<carrier, value, required, stage, outcome>>
Init == /\ carrier \in {"pointer", "presence", "union"}
        /\ value \in {"absent", "null", "concrete"}
        /\ (carrier = "pointer" => value # "null")
        /\ required \in BOOLEAN
        /\ stage = "start" /\ outcome = "none"
Admit == IF value = "absent" THEN IF required THEN "missing" ELSE "no-body" ELSE "encoded"
Next == /\ stage = "start"
        /\ outcome' = IF CheckFirst /\ (carrier # "union" \/ CheckKind) THEN Admit
                       ELSE IF carrier \in {"pointer", "union"} /\ value = "absent" THEN "panic"
                       ELSE "encoded"
        /\ stage' = "done"
        /\ UNCHANGED <<carrier, value, required>>
Spec == Init /\ [][Next]_vars
SafePresence == stage = "done" => outcome = Admit
=============================================================================
