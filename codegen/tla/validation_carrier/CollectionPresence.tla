--------------------- MODULE CollectionPresence ---------------------
EXTENDS Naturals
CONSTANT Mode
VARIABLES required, supplied, size, nilIsEmpty, phase, valid
vars == <<required, supplied, size, nilIsEmpty, phase, valid>>
Init == /\ required \in BOOLEAN
        /\ supplied \in BOOLEAN
        /\ size \in 0..5
        /\ nilIsEmpty \in BOOLEAN
        /\ (nilIsEmpty => supplied)
        /\ (~supplied => size = 0)
        /\ phase = "input"
        /\ valid = FALSE
WithinBounds == size >= 2 /\ size <= 4
Validate == /\ phase = "input"
            /\ valid' = CASE Mode = "legacy" -> WithinBounds
                         [] Mode = "native-only" ->
                              (~required /\ (~supplied \/ (nilIsEmpty /\ size = 0))) \/ WithinBounds
                         [] OTHER -> (~required /\ ~supplied) \/ WithinBounds
            /\ phase' = "validated"
            /\ UNCHANGED <<required, supplied, size, nilIsEmpty>>
Next == Validate
Spec == Init /\ [][Next]_vars
Omission == phase = "validated" /\ ~required /\ ~supplied => valid
Bounds == phase = "validated" /\ (required \/ supplied) => (valid = WithinBounds)
=====================================================================
