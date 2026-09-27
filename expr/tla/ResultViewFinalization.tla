---------------------- MODULE ResultViewFinalization ----------------------
EXTENDS Naturals, TLC
CONSTANT InheritBeforeView
VARIABLES explicit, canonical, view, stage
vars == <<explicit, canonical, view, stage>>
Init == /\ explicit \in BOOLEAN
        /\ canonical = {"right"}
        /\ view = IF explicit THEN {"right"} ELSE {}
        /\ stage = 0
Inherit == /\ canonical' = canonical \cup {"left"}
           /\ UNCHANGED view
Snapshot == /\ view' = IF explicit THEN view ELSE canonical
            /\ UNCHANGED canonical
Next == /\ stage < 2
        /\ IF (stage = 0) = InheritBeforeView THEN Inherit ELSE Snapshot
        /\ stage' = stage + 1
        /\ UNCHANGED explicit
Spec == Init /\ [][Next]_vars
ViewMatchesContract == stage = 2 =>
                       view = IF explicit THEN {"right"} ELSE canonical
=============================================================================
