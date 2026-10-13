------------------------- MODULE BranchCarrier -------------------------
EXTENDS Naturals
CONSTANTS Mode, Focus
VARIABLES carrier, value, minimum, phase, guard, outcome, predicates
vars == <<carrier, value, minimum, phase, guard, outcome, predicates>>
Collections == {"list", "map"}
Pointers == {"object", "nested"}
Carriers == Collections \cup Pointers \cup {"nullable", "primitive"}
Values(c) == IF c = "nullable" THEN {"absent", "null", "valid", "invalid"}
             ELSE IF c \in Collections THEN {"nil", "empty", "valid"}
             ELSE IF c \in Pointers THEN {"nil", "valid", "invalid"}
             ELSE {"valid", "invalid"}
Init == /\ carrier \in (IF Focus = "collections" THEN Collections ELSE Carriers)
        /\ predicates \in (IF Mode \in {"physical", "skip-empty"} /\ carrier \in Pointers
                           THEN {TRUE, FALSE} ELSE {TRUE})
        /\ value \in (IF predicates THEN Values(carrier) ELSE {"nil", "valid"})
        /\ minimum \in (IF carrier \in Collections THEN {0, 1} ELSE {0})
        /\ phase = "select"
        /\ guard = FALSE
        /\ outcome = "pending"
Select == /\ phase = "select"
          /\ guard' = IF Mode = "old" THEN carrier \in Collections \cup {"object"}
                      ELSE carrier \in Pointers
          /\ phase' = "invoke"
          /\ UNCHANGED <<carrier, value, minimum, outcome, predicates>>
Predicate == IF value = "invalid" THEN "error"
             ELSE IF carrier \in Collections /\ value \in {"nil", "empty"} /\ minimum = 1
                  THEN "error"
             ELSE "ok"
Invoke == /\ phase = "invoke"
          /\ outcome' = IF Mode = "skip-empty" /\ ~predicates THEN "ok"
                        ELSE IF guard /\ value = "nil" THEN "error"
                        ELSE IF carrier \in Pointers /\ value = "nil" THEN "panic"
                        ELSE IF carrier = "nullable" /\ value = "absent" THEN "error"
                        ELSE IF carrier \in Collections /\ value = "nil" /\ Mode \in {"old", "guard-only"}
                             THEN "ok" \* Existing named call skips nil unless value mode is set.
                        ELSE Predicate
          /\ phase' = "done"
          /\ UNCHANGED <<carrier, value, minimum, guard, predicates>>
Expected == IF carrier \in Pointers /\ value = "nil" THEN "error"
            ELSE IF carrier = "nullable" /\ value = "absent" THEN "error"
            ELSE Predicate
Next == Select \/ Invoke
Spec == Init /\ [][Next]_vars
NoPanic == outcome # "panic"
Contract == phase = "done" => outcome = Expected
=============================================================================
