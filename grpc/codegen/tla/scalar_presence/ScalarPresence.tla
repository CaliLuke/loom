---------------- MODULE ScalarPresence ----------------
EXTENDS Integers
CONSTANT ExplicitPresence
VARIABLES supplied, value, required, defaulted
vars == <<supplied, value, required, defaulted>>
Init == /\ supplied \in BOOLEAN
        /\ value \in {0, 1}
        /\ required \in BOOLEAN
        /\ defaulted \in BOOLEAN
WirePresent == IF ExplicitPresence \/ ~required THEN supplied ELSE supplied /\ value # 0
Accepted == IF ExplicitPresence THEN ~required \/ WirePresent ELSE TRUE
Decoded == IF ~WirePresent /\ ~required /\ defaulted THEN 2 ELSE value
RequiredPresence == Accepted = (~required \/ supplied)
ExplicitZero == (Accepted /\ supplied) => Decoded = value
OptionalDefault == (~required /\ ~supplied /\ defaulted) => Decoded = 2
Next == UNCHANGED vars
Spec == Init /\ [][Next]_vars
========================================================
