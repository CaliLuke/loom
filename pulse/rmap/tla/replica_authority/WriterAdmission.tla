------------------------- MODULE WriterAdmission -------------------------
EXTENDS Naturals, TLC
CONSTANT Policy
VARIABLES valid, phase, sideEffects, rejected
vars == <<valid,phase,sideEffects,rejected>>
Init == /\ valid \in [1..2 -> BOOLEAN] /\ phase = 0
        /\ sideEffects = {} /\ rejected = FALSE
Preflight == /\ phase = 0
             /\ phase' = IF Policy = "Shared" /\ ~\A k \in 1..2: valid[k]
                          THEN 4 ELSE 1
             /\ rejected' = (phase' = 4)
             /\ UNCHANGED <<valid,sideEffects>>
SideEffect == /\ phase = 1 /\ phase' = 2
              /\ sideEffects' = sideEffects \cup {"stream-or-owner"}
              /\ UNCHANGED <<valid,rejected>>
FirstMap == /\ phase = 2
            /\ phase' = IF valid[1] THEN 3 ELSE 4
            /\ sideEffects' = IF valid[1] THEN sideEffects \cup {"map-one"} ELSE sideEffects
            /\ rejected' = ~valid[1]
            /\ UNCHANGED valid
SecondMap == /\ phase = 3 /\ phase' = 4
             /\ sideEffects' = IF valid[2] THEN sideEffects \cup {"map-two"} ELSE sideEffects
             /\ rejected' = ~valid[2]
             /\ UNCHANGED valid
Next == Preflight \/ SideEffect \/ FirstMap \/ SecondMap
Spec == Init /\ [][Next]_vars
RejectedBeforeEffects == rejected => sideEffects = {}
=============================================================================
