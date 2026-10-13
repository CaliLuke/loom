------------------------ MODULE LocalPresence ------------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES depth, wrapper, anySource, element, nullable, phase, admitted
vars == <<depth, wrapper, anySource, element, nullable, phase, admitted>>
Init == /\ depth \in 0..2
        /\ wrapper \in BOOLEAN
        /\ (wrapper => depth = 1)
        /\ anySource \in BOOLEAN
        /\ element \in BOOLEAN
        /\ (element => depth > 0 /\ ~wrapper)
        /\ nullable \in BOOLEAN
        /\ phase = "declared"
        /\ admitted = FALSE
SupportedAny == anySource /\ ~element
Inspect == /\ phase = "declared"
           /\ admitted' = IF Mode = "shallow" /\ (depth > 1 \/ wrapper)
                           THEN TRUE
                           ELSE (~nullable \/
                              IF Mode = "all-any" THEN anySource ELSE SupportedAny)
           /\ phase' = "checked"
           /\ UNCHANGED <<depth, wrapper, anySource, element, nullable>>
Next == Inspect
Spec == Init /\ [][Next]_vars
LocalIntent == phase = "checked" /\ nullable /\ ~SupportedAny => ~admitted
SupportedValues == phase = "checked" /\ (~nullable \/ SupportedAny) => admitted
=====================================================================
