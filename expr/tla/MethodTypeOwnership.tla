---------------------- MODULE MethodTypeOwnership ----------------------
EXTENDS FiniteSets, TLC
CONSTANT SeparateCustomizedType
VARIABLES customized, extended, pending, declarations
vars == <<customized, extended, pending, declarations>>
Uses == {"payload", "result"}
Keys == {"shared", "custom"}
Key(use) == IF SeparateCustomizedType /\ extended /\ use = customized
            THEN "custom" ELSE "shared"
Shape(use) == IF extended /\ use = customized
              THEN {"right", "left"} ELSE {"right"}
Init == /\ customized \in Uses
        /\ extended \in BOOLEAN
        /\ pending = Uses
        /\ declarations = [k \in Keys |-> {}]
Emit(use) == /\ use \in pending
             /\ declarations' = IF declarations[Key(use)] = {}
                                THEN [declarations EXCEPT ![Key(use)] = Shape(use)]
                                ELSE declarations
             /\ pending' = pending \ {use}
             /\ UNCHANGED <<customized, extended>>
Next == \E use \in Uses: Emit(use)
Spec == Init /\ [][Next]_vars
EveryUseHasItsOwnShape == pending = {} =>
                         \A use \in Uses: declarations[Key(use)] = Shape(use)
=============================================================================
