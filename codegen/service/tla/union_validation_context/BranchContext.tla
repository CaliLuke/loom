--------------------------- MODULE BranchContext ---------------------------
EXTENDS TLC
CONSTANT Mode
VARIABLES projected, required, hasDefault, supplied, valid, phase, typed, panic, accepted
vars == <<projected, required, hasDefault, supplied, valid, phase, typed, panic, accepted>>
StoredPointer == projected \/ (~required /\ ~hasDefault)
Init == /\ projected \in BOOLEAN
        /\ required \in BOOLEAN
        /\ hasDefault \in BOOLEAN
        /\ supplied \in BOOLEAN
        /\ valid \in BOOLEAN
        /\ (~StoredPointer => supplied)
        /\ phase = "pending"
        /\ typed = TRUE /\ panic = FALSE /\ accepted = FALSE
Validate == /\ phase = "pending"
            /\ LET usesPointer == IF Mode = "legacy" THEN ~required /\ ~hasDefault
                                   ELSE StoredPointer
                   guardsRead == IF Mode = "dereference-only" THEN ~required
                                 ELSE usesPointer
               IN /\ typed' = (usesPointer = StoredPointer)
                  /\ panic' = (usesPointer /\ ~supplied /\ ~guardsRead)
                  /\ accepted' = (typed' /\ ~panic' /\ ((supplied /\ valid) \/ (~supplied /\ ~required)))
            /\ phase' = "done"
            /\ UNCHANGED <<projected, required, hasDefault, supplied, valid>>
Next == Validate
Spec == Init /\ [][Next]_vars
DeclarationAgreement == typed
NoAbsentRead == ~panic
Contract == phase = "done" => accepted = ((supplied /\ valid) \/ (~supplied /\ ~required))
=============================================================================
