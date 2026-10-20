------------------------ MODULE NullAdmission ------------------------
EXTENDS TLC
CONSTANT Policy
VARIABLES known, allowsNull, input, admitted, evaluated
vars == <<known, allowsNull, input, admitted, evaluated>>
Init == /\ known \in BOOLEAN
        /\ allowsNull \in BOOLEAN
        /\ input \in {"missing", "null", "value"}
        /\ admitted = FALSE
        /\ evaluated = FALSE
Expected == known /\ input # "missing" /\ (input # "null" \/ allowsNull)
Decode == /\ ~evaluated
          /\ evaluated' = TRUE
          /\ admitted' = (known /\ input # "missing"
               /\ (CASE Policy = "old" -> input # "null"
                         [] Policy = "allow-all" -> TRUE
                         [] OTHER -> input # "null" \/ allowsNull))
          /\ UNCHANGED <<known, allowsNull, input>>
Spec == Init /\ [][Decode]_vars /\ WF_vars(Decode)
Contract == evaluated => (admitted = Expected)
RequiredValue == evaluated /\ input = "missing" => ~admitted
KnownBranch == evaluated /\ ~known => ~admitted
Completes == <>evaluated
======================================================================
